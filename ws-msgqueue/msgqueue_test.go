package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"testing"
)

var serverRunning = false

func TestMain(m *testing.M) {
	ready := make(chan bool)

	go func() {
		err := initServer("localhost:8080", ready)

		if err != nil {
			fmt.Println(err)
			serverRunning = false
		}
	}()

	srv := <-ready

	if srv == true {
		serverRunning = true
	}

	os.Exit(m.Run())
}

func TestInitServer(t *testing.T) {
	if serverRunning == false {
		// t. will only show up after the tests are done, it doesn't work if we force the exit
		fmt.Println("Server not running")
		os.Exit(1)
	}
}

func TestRoundtrip(t *testing.T) {
	popUrl := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/popmsg"}
	pushUrl := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/pushmsg"}

	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("hello:test"))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	popConn, _, err := websocket.DefaultDialer.Dial(popUrl.String(), authHeader)

	if err != nil {
		t.Fatal(err)
	}

	pushConn, _, err := websocket.DefaultDialer.Dial(pushUrl.String(), authHeader)

	if err != nil {
		t.Fatal(err)
	}

	err = pushConn.WriteMessage(websocket.TextMessage, []byte("hello team!"))

	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := popConn.ReadMessage()

	if !bytes.Equal(msg, []byte("hello team!")) {
		t.Fatal("[tests] messages don't match")
	}
}
