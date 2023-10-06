package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WsHTTPHandler is a handler for websockets
func WsHTTPHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := &websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func main() {
	http.HandleFunc("/", WsHTTPHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
