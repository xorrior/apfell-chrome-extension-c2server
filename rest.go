package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func apfellRequest(endpoint string, data []byte, method string) []byte {
	url := fmt.Sprintf("%s%s", cf.ApfellBaseUrl, endpoint)
	log.Printf("Making request to %s", url)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	var respbody []byte

	if data == nil {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			log.Println("Error: ", err)
			ravenlog("Unable to complete apfell web request")
			return make([]byte, 0)
		}

		resp, err := client.Do(req)

		if err != nil {
			ravenlog(fmt.Sprintf("Error when sending request to the server: %s", err))
			return make([]byte, 0)
		}

		defer resp.Body.Close()
		respbody, _ = ioutil.ReadAll(resp.Body)
	} else {
		req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
		if err != nil {
			log.Println("Unable to create apfell request")
			return make([]byte, 0)
		}

		resp, err := client.Do(req)

		if err != nil {
			log.Println("Error when sending request to the server:", err)
			return make([]byte, 0)
		}

		defer resp.Body.Close()
		respbody, _ = ioutil.ReadAll(resp.Body)
	}

	return respbody
}
