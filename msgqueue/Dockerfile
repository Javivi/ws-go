FROM golang:1.9.2

# Remember to copy server.crt and server.key to the working directory
# We can't add them from parent directory
ADD . ./

RUN go get github.com/gorilla/websocket
RUN go build msgqueue.go

ENTRYPOINT ./msgqueue

EXPOSE 8080