package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options
var clients = make(map[string]SocketClient)

type SocketClient struct {
	Username string
	Conn     *websocket.Conn
}

type SocketMessage struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Got a /hello request")
	w.Write([]byte("Hello, HTTP!"))
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Got a /socket request")
	username := r.URL.Query().Get("username")
	if username == "" {
		w.Write([]byte("Missing 'name' query param!"))
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Info("Upgrade:", err)
		return
	}
	slog.Info(fmt.Sprintf("Upgraded to WS for user %v", username))
	clients[username] = SocketClient{Username: username, Conn: c}
	defer func() {
		delete(clients, username)
		c.Close()
	}()

	for {
		var socketMessage SocketMessage
		err := c.ReadJSON(&socketMessage)
		if err != nil {
			slog.Info("Read:", err)
			break
		}

		slog.Info(fmt.Sprintf("Received from %v: %+v", username, socketMessage))

		if socketMessage.Kind == "ping" {
			slog.Info(fmt.Sprintf("Sending '%v' to %v", "pong", username))
			c.WriteJSON("pong")
		} else if socketMessage.Kind == "chat" {
			broadcast(socketMessage.Content)
		}
	}
}

func broadcast(msg string) {
	for _, client := range clients {
		slog.Info(fmt.Sprintf("Sending '%v' to %v", msg, client.Username))
		client.Conn.WriteJSON(msg)
	}
}

func main() {
	port := 5050
	slog.Info(fmt.Sprintf("Starting server on port %v\n", port))

	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/socket", socketHandler)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), nil)
}
