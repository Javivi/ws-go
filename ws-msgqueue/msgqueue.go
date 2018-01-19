package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { //disable after done with js tests
		return true
	},
}

var messageQueue = make(chan []byte, 100)

func main() {
	defer close(messageQueue)

	http.HandleFunc("/pushmsg", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Println(err)
			return
		}

		go func() {
			defer conn.Close()

			for {
				_, msg, err := conn.ReadMessage()

				if err != nil {
					fmt.Println(err)
					return
				}
				messageQueue <- msg
			}
		}()
	})

	http.HandleFunc("/popmsg", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Println(err)
			return
		}

		go func() {
			defer conn.Close()

			for {
				select {
				case m := <-messageQueue:
					conn.WriteMessage(websocket.TextMessage, m)
				}
			}
		}()
	})

	err := http.ListenAndServeTLS("localhost:8080", os.Getenv("WS_CERT_DIR")+"server.crt", os.Getenv("WS_CERT_DIR")+"server.key", nil)

	if err != nil {
		fmt.Println(err)
	}
}
