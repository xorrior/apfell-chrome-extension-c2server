package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var jsonMap map[string]json.RawMessage

type taskResponse struct {
	Response string `json:"response"`
}

type simpleTaskResponse struct {
	Status string `json:"status"`
}

type initialFileResponse struct {
	TotalChunks int `json:"total_chunks"`
	TaskID      int `json:"task"`
}

type apfellFileDownloadMetaData struct {
	Chunks int    `json:"total_chunks"`
	TaskID string `json:"task"`
}

type apfellFileDownloadMeta struct {
	Response apfellFileDownloadMetaData `json:"response"`
}

type apfellFileChunkMeta struct {
	Response string `json:"response"`
}

// {"status": "success", "chunk": 2}

//  {"status":"success","active":true,"integrity_level":2,"init_callback":"11\/14\/2018 14:25:13","last_checkin":"11\/14\/2018 14:25:13","user":"m0rty","host":"macos-extension-dev.shared","pid":10527,"ip":"127.0.0.1","description":"debug","operator":"apfell_admin","registered_payload":"4540d5aabdde6e3747d68bba8e2ee36f0e516e9ddae43d93dca26f38c0a7cba6","payload_type":"apfell-macho","c2_profile":"slack","pcallback":"null","operation":"default","id":69,"encryption_type":"AES_SYM"}
type standardApfellCheckinResponse struct {
	Status         string `json:"status"`
	Active         bool   `json:"active"`
	IntegrityLevel int    `json:"integrity_level"`
	InitCallback   string `json:"init_callback"`
	LastCheckin    string `json:"last_checkin"`
	User           string `json:"user"`
	Host           string `json:"host"`
	Pid            int    `json:"pid"`
	IP             string `json:"ip"`
	Description    string `json:"description"`
	Operator       string `json:"operator"`
	Payload        string `json:"registered_payload"`
	PayloadType    string `json:"payload_type"`
	C2profile      string `json:"c2_profile"`
	PCallback      string `json:"pcallback"`
	Operation      string `json:"operation"`
	ID             string `json:"id"`
	EncryptionType string `json:"encryption_type"`
}

type standardApfellResponse struct {
	Status     string             `json:"status"`
	Timestamp  string             `json:"timestamp"`
	Task       nestedTaskResponse `json:"task"`
	Response   string             `json:"response"`
	ResponseID string             `json:"id"`
	FileID     string             `json:"file_id"`
}

type nestedTaskResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Command   string `json:"command"`
	Params    string `json:"params"`
	AttackID  int    `json:"attack_id"`
	Callback  int    `json:"callback"`
	Operator  string `json:"operator"`
}

type fileChunkMetaData struct {
	ChunkNum  int    `json:"chunk_num"`
	ChunkData string `json:"chunk_data"`
	FileID    string `json:"file_id"`
}

type fileChunkMeta struct {
	Response fileChunkMetaData `json:"response"`
}

type checkInStruct struct {
	User    string `json:"user"`
	Host    string `json:"host"`
	Pid     int    `json:"pid"`
	IP      string `json:"ip"`
	UUID    string `json:"uuid"`
	EncType string `json:"encryption_type"`
	DecKey  string `json:"decryption_key"`
	EncKey  string `json:"encryption_key"`
}

type task struct {
	Command string `json:"command"`
	Params  string `json:"params"`
	TaskID  string `json:"id"`
}

func apfellRequest(endpoint string, data []byte, method string) []byte {
	url := fmt.Sprintf("%s%s", cf.ApfellBaseUrl, endpoint)
	log.Printf("Making request to %s", url)
	client := &http.Client{}
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
