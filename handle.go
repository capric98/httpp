package httpp

import (
	"log"
	"net"
	"strings"
	"time"
)

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if p := recover(); p != nil {
			conn.Close()
		}
	}()
	s.setTimeout(conn)

	method := make([]byte, 4)
	if _, e := conn.Read(method); e != nil {
		panic(e)
	}
	if strings.ToUpper(string(method)) == "GET " {
		s.handleGET(conn)
	} else {
		if _, e := conn.Read(method); e != nil {
			panic(e)
		}
		s.handleCONNECT(conn)
	}
}

func (s *Server) handleGET(conn net.Conn) {
	_ = readRawLine(conn)
}
func (s *Server) handleCONNECT(conn net.Conn) {
	_ = readRawLine(conn)
	readEnd(conn)
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
		log.Println("readRawLine=", line)
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
		log.Println("readLine key=", key, "value=", value)
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
