package httpp

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// Request illustrates a request.
type Request struct {
	typ  string
	host string
	clt  net.Conn

	predata []byte

	ctx    context.Context
	cancel context.CancelFunc
}

// Type returns request's type.
func (req *Request) Type() string {
	return req.typ
}

// Host returns request's target host.
func (req *Request) Host() string {
	return req.host
}

// ClientAddr returns client's address.
func (req *Request) ClientAddr() net.Addr {
	return req.clt.RemoteAddr()
}

// Success approves the Request with an interface
// which can be converted to net.Conn.
func (req *Request) Success(conn net.Conn) {
	// Cancel deadline.
	req.clt.SetDeadline(time.Time{})

	defer func() {
		if p := recover(); p != nil {
			conn.Close()
			req.Fail(fmt.Errorf("request from %v - %v", req.ClientAddr(), p))
		}
	}()

	if req.typ == CONNECT {
		_, e := req.clt.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if e != nil {
			panic(e)
		}
	} else {
		_, e := conn.Write(req.predata)
		if e != nil {
			panic(e)
		}
	}

	go func() {
		_, _ = io.Copy(req.clt, conn)
		conn.Close()
		req.cancel()
	}()
	go func() {
		_, _ = io.Copy(conn, req.clt)
		conn.Close()
		req.cancel()
	}()
}

// Fail denies the Request with a given error.
func (req *Request) Fail(e error) {
	_, _ = req.clt.Write([]byte(fmt.Sprintf("HTTP/1.1 500 Internal Server Error\r\n%v\r\n\r\n", e)))
	req.cancel()
}

func (req *Request) watch() {
	go func() {
		<-req.ctx.Done()
		_ = req.clt.Close()
	}()
}
