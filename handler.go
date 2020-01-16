package main

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var conn net.Conn
var upgrader = websocket.Upgrader{}

type message struct {
	Client bool   `json:"client"`
	Data   string `json:"data"`
	Tag    string `json:"tag"`
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade to a websocket connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ravenlog(fmt.Sprintf("Websocket connection/upgrade failed: %s", err))
		http.Error(w, "Websocket connection failed", http.StatusBadRequest)
		return
	}

	c := make(chan interface{})
	clID := String(8)

	ravenlog("Handling new websocket connection")
	go manageClient(conn, clID, c)

}

func manageClient(c *websocket.Conn, clid string, newtask chan interface{}) {
	defer func() { _ = c.Close() }()

	for {
		newMsg := message{}
		ravenlog("Reading message from client")

		err := c.ReadJSON(&newMsg)

		if err != nil {
			ravenlog(fmt.Sprintf("error: %v", err))
			break
		}

		// Check if the interface is empty

		if len(newMsg.Data) != 0 {
			decoded, _ := base64.StdEncoding.DecodeString(newMsg.Data)
			ravenlog(fmt.Sprintf("Received message from client %s\n", string(decoded)))
			resp := apfellRequest("agent_message", []byte(newMsg.Data), "POST")

			if len(resp) != 0 {
				ravenlog(fmt.Sprintf("Received apfell response: %s\n", string(resp)))
				response := message{}
				response.Data = string(resp)
				response.Client = false
				response.Tag = ""

				err = c.WriteJSON(response)
				if err != nil {
					ravenlog(fmt.Sprintf("Unable to send response to client %s", err))
					break
				}
			}
		}

		// return the data to the client

		time.Sleep(time.Duration(cf.Interval) * time.Second)
	}
}
