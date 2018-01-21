package main

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
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

var thingsToPush = make(chan []byte, 10)

func main() {
	pushConn, err := dialToService("localhost:8080", "/pushmsg", "hello", "test")

	if err != nil {
		fmt.Printf("[publisher] error dialing server\n%s", err)
		return
	}

	go pushMessages(pushConn)

	ready := make(chan bool, 1)
	err = initServer("localhost:8081", os.Getenv("WS_CERT_DIR"), ready)

	if err != nil {
		fmt.Printf("[publisher] error initialising server\n%s", err)
	}
}

func dialToService(addr string, path string, username string, password string) (*websocket.Conn, error) {
	serviceURL := url.URL{Scheme: "wss", Host: addr, Path: path}
	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	serviceConn, _, err := websocket.DefaultDialer.Dial(serviceURL.String(), authHeader)

	if err != nil {
		return nil, err
	}

	return serviceConn, nil
}

func pushMessages(conn *websocket.Conn) {
	for {
		select {
		case msg := <-thingsToPush:
			err := conn.WriteMessage(websocket.TextMessage, msg)

			if err != nil {
				fmt.Printf("[publisher] error sending msg: %s\n%s", msg, err)
				continue
			}

			fmt.Printf("[publisher] pushing %s\n", msg)
		}
	}
}

func initServer(addr string, certDir string, ch chan<- bool) error {
	if certDir == "" {
		err := errors.New("initServer: certDir is not defined")
		return err
	}

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[publisher] error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[publisher] error upgrading connection\n%s", err)
			return
		}

		go func() {
			for {
				_, msg, err := conn.ReadMessage()

				if err != nil {
					fmt.Println(err)
					return
				}

				thingsToPush <- msg

				fmt.Printf("[publisher] received %s\n", msg)
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
