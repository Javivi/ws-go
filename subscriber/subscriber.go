package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type safeSubscribe struct {
	subs map[string]map[*websocket.Conn]bool
	mux  sync.Mutex
}

var subscribers = safeSubscribe{subs: make(map[string]map[*websocket.Conn]bool)}

type message struct {
	Topic   string
	Content string
}

func main() {
	popConn, err := dialToService("localhost:8080", "/popmsg", "hello", "test")

	if err != nil {
		fmt.Printf("[subscriber] Error dialing server\n%s\n", err)
		return
	}

	go popMessages(popConn, nil)

	err = initServer("localhost:8082", os.Getenv("WS_CERT_DIR"), nil)

	if err != nil {
		fmt.Printf("[subscriber] Error initialising server\n%s\n", err)
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

func popMessages(conn *websocket.Conn, connClosed chan bool) {
	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Printf("[subscriber] Error reading message\n%s\n", err)

			if connClosed != nil {
				close(connClosed)
			}

			break
		}

		fmt.Printf("[subscriber] Received %s\n", msg)

		var m message
		err = json.Unmarshal(msg, &m)

		if err != nil {
			fmt.Printf("[subscriber] JSON unmarshal error %s\n", err)
			return
		}

		subscribers.mux.Lock()
		for sub := range subscribers.subs[m.Topic] {
			err := sub.WriteMessage(websocket.TextMessage, msg)

			if err != nil {
				fmt.Printf("[subscriber] Error sending message to one subscriber\n%s\n", err)
				continue
			}

			fmt.Printf("[subscriber] Pushing %s\n", msg)
		}

		if len(subscribers.subs[m.Topic]) == 0 {
			fmt.Printf("[subscriber] Ignoring message for topic without subscribers %s\n", msg)
		}
		subscribers.mux.Unlock()
	}
}

func initServer(addr string, certDir string, serverReady chan<- bool) error {
	cert, err := tls.LoadX509KeyPair(certDir+"server.crt", certDir+"server.key")

	if err != nil {
		return err
	}

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("[subscriber] Error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[subscriber] Error upgrading connection\n%s\n", err)
			return
		}

		go func() {
			for {
				msg := &message{}
				err := conn.ReadJSON(msg)

				if err != nil {
					fmt.Printf("[subscriber] Error reading JSON\n%s\n", err)
					return
				}

				if subscribers.subs[msg.Topic] == nil {
					subscribers.mux.Lock()
					subscribers.subs[msg.Topic] = make(map[*websocket.Conn]bool)
					subscribers.mux.Unlock()
				}

				if msg.Content == "sub" {
					subscribers.mux.Lock()
					subscribers.subs[msg.Topic][conn] = true
					subscribers.mux.Unlock()
					fmt.Printf("[subscriber] Subscribed to %s\n", msg.Topic)
					continue
				}

				if msg.Content == "unsub" {
					subscribers.mux.Lock()
					delete(subscribers.subs[msg.Topic], conn)
					subscribers.mux.Unlock()
					fmt.Printf("[subscriber] Unsubscribed from %s\n", msg.Topic)
					continue
				}

				fmt.Printf("[subscriber] Ignoring invalid message %s\n", msg)
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

	http.Serve(listener, nil)

	return nil
}
