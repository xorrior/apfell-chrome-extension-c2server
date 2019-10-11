package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/kabukky/httpscerts"
)

var logger *log.Logger

type configuration struct {
	Debug          bool   `json:"debug"`
	Ssl            bool   `json:"ssl"`
	ApfellBaseUrl  string `json:"apfellbaseurl"`
	Bindaddress    string `json:"bindaddress"`
	Interval       int    `json:"interval"`
	DefaultPage    string `json:"defaultpage"`
	CheckinRetries int    `json:"checkinretries"`
}

var cf configuration

type token struct {
	Token string `json:"token"`
}

func serveDefaultPage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request ", r.URL)

	if r.URL.Path == "/" && r.Method == "GET" {
		// Serve the default page if we receive a GET request at the base URI
		http.ServeFile(w, r, cf.DefaultPage)
		return
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
}

func ravenlog(msg string) {
	if logger != nil {
		logger.Println(msg)
	}
}

func main() {
	// Main function responsible
	// Read the c2 profile config from the json file
	configFile, err := os.Open("c2config.json")

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	configBytes, _ := ioutil.ReadAll(configFile)
	json.Unmarshal(configBytes, &cf)

	// Check to make sure the apfell server flag was used
	if cf.ApfellBaseUrl == "" {
		log.Println("Apfell baseurl value empty")
		os.Exit(1)
	}

	if cf.Debug == true {
		logger = log.New(os.Stdout, "server: ", log.Lshortfile|log.LstdFlags)
	}

	http.HandleFunc("/", serveDefaultPage)
	http.HandleFunc("/websocket", socketHandler)

	if cf.Ssl == true {
		err := httpscerts.Check("cert.pem", "key.pem")
		if err != nil {
			ravenlog("cert.pem and key.pem not found. Generating SSL pem and private key.")
			err = httpscerts.Generate("cert.pem", "key.pem", cf.Bindaddress)
			if err != nil {
				log.Fatal("Error generating https cert")
				os.Exit(1)
			}
		}

		ravenlog(fmt.Sprintf("Starting SSL Web and Websockets server at https://%s and wss://%s", cf.Bindaddress, cf.Bindaddress))
		err = http.ListenAndServeTLS(cf.Bindaddress, "cert.pem", "key.pem", nil)
		if err != nil {
			log.Fatal("Failed to start raven server: ", err)
		}

	} else {
		ravenlog(fmt.Sprintf("Starting Web and Websockets server at http://%s and ws://%s", cf.Bindaddress, cf.Bindaddress))
		err := http.ListenAndServe(cf.Bindaddress, nil)
		if err != nil {
			log.Fatal("Failed to start raven server: ", err)
		}
	}
}
