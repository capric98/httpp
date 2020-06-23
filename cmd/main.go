package main

import (
	"time"

	"github.com/capric98/httpp"
)

func main() {
	srv := httpp.NewServer("127.0.0.1:8080", false, 10*time.Second)
	srv.Listen()
	for {
		time.Sleep(time.Second)
	}
}
