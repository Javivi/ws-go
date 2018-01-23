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
		err := initServer("localhost:8080", os.Getenv("WS_CERT_DIR"), ready)

		if err != nil {
			fmt.Println(err)
			serverRunning = false
			close(ready)
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
		fmt.Println("[tests] Server not running")
		os.Exit(1)
	}
}

func TestNoCertDir(t *testing.T) {
	ready := make(chan bool)

	go func() {
		initServer("localhost:8888", ".invaliddir", ready)
	}()

	select {
	case <-ready:
		t.Fatal()
	case <-time.After(time.Second * 3):
		return
	}
}

func TestInvalidCredentials(t *testing.T) {
	popURL := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/popmsg"}
	pushURL := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/pushmsg"}

	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("fail:test"))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	_, _, err := websocket.DefaultDialer.Dial(popURL.String(), authHeader)

	if err != websocket.ErrBadHandshake {
		t.Fatal("[test] Successfully authenticated with bad credentials")
	}

	_, _, err = websocket.DefaultDialer.Dial(pushURL.String(), authHeader)

	if err != websocket.ErrBadHandshake {
		t.Fatal("[test] Successfully authenticated with bad credentials")
	}
}

func TestFailedUpgrade(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", "https://localhost:8080/popmsg", nil)

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

	req, err = http.NewRequest("GET", "https://localhost:8080/pushmsg", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("hello", "test")
	response, err = client.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	response.Body.Close()
	if response.Status != "400 Bad Request" {
		t.Fatal("[test] Invalid ws upgrade didn't fail")
	}
}

func TestRoundtrip(t *testing.T) {
	popURL := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/popmsg"}
	pushURL := url.URL{Scheme: "wss", Host: "localhost:8080", Path: "/pushmsg"}

	authHeader := http.Header{"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("hello:test"))}}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	popConn, _, err := websocket.DefaultDialer.Dial(popURL.String(), authHeader)

	if err != nil {
		t.Fatal(err)
	}

	pushConn, _, err := websocket.DefaultDialer.Dial(pushURL.String(), authHeader)

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
