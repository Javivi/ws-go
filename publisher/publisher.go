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

var thingsToPush = make(chan []byte, 10)

func main() {
	pushConn, err := dialToService("localhost:8080", "/pushmsg", "hello", "test")

	if err != nil {
		fmt.Printf("[publisher] Error dialing server\n%s", err)
		os.Exit(1)
	}

	go pushMessages(pushConn)

	err = initServer("localhost:8081", os.Getenv("WS_CERT_DIR"), nil)

	if err != nil {
		fmt.Printf("[publisher] Error initialising server\n%s", err)
		os.Exit(1)
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
				fmt.Printf("[publisher] Error sending msg: %s\n%s", msg, err)
				continue
			}

			fmt.Printf("[publisher] Pushing %s\n", msg)
		}
	}
}

func initServer(addr string, certDir string, serverReady chan<- bool) error {
	cert, err := tls.LoadX509KeyPair(certDir+"server.crt", certDir+"server.key")

	if err != nil {
		return err
	}

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[publisher] Error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[publisher] Error upgrading connection\n%s", err)
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

				fmt.Printf("[publisher] Received %s\n", msg)
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

	fmt.Println("[publisher] Server running")
	http.Serve(listener, nil)

	return nil
}
