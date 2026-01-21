package websocket

import (
	"io"
	"log"

	"github.com/mqverk/shlx/backend/session"
)

// PTYReader reads from PTY and broadcasts to all users
func PTYReader(sess *session.Session) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.PTY.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("PTY read error: %v", err)
			}
			return
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			sess.Broadcast(data)
		}
	}
}
