package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
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

var subscribers = make(map[string]map[*websocket.Conn]bool)

type Message struct {
	Topic   string
	Content string
}

func main() {
	popConn, err := dialToService("localhost:8080", "/popmsg", "hello", "test")

	if err != nil {
		fmt.Printf("[subscriber] error dialing server\n%s", err)
		return
	}

	go popMessages(popConn)

	ready := make(chan bool, 1)
	err = initServer("localhost:8082", os.Getenv("WS_CERT_DIR"), ready)

	if err != nil {
		fmt.Printf("[subscriber] error initialising server\n%s", err)
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

func popMessages(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("[subscriber] received %s\n", msg)

		var m Message
		err = json.Unmarshal(msg, &m)

		if err != nil {
			fmt.Println(err)
			return
		}

		for sub := range subscribers[m.Topic] {
			err := sub.WriteMessage(websocket.TextMessage, msg)

			if err != nil {
				fmt.Printf("[subscriber] error sending message to one subscriber\n%s", err)
				continue
			}

			fmt.Printf("[subscriber] pushing %s\n", msg)
		}
	}
}

func initServer(addr string, certDir string, ch chan<- bool) error {
	if certDir == "" {
		err := errors.New("initServer: certDir is not defined")
		return err
	}

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[subscriber] error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[subscriber] error upgrading connection\n%s", err)
			return
		}

		go func() {
			for {
				msg := &Message{}
				err := conn.ReadJSON(msg)

				if err != nil {
					fmt.Println(err)
					return
				}

				if subscribers[msg.Topic] == nil {
					subscribers[msg.Topic] = make(map[*websocket.Conn]bool)
				}

				if msg.Content == "sub" {
					subscribers[msg.Topic][conn] = true
					fmt.Printf("[subscriber] subscribed to %s\n", msg.Topic)
					continue
				}

				if msg.Content == "unsub" {
					delete(subscribers[msg.Topic], conn)
					fmt.Printf("[subscriber] unsubscribed from %s\n", msg.Topic)
					continue
				}

				fmt.Printf("[subscriber] ignoring invalid message %v\n", msg)
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
