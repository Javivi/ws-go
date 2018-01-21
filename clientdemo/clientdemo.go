package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Message struct {
	Topic   string
	Content string
}

func main() {
	pubConn, err := dialToService("localhost:8081", "/publish", "hello", "test")

	if err != nil {
		fmt.Printf("[clientdemo] error dialing server\n%s", err)
		return
	}

	subConn, err := dialToService("localhost:8082", "/subscribe", "hello", "test")

	if err != nil {
		fmt.Printf("[clientdemo] error dialing server\n%s", err)
		return
	}

	go readMessages(subConn)

	writeMessages(pubConn, subConn)
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

func readMessages(conn *websocket.Conn) {
	for {
		msg := &Message{}
		err := conn.ReadJSON(msg)

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("<[%s] %s\n", msg.Topic, msg.Content)
	}
}

func writeMessages(pubConn *websocket.Conn, subConn *websocket.Conn) {
	input := bufio.NewScanner(os.Stdin)

	for input.Scan() {
		err := input.Err()

		if err != nil {
			fmt.Println("[clientdemo] error reading input")
			return
		}

		msg := string(input.Bytes())

		if msg == "exit" {
			return
		}

		firstWhitespace := strings.Index(msg, " ")

		if firstWhitespace == -1 {
			fmt.Println("[clientdemo] Invalid syntax, correct one: [topic message] or [sub/unsub topic]")
			continue
		}

		firstPart := msg[:firstWhitespace]
		secondPart := msg[firstWhitespace+1:]

		if firstPart == "sub" || firstPart == "unsub" {
			err = subConn.WriteJSON(Message{secondPart, firstPart})

			if err != nil {
				fmt.Println(err)
				continue
			}

			continue
		}

		err = pubConn.WriteJSON(Message{firstPart, secondPart})

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf(">[%s] %s\n", firstPart, secondPart)
	}
}
