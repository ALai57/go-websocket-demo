package main

import (
	"encoding/json"
	"fmt"
	"go_websocket_demo/pkg/websocket_api"
	"log/slog"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 1 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 5) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// /////////////////////////////////////////////////////////////////
// HTTP adapter
// /////////////////////////////////////////////////////////////////
func DecodeWebsocketRequest(senderAddr string, s string) (websocket_api.Command, error) {
	in := make(map[string]interface{})
	err := json.Unmarshal([]byte(s), &in)
	if err != nil {
		slog.Error(fmt.Sprintf("Error unmarshaling: not JSON formatted %v", err))
		return nil, nil
	}

	var req websocket_api.Command
	if v, exists := in["action"]; exists {
		slog.Debug(fmt.Sprintf("Message type: %v", v))

		switch v {
		case "broadcast":
			tmp := websocket_api.Broadcast{}
			err := json.Unmarshal([]byte(s), &tmp)
			if err != nil {
				slog.Error(fmt.Sprintf("Error unmarshaling Broadcast message: %v", err))
			}
			tmp.From = senderAddr
			req = tmp
		case "whoall":
			tmp := websocket_api.WhoAll{}
			err := json.Unmarshal([]byte(s), &tmp)
			if err != nil {
				slog.Error(fmt.Sprintf("Error unmarshaling Broadcast message: %v", err))
			}
			tmp.From = senderAddr
			req = tmp

		}
		slog.Debug(fmt.Sprintf("Decoded action %v", spew.Sdump(req)))
	}
	return req, nil
}

func inboundMessageReader(s *websocket_api.Service, conn *websocket.Conn) {
	senderAddr := conn.RemoteAddr().String()
	slog.Debug(fmt.Sprintf("Starting reader for: %v", senderAddr))

	defer func() {
		conn.Close()
		slog.Info(fmt.Sprintf("[%v] Closing", senderAddr))
	}()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		slog.Debug(fmt.Sprintf("[%v] Connection alive", senderAddr))
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, b, err := conn.ReadMessage()
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading websocket message %v", err))
			break
		}

		msg := string(b)
		slog.Debug(fmt.Sprintf("[%v] %v", senderAddr, msg))

		if action, err := DecodeWebsocketRequest(senderAddr, msg); action != nil {
			action.Exec(s)
		} else {
			slog.Error(fmt.Sprintf("Error reading websocket message %v", err))
		}
	}
}

func outboundMessageWriter(s *websocket_api.Service, conn *websocket.Conn) {

	remoteAddr := conn.RemoteAddr().String()
	slog.Debug(fmt.Sprintf("Starting writer for: %v", remoteAddr))

	publisher, exists := s.SocketMgr.(*InMemorySocketManager).DB[remoteAddr]
	if !exists {
		slog.Error(fmt.Sprintf("Cannot write to websocket %v", remoteAddr))
	}

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		c := publisher.(*LocalUser).channel
		select {
		case message, ok := <-c:
			slog.Debug(fmt.Sprintf("Reading from channel"))
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(message))

			// Add queued chat messages to the current websocket message.
			n := len(c)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write([]byte(<-c))
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			slog.Debug(fmt.Sprintf("[%v] Sending heartbeat", remoteAddr))
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func heartbeat(conn *websocket.Conn) {
	senderAddr := conn.RemoteAddr().String()
	slog.Debug(fmt.Sprintf("Starting reader for: %v", senderAddr))
	defer conn.Close()

	for {
		timer := time.After(time.Second * 5)
		<-timer
		fmt.Println("Heartbeat")
	}
}

// List of clients connected

// https://github.com/gorilla/websocket/blob/main/examples/chat/client.go#L125
func ServeWebSocket(s *websocket_api.Service) http.Handler {
	// The http.Request is the request to intiate a connection - it does
	// not represent new websocket messages getting published
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error(fmt.Sprintf("Could not upgrade: %v", err))
			return
		}

		slog.Debug(fmt.Sprintf("Connection: %v", conn.NetConn().LocalAddr()))

		senderAddr := conn.RemoteAddr().String()

		s.SocketMgr.Create(senderAddr)

		go inboundMessageReader(s, conn)
		go outboundMessageWriter(s, conn)
		//go heartbeat(conn)
	})
}

func GetParticipants(s *websocket_api.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
	})
}

func NewRouter(svc *websocket_api.Service) *mux.Router {

	router := mux.NewRouter()
	router.Handle("/ws", ServeWebSocket(svc))
	router.Handle("/participants", GetParticipants(svc))

	return router
}

// /////////////////////////////////////////////////////////////////
// Custom data types
// /////////////////////////////////////////////////////////////////
type LocalUser struct {
	address string `json:"Address"`
	name    string `json:"Name"`
	channel chan string
}

func (lu *LocalUser) Address() string {
	return lu.address
}

func (lu *LocalUser) Name() string {
	return "Unimplemented"
}

func (lu *LocalUser) SendMessage(s string) error {
	slog.Info(fmt.Sprintf("Sending Message '%v'", s))
	lu.channel <- s
	return nil
}

func (lu *LocalUser) MarshalJSON() ([]byte, error) {
	result := map[string]string{
		"Name":    "Unidentified user",
		"Address": "Unknown",
	}

	if lu.name != "" {
		result["Name"] = lu.name
	}

	if lu.address != "" {
		result["Address"] = lu.address
	}

	return json.Marshal(result)
}

func NewLocalUser(addr string) LocalUser {
	return LocalUser{
		address: addr,
		channel: make(chan string),
	}
}

// /////////////////////////////////////////////////////////////////
// Manage Websocket connections
// Different in HTTP vs Lambda
// /////////////////////////////////////////////////////////////////
type InMemorySocketManager struct {
	DB map[string]websocket_api.Messageable
}

func (memSM *InMemorySocketManager) Create(addr string) (websocket_api.Messageable, error) {
	slog.Info(fmt.Sprintf("Creating connection %v", addr))
	publisher := NewLocalUser(addr)
	memSM.DB[addr] = &publisher
	return &publisher, nil
}

func (memSM *InMemorySocketManager) Delete(id string) error {
	return nil
}

func (memSM *InMemorySocketManager) GetConnections() ([]websocket_api.Messageable, error) {
	slog.Info("Getting Live Connections from connection manager")

	var results []websocket_api.Messageable
	for _, c := range memSM.DB {
		results = append(results, c)
	}
	return results, nil
}

func (memSM *InMemorySocketManager) Find(addr string) (websocket_api.Messageable, bool) {
	slog.Info("Looking up connection...")

	result, exists := memSM.DB[addr]
	return result, exists
}

// /////////////////////////////////////////////////////////////////
// Main
// /////////////////////////////////////////////////////////////////
func main() {
	slog.Info("Starting up Web server on port 7002")

	svc := websocket_api.NewService()
	socketMgr := InMemorySocketManager{
		DB: map[string]websocket_api.Messageable{},
	}
	svc.SocketMgr = &socketMgr

	router := NewRouter(svc)
	srv := &http.Server{
		Addr:    ":7002",
		Handler: router,
	}
	err := srv.ListenAndServe()
	if err != nil {
		slog.Error(err.Error())
		panic("Could not start")
	}
}
