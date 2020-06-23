package httpp

import (
	"context"
	"net"
	"sync"
	"time"
)

// Server illustrates a server.
type Server struct {
	addr      string
	forceAuth bool

	amu   sync.RWMutex
	auths map[string]Authenticator

	ctx     context.Context
	cancel  context.CancelFunc
	timeout time.Duration

	reqs chan Request
}

// NewServer news a Server.
func NewServer(addr string, forceAuth bool, timeout time.Duration) *Server {
	return &Server{
		addr:      addr,
		forceAuth: forceAuth,
		auths:     make(map[string]Authenticator),
		timeout:   timeout,
		reqs:      make(chan Request, 65535),
	}
}

// Listen listens the address and serves.
func (s *Server) Listen() (e error) {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	bctx, bcancel := context.WithCancel(context.Background())
	s.ctx = bctx
	s.cancel = func() {
		bcancel()
		l.Close()
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if e, ok := err.(*net.OpError); ok {
					// I cannot use internal.poll.ErrNetClosing since its an internal package.
					if e.Unwrap() != nil && e.Unwrap().Error() == "use of closed network connection" {
						// l was closed, quit this goroutine
						return
					}
				}
			} else {
				go s.handle(conn)
			}
		}
	}()

	return
}

// Stop stops the server.
// Stop a server before Listen() will cause panic.
func (s *Server) Stop() {
	s.cancel()
}

// SetAuth sets an authenticator, it will overwrite
// the authenticator with same type if exists.
func (s *Server) SetAuth(a Authenticator) {
	s.amu.Lock()
	s.auths[a.Type()] = a
	s.amu.Unlock()
}

// DelAuth deletes the authenticator with given type.
func (s *Server) DelAuth(typ string) {
	s.amu.Lock()
	delete(s.auths, typ)
	s.amu.Unlock()
}

// GetAuth gets the authenticator with given type.
// It will return a nil authenticator when not exist.
func (s *Server) GetAuth(typ string) (a Authenticator) {
	s.amu.Lock()
	a = s.auths[typ]
	s.amu.Unlock()
	return
}
