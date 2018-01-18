package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"fmt"
)

var upgrader = websocket.Upgrader {
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { //disable after done with js tests
		return true
	},
}

var message_queue = make(chan []byte, 100)

func main() {
	defer close(message_queue)

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
				message_queue <- msg
			}
		}()
	})

	http.HandleFunc("/popmsg", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println(err)
			return
		}

		go func() {
			defer conn.Close()

			for {
				select {
				case m := <- message_queue:
					conn.WriteMessage(websocket.TextMessage, m)
				}
			}
		}()
	})

	http.ListenAndServe("localhost:8080", nil)
}