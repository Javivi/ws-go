package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { //disable after done with js tests
		return true
	},
}

var thingsToPush = make(chan []byte)

func main() {
	defer close(thingsToPush)

	url := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/pushmsg"}
	serviceConn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)

	if err != nil { // add retry later
		fmt.Println(err)
		return
	}

	go func() {
		defer serviceConn.Close()

		for {
			select {
			case m := <-thingsToPush:
				err := serviceConn.WriteMessage(websocket.TextMessage, m)

				if err != nil {
					fmt.Printf("Error sending msg: %s", m)
				}
			}
		}
	}()

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		defer conn.Close()

		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			_, msg, err := conn.ReadMessage()

			if err != nil {
				fmt.Println(err)
				return
			}

			thingsToPush <- msg
			return
		}
	})

	http.ListenAndServe("localhost:8081", nil)
}
