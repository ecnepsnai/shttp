package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ecnepsnai/shttp"
)

func main() {
	id, err := shttp.NewIdentity()
	if err != nil {
		panic(err)
	}

	l, err := shttp.SetupListener(shttp.ListenOptions{
		Address:  "127.0.0.1:8080",
		Identity: id.Signer(),
	}, func(conn *shttp.Connection) {
		defer conn.Close()
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid request: %s\n", err.Error())
			return
		}
		reply := []byte("Hello, world!")
		resp := http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.0",
			ProtoMajor:    1,
			ProtoMinor:    0,
			Body:          io.NopCloser(bytes.NewBuffer(reply)),
			ContentLength: int64(len(reply)),
			Request:       req,
		}
		resp.Write(conn)
	})
	if err != nil {
		panic(err)
	}
	l.Accept()
}
