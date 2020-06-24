package httpp

import (
	"context"
	"log"
	"net"
	"strings"
	"time"
)

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if p := recover(); p != nil {
			log.Println(p)
			conn.Close()
		}
	}()
	s.setTimeout(conn)

	var method []byte
	one := make([]byte, 1)
	for {
		if _, e := conn.Read(one); e != nil {
			panic(e)
		}
		if one[0] == ' ' {
			break
		}
		method = append(method, one[0])
	}
	if strings.ToUpper(string(method)) == "CONNECT" {
		if _, e := conn.Read(method); e != nil {
			panic(e)
		}
		s.handleCONNECT(conn)
	} else {
		s.handleGeneral(conn, string(method))
	}
}

func (s *Server) handleGeneral(conn net.Conn, method string) {
	path := readRawLine(conn)
	dp := strings.Index(path, ":") // scheme
	path = path[dp+3:]
	dp = strings.Index(path, "/") // path
	path = path[dp:]
	dp = strings.Index(path, " ") // HTTP/1.x
	path = path[:dp]

	sb := strings.Builder{}
	_, _ = sb.WriteString(method)
	_, _ = sb.WriteString(" ")
	_, _ = sb.WriteString(path)
	_, _ = sb.WriteString(" HTTP/1.1\r\n")

	req := &Request{typ: method, clt: conn}
	req.ctx, req.cancel = context.WithCancel(s.ctx)

	var atype, credential string
	key, value := readLine(conn)
	for key != `\r\n` && value != `\r\n` {
		switch strings.ToLower(key) {
		case "host":
			if strings.Index(value, ":") == -1 {
				req.host = value + ":80"
			} else {
				req.host = value
			}
			_, _ = sb.WriteString(key)
			_, _ = sb.WriteString(": ")
			_, _ = sb.WriteString(value)
			_, _ = sb.WriteString("\r\n")
		case "proxy-authorization":
			dp = strings.Index(value, " ")
			// Let it panic if not a valid credential.
			atype = strings.ToLower(value[:dp])
			credential = value[dp+1:]
		default:
			_, _ = sb.WriteString(key)
			_, _ = sb.WriteString(": ")
			_, _ = sb.WriteString(value)
			_, _ = sb.WriteString("\r\n")
		}
		key, value = readLine(conn)
	}
	_, _ = sb.WriteString("\r\n")
	req.predata = []byte(sb.String())

	if s.forceAuth {
		if credential == "" {
			authrequired(conn)
			conn.Close()
			return
		}
		s.amu.RLock()
		if !s.auths[atype].Cred(credential) {
			s.amu.RUnlock()
			panic("auth: invalid credential")
		}
		s.amu.RUnlock()
	}
	s.reqs <- req
}
func (s *Server) handleCONNECT(conn net.Conn) {
	_ = readRawLine(conn)

	req := &Request{typ: CONNECT, clt: conn}
	req.ctx, req.cancel = context.WithCancel(s.ctx)

	var atype, credential string
	key, value := readLine(conn)
	for key != `\r\n` && value != `\r\n` {
		switch strings.ToLower(key) {
		case "host":
			req.host = value
		case "proxy-authorization":
			log.Println(key, value)
			dp := strings.Index(value, " ")
			// Let it panic if not a valid credential.
			atype = strings.ToLower(value[:dp])
			credential = value[dp+1:]
		default:
		}
		key, value = readLine(conn)
	}
	if s.forceAuth {
		if credential == "" {
			authrequired(conn)
			conn.Close()
			return
		}
		s.amu.RLock()
		if !s.auths[atype].Cred(credential) {
			s.amu.RUnlock()
			panic("auth: invalid credential")
		}
		s.amu.RUnlock()
	}
	s.reqs <- req
}

func readEnd(conn net.Conn) {
	for {
		key, value := readLine(conn)
		if key == `\r\n` && value == `\r\n` {
			key, value = readLine(conn)
			if key == `\r\n` && value == `\r\n` {
				return
			}
		}
	}
}

func readRawLine(conn net.Conn) (line string) {
	sb := strings.Builder{}
	defer func() {
		line = sb.String()
		// log.Println("readRawLine=", line)
	}()

	var e error
	one := make([]byte, 1)
	_, e = conn.Read(one)
	if e != nil {
		panic(e)
	}

	for {
		for one[0] != '\r' {
			_ = sb.WriteByte(one[0])
			_, e = conn.Read(one)
			if e != nil {
				panic(e)
			}
		}
		tone := make([]byte, 1)
		_, e = conn.Read(tone)
		if e != nil {
			panic(e)
		}
		if tone[0] == '\n' {
			// \r\n
			return
		}
		_ = sb.WriteByte(one[0])
		_ = sb.WriteByte(tone[0])
		_, e = conn.Read(one)
		if e != nil {
			panic(e)
		}
	}
}

func readLine(conn net.Conn) (key, value string) {
	ksb := strings.Builder{}
	vsb := strings.Builder{}
	defer func() {
		key = ksb.String()
		value = vsb.String()
		// log.Println("readLine key=", key, "value=", value)
	}()

	var e error
	one := make([]byte, 1)

	_, e = conn.Read(one)
	if e != nil {
		panic(e)
	}
	if one[0] == '\r' {
		_, e = conn.Read(one)
		if e != nil {
			panic(e)
		}
		if one[0] == '\n' && ksb.String() == "" {
			ksb.WriteString(`\r\n`)
			vsb.WriteString(`\r\n`)
			return
		}
	}

	// Read key.
	for {
		for one[0] != ':' && one[0] != '\r' {
			_ = ksb.WriteByte(one[0])
			_, e = conn.Read(one)
			if e != nil {
				panic(e)
			}
		}
		if one[0] == ':' {
			break
		} else {
			tone := make([]byte, 1)
			_, e = conn.Read(tone)
			if e != nil {
				panic(e)
			}
			if tone[0] == '\n' {
				// \r\n
				return
			}
			if tone[0] == ':' {
				_ = ksb.WriteByte(one[0])
				break
			}
			_ = ksb.WriteByte(one[0])
			_ = ksb.WriteByte(tone[0])
			_, e = conn.Read(one)
			if e != nil {
				panic(e)
			}
		}
	}

	// Read space.
	_, e = conn.Read(one)
	if e != nil {
		panic(e)
	}
	_, e = conn.Read(one)
	if e != nil {
		panic(e)
	}

	// Read value.
	for {
		for one[0] != '\r' {
			_ = vsb.WriteByte(one[0])
			_, e = conn.Read(one)
			if e != nil {
				panic(e)
			}
		}
		tone := make([]byte, 1)
		_, e = conn.Read(tone)
		if e != nil {
			panic(e)
		}
		if tone[0] == '\n' {
			// \r\n
			return
		}
		_ = vsb.WriteByte(one[0])
		_ = vsb.WriteByte(tone[0])
		_, e = conn.Read(one)
		if e != nil {
			panic(e)
		}
	}
}

func (s *Server) setTimeout(conn net.Conn) {
	if s.timeout == 0 {
		cancelTimeout(conn)
		return
	}
	conn.SetDeadline(time.Now().Add(s.timeout))
}

func cancelTimeout(conn net.Conn) {
	conn.SetDeadline(time.Time{})
}
