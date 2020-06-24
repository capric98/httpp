package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/capric98/httpp"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	srv := httpp.NewServer("127.0.0.1:8080", false, 10*time.Second)
	srv.Listen()
	for {
		req := srv.Accept()
		if req != nil {
			go func(r *httpp.Request) {
				conn, err := net.DialTimeout("tcp", r.Host(), 10*time.Second)
				if err != nil {
					r.Fail(err)
				} else {
					r.Success(conn)
				}
			}(req)
		}
	}
}
