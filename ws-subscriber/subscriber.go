package main

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { //disable after done with js tests
		return true
	},
}

var subscribed = make(map[*websocket.Conn]bool)

func main() {
	url := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/popmsg"}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	serviceConn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)

	if err != nil { // add retry later
		fmt.Println(err)
		return
	}

	go func() {
		defer serviceConn.Close()

		for {
			_, msg, err := serviceConn.ReadMessage()

			if err != nil {
				fmt.Println(err)
				return
			}

			for sub := range subscribed {
				sub.WriteMessage(websocket.TextMessage, msg)
			}
		}
	}()

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Println(err)
			return
		}

		subscribed[conn] = true
	})

	err = http.ListenAndServeTLS("localhost:8082", os.Getenv("WS_CERT"), os.Getenv("WS_CERTKEY"), nil)

	if err != nil {
		fmt.Println(err)
	}
}
