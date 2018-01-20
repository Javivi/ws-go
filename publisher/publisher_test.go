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
	"time"
)

var serverRunning = false

func TestMain(m *testing.M) {
	ready := make(chan bool)

	go func() {
		err := initServer("localhost:8081", os.Getenv("WS_CERT_DIR"), ready)

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

func TestNoCertDir(t *testing.T) {
	ready := make(chan bool)

	go func() {
		initServer("localhost:8081", "", ready)
	}()

	select {
	case <-ready:
		t.Fatal()
	case <-time.After(time.Second * 3):
		return
	}
}

func TestInitServer(t *testing.T) {
	if serverRunning == false {
		// t. will only show up after the tests are done, it doesn't work if we force the exit
		fmt.Println("[tests] Server not running")
		os.Exit(1)
	}
}

func TestInvalidCredentials(t *testing.T) {
	pubURL := url.URL{Scheme: "wss", Host: "localhost:8081", Path: "/publish"}

	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("fail:test"))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	_, _, err := websocket.DefaultDialer.Dial(pubURL.String(), authHeader)

	if err != websocket.ErrBadHandshake {
		t.Fatal("[test] Successfully authenticated with bad credentials")
	}
}

func TestFailedUpgrade(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", "https://localhost:8081/publish", nil)

	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("hello", "test")
	response, err := client.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	response.Body.Close()
	if response.Status != "400 Bad Request" {
		t.Fatal("[test] Invalid ws upgrade didn't fail")
	}
}

func TestRoundtrip(t *testing.T) {
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok || username != "hello" || password != "test" {
			fmt.Printf("ERROR [%s:%s] invalid credentials\n", username, password)

		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("ERROR [%s] upgrading connection\n", err)
			return
		}

		_, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Printf("ERROR [%s] reading message\n", err)
			return
		}

		err = conn.WriteMessage(websocket.TextMessage, msg)

		if err != nil {
			fmt.Printf("ERROR [%s] sending message\n", err)
			return
		}
	})

	go http.ListenAndServeTLS("localhost:8089", os.Getenv("WS_CERT_DIR")+"server.crt", os.Getenv("WS_CERT_DIR")+"server.key", nil)

	pushConn, err := dialToService("localhost:8081", "/publish", "hello", "test")

	if err != nil {
		t.Fatal(err)
	}

	err = pushConn.WriteMessage(websocket.TextMessage, []byte("hello team!"))

	if err != nil {
		t.Fatal(err)
	}

	replyConn, err := dialToService("localhost:8089", "/test", "hello", "test")

	if err != nil {
		t.Fatal(err)
	}

	go pushMessages(replyConn)

	replyConn.SetReadDeadline(time.Now().Add(time.Second * 10))
	_, msg, err := replyConn.ReadMessage()

	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(msg, []byte("hello team!")) {
		t.Fatal("[tests] messages don't match")
	}
}
