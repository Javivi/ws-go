package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
	"os"
)

func main() {
	demo := flag.String("demo", "", "act as a pub or sub")
	addr := flag.String("addr", "", "microservice address")
	flag.Parse()

	path := ""
	if *demo == "pub" {
		path = "/publish"
	} else if *demo == "sub" {
		path = "/subscribe"
	} else {
		fmt.Println("-demo must have a valid value [pub,sub]")
		return
	}

	if *addr == "" {
		fmt.Println("-addr must have a valid value [addr:port")
		return
	}

	url := url.URL{Scheme: "wss", Host: *addr, Path: path}
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	serviceConn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)

	if err != nil {
		fmt.Printf("%s [%s]", err, *addr)
		return
	}

	if *demo == "pub" {
		input := bufio.NewScanner(os.Stdin)

		for input.Scan() {
			msg := input.Bytes()
			fmt.Printf("Message SENT\n")

			err := serviceConn.WriteMessage(websocket.TextMessage, msg)

			if err != nil {
				fmt.Println(err)
				return
			}

			return
		}

		if err := input.Err(); err != nil {
			fmt.Println("error reading input")
			return
		}
	} else {
		go func() {
			defer serviceConn.Close()

			for {
				_, msg, err := serviceConn.ReadMessage()

				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Printf("%s\n", msg)
			}
		}()

		fmt.Println("Press ENTER to stop subscribing and end the program")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}
