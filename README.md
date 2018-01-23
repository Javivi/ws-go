[![Build Status](https://travis-ci.org/Javivi/ws-go.svg?branch=master)](https://travis-ci.org/Javivi/ws-go)
[![Coverage Status](https://coveralls.io/repos/github/Javivi/ws-go/badge.svg?branch=master)](https://coveralls.io/github/Javivi/ws-go?branch=master)

![demo](https://github.com/Javivi/ws-go/blob/master/smalldemo.gif)

# WebSocket pub/sub/queue example

![diagram](https://github.com/Javivi/ws-go/blob/master/diagram.svg)

This project consist of three parts:
* [msgqueue](https://github.com/Javivi/ws-go/tree/master/msgqueue): A simple FIFO message queue that receives messages from the publishers and sends them to the subscribers
* [publisher](https://github.com/Javivi/ws-go/tree/master/publisher): A microservice that listens for incoming messages and pushes them to the message queue
* [subscriber](https://github.com/Javivi/ws-go/tree/master/subscriber): A microservice that listens for incoming subscribe/unsubscribe messages and also handles messages coming from the message queue and pushes them to whoever has subscribed to the topic of the message

There's also [an example client](https://github.com/Javivi/ws-go/tree/master/clientdemo) that can be used to test the microservices as shown [on this demonstration video](https://github.com/Javivi/ws-go/blob/master/fulldemo.webm?raw=true).

## Endpoints
In order to connect to any of the endpoints, a TLS connection must be used and a Basic HTTP Authentication header with valid credentials must be present on the request. For this demonstration project, a self-signed certificate can be found at the directory defined on the environment variable *WS_CERT_DIR*. The validity of this certificate is not tested when making new connections. As for the authentication, the hardcoded values *hello* and *test* are used as username and password.

### msgqueue
* **/pushmsg**: One publisher may connect here to send messages
* **/popmsg**: One subscriber may connect here to receive messages
* Listens on *localhost:8080*

### publisher
* **/publish**: Multiple clients may connect here to send messages to the msgqueue microservice
* Listens on *localhost:8081*

### subscriber
* **/subscribe**: Multiple clients may connect here to request to be subscribed or unsubscribed from a certain topic
* Listens on *localhost:8082*
* Valid requests: *sub topic* and *unsub topic*

# Tests

Every endpoint is tested without the need of the other microservices to be running. In order to do so, both the microservice's server and an extra one are used to simulate the current and the next microservice (as shown on the flow diagram).

### Tests done
Test Name | Purpose
-------------------
TestInitServer | Tests if the server could be initialised for the tests
TestNoCertDir | Tests if the server could be initialised without an SSL certificate
TestInvalidCredentials | Tests if a connection can be made to the endpoints using the credentials *fail* and *test*, that differ from the hardcoded ones
TestDialerFail | Tests that the dialer fails to connect to an invalid address
TestFailedUpgrade | Tests if an invalid websocket connection can be made
TestRoundtrip | Tests a full message roundtrip, simulating sending/receiving a message and sending/receiving it back, and checking the integrity of the message after the trip

### Coverage
As reported by [coveralls.io](https://coveralls.io/github/Javivi/ws-go?branch=master)

As reported by *go test -cover*

Service | Coverage
------------------
msgqueue | 82%
publisher | 77%
subscriber | 75%


# Continuous integration
### Travis CI
After a commit is made, a Travis CI build is triggered. Travis checks that all the .go files have gone through **gofmt**, **go vet**, **golint** and **go test**.

The Travis CI configuration file can be found [here](https://github.com/Javivi/ws-go/blob/master/.travis.yml)

### Coveralls
If the Travis CI build succeeds, the coverage results are sent to Coveralls for easy visualization and to help keep track of coverage.

# Deployment
### Docker
Dockerfiles are provided to help with the creation of images. Due to limitations on docker, the server.crt and .key files have to be moved to the working directory before creating an image.

The microservices are hardcoded to listen to the ports 8080, 8081, 8082, and during the tests another two servers listen to 8089 and 8999



# End notes about the project
- A mutex is used on the subscriber because maps are not thread safe
- golang test -race helped me diagnose that
- initServer() can receive a go channel as a parameter to indicate that the server is up and running. This is used for the tests
- popMessages() can also receive a go channel to indicate that something went wrong while reading a message to the go routine that is running it. This was also originally added for the tests but is not of much use now
- There's no timeout, closing or retry mechanism for the websockets, I did consider that out of scope for this project, but I had had to implement it, the websocket library that I used supported Ping/Pong Handlers to keep connections alive, and I did [a small test](https://github.com/Javivi/ws-go/blob/master/subscriber/subscriber_test.go#L127) simulating [a simple close handshake](https://github.com/Javivi/ws-go/blob/master/subscriber/subscriber_test.go#L244)
- Only 1 concurrent publisher/subscriber is correctly handled on the msgqueue. Due to it being a naive and simple implementation no mechanism of control was implemented and concurrency may have strange behaviour
- Due to that, there's no possible scalability or load balance. If multiple subscribers were possible, a simple load balance consisting on topic discrimination could be easily done, and if multiple publishers were possible, the clients pushing the data could select one of the available one randomly or depending on other factors