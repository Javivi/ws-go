package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var messageQueue = make(chan []byte, 100)

func main() {
	ready := make(chan bool, 10)
	err := initServer("localhost:8080", os.Getenv("WS_CERT_DIR"), ready)

	if err != nil {
		fmt.Printf("ERROR [%s] initialising server\n", err)
	}
}

func initServer(addr string, certDir string, ch chan<- bool) error {
	//defer close(ch)
	if certDir == "" {
		err := errors.New("initServer: certDir is not defined")
		return err
	}

	http.HandleFunc("/pushmsg", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("ERROR [%s:%s] invalid credentials\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("ERROR [%s] upgrading connection\n", err)
			return
		}

		go func() {
			for {
				_, msg, err := conn.ReadMessage()

				if err != nil {
					fmt.Printf("ERROR [%s] reading message\n", err)
					return
				}

				messageQueue <- msg
				fmt.Printf("[msgqueue] pushing message: %s\n", msg)
			}
		}()
	})

	http.HandleFunc("/popmsg", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("ERROR [%s:%s] invalid credentials\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("ERROR [%s] upgrading connection\n", err)
			return
		}

		go func() {
			for {
				select {
				case msg := <-messageQueue:
					fmt.Printf("[msgqueue] popping message: %s\n", msg)
					err := conn.WriteMessage(websocket.TextMessage, msg)

					if err != nil {
						fmt.Printf("ERROR [%s] sending message\n", err)
						return
					}
				}
			}
		}()
	})

	ch <- true

	err := http.ListenAndServeTLS(addr, certDir+"server.crt", certDir+"server.key", nil)

	if err != http.ErrServerClosed {
		return err
	}

	return nil
}
