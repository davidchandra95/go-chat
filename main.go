package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"html/template"
	"os"

	"log"
	"net/http"
	"strings"
)

type M map[string]interface{}

const (
	MESSAGE_NEW_USER = "New User"
	MESSAGE_CHAT     = "CHAT"
	MESSAGE_LEAVE    = "Leave"
)

var connections = make([]*WebSocketConnection, 0)

// Storing payload from front-end
type SocketPayload struct {
	Message string
}

// For broadcasting message to all connected users
type SocketResponse struct {
	From    string
	Type    string
	Message string
}

// Store client data
type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	port := os.Getenv("PORT") //Get port from .env file, we did not specify any port so this should return an empty string when tested locally
	if port == "" {
		port = "8000" //localhost
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type Index struct {
			Port string
		}

		content := &Index{Port:port}

		t, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, "Could not open requested file", http.StatusInternalServerError)
			return
		}

		_ = t.Execute(w, content)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		currentGorillaConn, err := upgrader.Upgrade(w, r, w.Header())
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		}

		username := r.URL.Query().Get("username")
		currentConn := WebSocketConnection{Conn: currentGorillaConn, Username: username}
		connections = append(connections, &currentConn)

		go handleIO(&currentConn, connections)
	})

	fmt.Println("Server starting at http://localhost:" + port)
	err := http.ListenAndServe(":"+port, nil) //Launch the app, visit localhost:8000/api
	if err != nil {
		fmt.Print(err)
	}
}

func handleIO(currentConn *WebSocketConnection, connections []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)

				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	// remove conn from list of connected
	for i := len(connections) - 1; i >= 0; i-- {
		connection := connections[i]
		// Condition to decide if current element has to be deleted:
		if connection == currentConn {
			connections = append(connections[:i],
				connections[i+1:]...)
		}
	}
}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		_ = eachConn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}
