package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"testing"
	"time"
)

var serverRunning = false

func TestMain(m *testing.M) {
	ready := make(chan bool)

	go func() {
		err := initServer("localhost:8082", os.Getenv("WS_CERT_DIR"), ready)

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

func TestNoCertDir(t *testing.T) {
	ready := make(chan bool)

	go func() {
		initServer("localhost:8899", "", ready)
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
	_, err := dialToService("localhost:8082", "/subscribe", "fail", "test")
	if err != websocket.ErrBadHandshake {
		t.Fatal("[test] Successfully authenticated with bad credentials")
	}
}

func TestDialerFail(t *testing.T) {
	_, err := dialToService("invalid addr", "", "fail", "test")

	if err == nil {
		t.Fatal("[test] Successfully dialed to a wrong address")
	}
}

func TestFailedUpgrade(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", "https://localhost:8082/subscribe", nil)

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
			fmt.Printf("[tests] Error validating credentials [%s:%s]\n", username, password)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Printf("[tests] Error upgrading connection\n%s", err)
			return
		}

		for {
			_, msg, err := conn.ReadMessage()

			if err != nil {
				fmt.Printf("[tests] Error reading message\n%s", err)
				return
			}

			if bytes.Equal(msg, []byte("")) {
				err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

				if err != nil {
					fmt.Printf("[tests] Error closing\n%s", err)
					return
				}

				return
			}

			err = conn.WriteMessage(websocket.TextMessage, msg)

			if err != nil {
				fmt.Printf("[tests] Error sending message\n%s", err)
				return
			}
		}
	})

	cert, err := tls.LoadX509KeyPair(os.Getenv("WS_CERT_DIR")+"server.crt", os.Getenv("WS_CERT_DIR")+"server.key")

	if err != nil {
		t.Fatal(err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", "localhost:8999", config)

	if err != nil {
		t.Fatal(err)
	}

	defer listener.Close()

	go http.Serve(listener, nil)

	popConn, err := dialToService("localhost:8999", "/test", "hello", "test")

	if err != nil {
		t.Fatal(err)
	}

	err = popConn.WriteJSON(message{"test", "message"})

	if err != nil {
		t.Fatal(err)
	}

	subConn, err := dialToService("localhost:8082", "/subscribe", "hello", "test")

	if err != nil {
		t.Fatal(err)
	}

	err = subConn.WriteJSON(message{"test", "sub"})

	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)

	exitRoutine := make(chan bool)
	go func() {
		for {
			select {
			case <-exitRoutine:
				return
			default:
				popMessages(popConn, exitRoutine)
			}
		}
	}()

	subConn.SetReadDeadline(time.Now().Add(time.Second * 3))

	msg := &message{}
	err = subConn.ReadJSON(msg)

	if err != nil {
		t.Fatal(err)
	}

	if msg.Topic != "test" || msg.Content != "message" {
		t.Fatal("[tests] Messages don't match")
	}

	err = subConn.WriteJSON(message{"test", "unsub"})

	if err != nil {
		t.Fatal(err)
	}

	err = popConn.WriteJSON(message{"test", "message"})

	if err != nil {
		t.Fatal(err)
	}

	subConn.SetReadDeadline(time.Now().Add(time.Second * 3))

	_, _, err = subConn.ReadMessage()

	if err == nil {
		t.Fatal("[tests] Received a message after unsubscribing")
	}

	err = popConn.WriteMessage(websocket.TextMessage, []byte(""))

	if err != nil {
		t.Fatal(err)
	}

	subConn.SetReadDeadline(time.Now().Add(time.Second * 3))

	_, _, err = subConn.ReadMessage()

	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Fatal("[tests] Received a message after closing the websocket")
	}
}
