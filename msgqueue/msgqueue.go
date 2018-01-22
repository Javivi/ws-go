package main

import (
	"crypto/tls"
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
	err := initServer("localhost:8080", os.Getenv("WS_CERT_DIR"), nil)

	if err != nil {
		fmt.Printf("[msgqueue] Error initialising server\n%s", err)
		os.Exit(1)
	}
}

func initServer(addr string, certDir string, serverReady chan<- bool) error {
	cert, err := tls.LoadX509KeyPair(certDir+"server.crt", certDir+"server.key")

	if err != nil {
		return err
	}

	http.HandleFunc("/pushmsg", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[msgqueue] Error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[msgqueue] Error upgrading connection\n%s", err)
			return
		}

		go func() {
			for {
				_, msg, err := conn.ReadMessage()

				if err != nil {
					fmt.Printf("[msgqueue] Error reading message\n%s", err)
					return
				}

				messageQueue <- msg

				fmt.Printf("[msgqueue] Pushing message: %s\n", msg)
			}
		}()
	})

	http.HandleFunc("/popmsg", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[msgqueue] Error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[msgqueue] Error upgrading connection\n%s", err)
			return
		}

		go func() {
			for {
				select {
				case msg := <-messageQueue:
					fmt.Printf("[msgqueue] Popping message: %s\n", msg)
					err := conn.WriteMessage(websocket.TextMessage, msg)

					if err != nil {
						fmt.Printf("[msgqueue] Error sending message\n%s", err)
						return
					}
				}
			}
		}()
	})

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", addr, config)

	if err != nil {
		return err
	}

	defer listener.Close()

	if serverReady != nil {
		serverReady <- true
	}

	fmt.Println("[msgqueue] Server running")
	http.Serve(listener, nil)

	return nil
}
