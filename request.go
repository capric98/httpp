package httpp

import (
	"context"
	"net"
)

// Request illustrates a request.
type Request struct {
	host string
	clt  net.Conn

	predata []byte

	ctx    context.Context
	cancel context.CancelFunc
}
