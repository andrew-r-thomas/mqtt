package mqtt

import (
	"net"
	"time"

	"github.com/andrew-r-thomas/mqtt/packets"
)

// the Client system is a child of the Server system, responsible for handling
// a single connection
type Client struct {
	server    *Server
	conn      net.Conn
	id        string
	session   Session
	writeChan chan []byte
}

// SetupClient is called by the Server system upon establishing a new network
// connection
func SetupClient(conn net.Conn, s *Server) (ch Client, err error) {
	// set connection deadline and wait for connect packet
	conn.SetDeadline(time.Now().Add(s.connDeadline))
	buf := s.bp.GetBuf()
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	fh := s.fp.GetFH()
	off := packets.DecodeFixedHeader(&fh, buf)
	if off == -1 {
		// TODO:
	}

	return
}

func (c *Client) Run() {
	readChan := make(chan Packet, 128)
	go c.readPump(readChan)
	go c.writePump()

	for packet := range readChan {
		// TODO:
	}
}
func (c *Client) readPump(readChan chan<- Packet) {

}
func (c *Client) writePump() {}
