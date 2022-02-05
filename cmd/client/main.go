package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ecnepsnai/shttp"
)

func main() {
	id, err := shttp.NewIdentity()
	if err != nil {
		panic(err)
	}

	c, err := shttp.Dial(shttp.DialOptions{
		Network:  "tcp",
		Address:  "127.0.0.1:8080",
		Identity: id.Signer(),
		Timeout:  10 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", "shttp://localhost:8080/", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", "shttpclient/1.0")

	if err := req.Write(c); err != nil {
		panic(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n%s\n", resp.Status, body)
}
