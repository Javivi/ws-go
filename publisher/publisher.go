package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var thingsToPush = make(chan []byte)

func main() {
	pushUrl := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/pushmsg"}
	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("hello:test"))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	pushConn, _, err := websocket.DefaultDialer.Dial(pushUrl.String(), authHeader)

	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		for {
			select {
			case m := <-thingsToPush:
				err := pushConn.WriteMessage(websocket.TextMessage, m)

				if err != nil {
					fmt.Printf("ERROR [%s] sending msg: %s\n", err, m)
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

	err = http.ListenAndServeTLS("localhost:8081", os.Getenv("WS_CERT_DIR")+"server.crt", os.Getenv("WS_CERT_DIR")+"server.key", nil)

	if err != nil {
		fmt.Println(err)
	}
}
