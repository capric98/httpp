package httpp

import (
	"encoding/base64"
	"net"
	"sync"
)

// Authenticator interface
type Authenticator interface {
	Type() string
	Cred(string) bool
}

// Basic Auth

// BasicAuth ...
type BasicAuth struct {
	creds map[string]bool
	rwmu  sync.RWMutex
}

// Type returns Proxy-Authorization type.
func (ba *BasicAuth) Type() string {
	return "basic"
}

// Cred validates a credential.
func (ba *BasicAuth) Cred(c string) (b bool) {
	ba.rwmu.RLock()
	b = ba.creds[c]
	ba.rwmu.RUnlock()
	return
}

// Add adds a credential.
// cred = string(USERNAME) + ":" + string(PASSWORD)
func (ba *BasicAuth) Add(c string) {
	c = base64.StdEncoding.EncodeToString([]byte(c))
	ba.rwmu.Lock()
	ba.creds[c] = true
	ba.rwmu.Unlock()
}

// Del deletes a credential.
// cred = string(USERNAME) + ":" + string(PASSWORD)
func (ba *BasicAuth) Del(c string) {
	c = base64.StdEncoding.EncodeToString([]byte(c))
	ba.rwmu.Lock()
	delete(ba.creds, c)
	ba.rwmu.Unlock()
}

func unauthorized(conn net.Conn) {
	_, _ = conn.Write([]byte("HTTP/1.1 401 Unauthorized\r\n\r\n"))
}

func authrequired(conn net.Conn) {
	_, _ = conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n\r\n"))
}
