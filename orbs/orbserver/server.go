// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package orbserver

import (
	"fmt"
	"net/http"
	"log"
	"strconv"
	"strings"
	"regexp"
	"errors"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

var (
	maxID = 512
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	isOkName = regexp.MustCompile("^[A-Za-z0-9]+$").MatchString
	delimstr = "\uffff"
	delimrune = '\uffff'
)

type ConnInfo struct {
	Connect *websocket.Conn
	Ip string
}

type HubController struct {
	hubs []*Hub
	authUUID string
}

func (h *HubController) addHub(roomName string, spriteNames, systemNames []string) {
	hub := NewHub(roomName, spriteNames, systemNames, h)
	h.hubs = append(h.hubs, hub)
	go hub.Run()
}

func (h *HubController) auth(inpUUID string) bool {
	return inpUUID == h.authUUID
}

func (h *HubController) globalBroadcast(inpData []byte) {
	for _, hub := range h.hubs {
		hub.broadcast(inpData)
	}
}


// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// True if the id is in use
	id map[int]bool

	// Inbound messages from the clients.
	processMsgCh chan *Message

	// Connection requests from the clients.
	connect chan *ConnInfo

	// Unregister requests from clients.
	unregister chan *Client

	roomName string
	//list of valid game character sprite resource keys
	spriteNames []string
	systemNames []string

	controller *HubController
}

func writeLog(ip string, payload string, errorcode int) {
	log.Printf("%v \"%v\" %v\n", ip, strings.Replace(payload, "\"", "'", -1), errorcode)
}

func writeErrLog(ip string, payload string) {
	writeLog(ip, payload, 400)
}

func CreateAllHubs(roomNames, spriteNames, systemNames []string, authUUID string) {
	h := HubController{
		authUUID: authUUID,
	}

	for _, roomName := range roomNames {
		h.addHub(roomName, spriteNames, systemNames)
	}
}


func NewHub(roomName string, spriteNames []string, systemNames []string, h *HubController) *Hub {
	return &Hub{
		processMsgCh:  make(chan *Message),
		connect:   make(chan *ConnInfo),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		id: make(map[int]bool),
		roomName: roomName,
		spriteNames: spriteNames,
		systemNames: systemNames,
		controller: h,
	}
}

func (h *Hub) Run() {
	http.HandleFunc("/" + h.roomName, h.serveWs)
	for {
		select {
		case conn := <-h.connect:
			id := 0
			for i := 0; i <= maxID; i++ {
				if !h.id[i] {
					id = i
					break
				}
			}

			ip_limit := 2
			same_ip := 0
			for other_client := range h.clients {
				log.Println(other_client.ip, conn.Ip)
				if other_client.ip == conn.Ip {
					same_ip++
				}
				if same_ip >= ip_limit {
					writeErrLog(conn.Ip, "max client in room exceeded")
					break
				}
			}

			//sprite index < 0 means none
			client := &Client{
				hub: h,
				conn: conn.Connect,
				ip: conn.Ip,
				banned: same_ip >= ip_limit,
				send: make(chan []byte, 256),
				id: id,
				x: 0,
				y: 0,
				name: "",
				spd: 3,
				spriteName:	"none",
				spriteIndex: -1}
			go client.writePump()
			go client.readPump()

			client.send <- []byte("say" + delimstr + fmt.Sprintf("This room has %d players.", len(h.clients) + 1))
			client.send <- []byte("s" + delimstr + strconv.Itoa(id)) //"your id is %id%" message
			//send the new client info about the game state
			for other_client := range h.clients {
				client.send <- []byte("c" + delimstr + strconv.Itoa(other_client.id))
				client.send <- []byte("m" + delimstr + strconv.Itoa(other_client.id) + delimstr + strconv.Itoa(other_client.x) + delimstr + strconv.Itoa(other_client.y));
				client.send <- []byte("spd" + delimstr + strconv.Itoa(other_client.id) + delimstr + strconv.Itoa(other_client.spd));
				if other_client.name != "" {
					client.send <- []byte("name" + delimstr + strconv.Itoa(other_client.id) + delimstr + other_client.name);
				}
				if other_client.spriteIndex >= 0 { //if the other client sent us valid sprite and index before
					client.send <- []byte("spr" + delimstr + strconv.Itoa(other_client.id) + delimstr + other_client.spriteName + delimstr + strconv.Itoa(other_client.spriteIndex));
				}
				if other_client.systemName != "" {
					client.send <- []byte("sys" + delimstr + strconv.Itoa(other_client.id) + other_client.systemName);
				}
			}
			//register client in the structures
			h.id[id] = true
			h.clients[client] = true
			//tell everyone that a new client has connected
			if !client.banned {
				h.broadcast([]byte("c" + delimstr + strconv.Itoa(id))) //user %id% has connected
			}

			writeLog(conn.Ip, "connect", 200)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.deleteClient(client)
			}
			writeLog(client.ip, "disconnect", 200)
		case message := <-h.processMsgCh:
			err := h.processMsg(message)
			if err != nil {
				writeErrLog(message.sender.ip, err.Error())
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func (hub *Hub) serveWs(w http.ResponseWriter, r *http.Request) {
	protocols := r.Header.Get("Sec-Websocket-Protocol")
	conn, err := upgrader.Upgrade(w, r, http.Header{"Sec-Websocket-Protocol": {protocols}})
	if err != nil {
		log.Println(err)
		return
	}
	ip := r.Header.Get("x-forwarded-for")
	if ip == "" {
		ip = r.RemoteAddr
	}
	hub.connect <- &ConnInfo{Connect: conn, Ip: strings.Split(ip, ":")[0]}
}

func (h *Hub) broadcast(data []byte) {
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			h.deleteClient(client)
		}
	}
}

func (h *Hub) deleteClient(client *Client) {
	delete(h.id, client.id)
	close(client.send)
	delete(h.clients, client)
	h.broadcast([]byte("d" + delimstr + strconv.Itoa(client.id))) //user %id% has disconnected message
}

func (h *Hub) processMsg(msg *Message) error {
	if msg.sender.banned {
		return errors.New("Banned")
	}

	if len(msg.data) > 1024 {
		return errors.New("Request too long")
	}

	for _, v := range msg.data {
		if v < 32 {
			return errors.New("Bad byte sequence")
		}
	}

	if !utf8.Valid(msg.data) {
		return errors.New("Invalid UTF-8")
	}

	msgStr := string(msg.data[:])

	err := errors.New(msgStr)
	msgFields := strings.FieldsFunc(msgStr, func(c rune) bool {
		return c == delimrune
	}) //split message string on delimiting character

	if len(msgFields) == 0 {
		return err
	}

	switch msgFields[0] {
	case "m": //"i moved to x y"
		if len(msgFields) != 3 {
			return err
		}
		//check if the coordinates are valid
		x, errconv := strconv.Atoi(msgFields[1])
		if errconv != nil {
			return err
		}
		y, errconv := strconv.Atoi(msgFields[2]);
		if errconv != nil {
			return err
		}
		msg.sender.x = x
		msg.sender.y = y
		h.broadcast([]byte("m" + delimstr + strconv.Itoa(msg.sender.id) + delimstr + msgFields[1] + delimstr + msgFields[2])) //user %id% moved to x y
	case "spd": //change my speed to spd
		if len(msgFields) != 2 {
			return err
		}
		spd, errconv := strconv.Atoi(msgFields[1])
		if errconv != nil {
			return err
		}
		if spd < 0 || spd > 10 { //something's not right
			return err
		}
		msg.sender.spd = spd
		h.broadcast([]byte("spd" + delimstr + strconv.Itoa(msg.sender.id) + delimstr + msgFields[1]));
	case "spr": //change my sprite
		if len(msgFields) != 3 {
			return err
		}
		if !h.isValidSpriteName(msgFields[1]) {
			return err
		}
		index, errconv := strconv.Atoi(msgFields[2])
		if errconv != nil || index < 0 {
			return err
		}
		msg.sender.spriteName = msgFields[1]
		msg.sender.spriteIndex = index
		h.broadcast([]byte("spr" + delimstr + strconv.Itoa(msg.sender.id) + delimstr + msgFields[1] + delimstr + msgFields[2]));
	case "sys": //change my system graphic
		if len(msgFields) != 2 {
			return err
		}
		if !h.isValidSystemName(msgFields[1]) {
			return err
		}
		msg.sender.systemName = msgFields[1];
		h.broadcast([]byte("sys" + delimstr + strconv.Itoa(msg.sender.id) + delimstr + msgFields[1]));
	case "say":
		if len(msgFields) != 2 {
			return err
		}
		if msg.sender.name == "" {
			return err
		}

		msgContents := msgFields[1]

		// If it's a global message, broadcast it to every hub
		// Otherwise, broadcast only to the current hub
		switch {
		case strings.HasPrefix(msgContents, "!login "):
			msgContents := msgContents[7:]
			log.Printf("Trying to login with %s", msgContents)
			if h.controller.auth(msgContents) {
				msg.sender.send <- []byte("say" + delimstr + "Login successful")
				msg.sender.privilegedSession = true
			} else {
				msg.sender.send <- []byte("say" + delimstr + "Login unsuccessful")
			}

		case strings.HasPrefix(msgContents, "!global ") && msg.sender.privilegedSession:
			msgContents := msgContents[8:]
			h.controller.globalBroadcast([]byte("say" + delimstr + "<" + msg.sender.name + "> " + fmt.Sprintf("(GLOBAL) %s", msgContents)))
		case strings.HasPrefix(msgContents, "!"):
			// do nothing
		default:
			h.broadcast([]byte("say" + delimstr + "<" + msg.sender.name + "> " + msgContents))
		}
	case "name": // nick set
		if msg.sender.name != "" || len(msgFields) != 2 || !isOkName(msgFields[1]) || len(msgFields[1]) > 7 {
			return err
		}
		msg.sender.name = msgFields[1]
		h.broadcast([]byte("name" + delimstr + strconv.Itoa(msg.sender.id) + delimstr + msg.sender.name));
	default:
		return err
	}

	writeLog(msg.sender.ip, msgStr, 200)

	return nil
}

func (h *Hub) isValidSpriteName(name string) bool {
	for _, otherName := range h.spriteNames {
		if otherName == strings.ToLower(name) {
			return true
		}
	}
	return false
}

func (h *Hub) isValidSystemName(name string) bool {
	for _, otherName := range h.systemNames {
		if otherName == strings.ToLower(name) {
			return true
		}
	}
	return false
}
