package mqtt

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
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
	wg        sync.WaitGroup
	keepalive uint16
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

func (c *Client) Run(ctx context.Context) {
	c.server.wg.Add(1)

	ctx, cancel := context.WithCancel(ctx)
	readChan := make(chan Packet, 128)
	go c.readPump(ctx, readChan)
	go c.writePump(ctx)

	for {
		select {
		case <-ctx.Done():
			// TODO: graceful shutdown
			cancel()
			c.wg.Wait()
			c.server.wg.Done()
			return
		case p := <-readChan:
			c.handlePacket(p)
		}
	}
}

func (c *Client) readPump(
	ctx context.Context,
	readChan chan<- Packet,
) {
	c.wg.Add(1)
	buf := make([]byte, c.server.maxPacketSize)
	accum := 0

pump:
	for {
		select {
		case <-ctx.Done():
			close(readChan)
			c.wg.Done()
			return
		default:
			// read some data into a buffer
			n, err := c.conn.Read(buf[accum:])
			if err != nil {
				log.Fatalf(
					"error reading from conn: %v\n",
					err,
				)
			}
			accum += n

			// parse up all the packets from the buffer
			for {
				if accum < 2 {
					// not enough data to read a fixed header
					continue pump
				}

				fh := c.server.fp.GetFH()

				offset := packets.DecodeFixedHeader(
					&fh,
					buf,
				)
				if offset == -1 {
					if accum > 5 {
						// we have enough data for a full fixed header,
						// so this is a true error
						log.Fatalf(
							"error decoding fixed header\n",
						)
					}

					// we might not have read enough
					// for the full fixed header
					c.server.fp.ReturnFH(fh)
					continue pump
				}

				if offset+int(fh.RemLen) > accum {
					// PERF: this means we're doing extra fh decoding
					continue pump
				}

				b := c.server.bp.GetBuf()
				nn := copy(b, buf[:offset+int(fh.RemLen)])
				if nn < offset+int(fh.RemLen) {
					b = append(
						b,
						buf[nn:offset+int(fh.RemLen)]...,
					)
				}

				readChan <- Packet{
					fh:  &fh,
					buf: b[:offset+int(fh.RemLen)],
				}

				clear(buf[:offset+int(fh.RemLen)])
				copy(buf, buf[offset+int(fh.RemLen):])
				accum -= offset + int(fh.RemLen)
			}
		}
	}
}
func (c *Client) writePump(
	ctx context.Context,
) {
	c.wg.Add(1)

	for {
		select {
		case <-ctx.Done():
			// TODO:
			c.wg.Done()
			return
		case b := <-c.writeChan:
		}
	}
}

func (c *Client) handlePacket(p Packet) {
	switch p.fh.Pt {
	case packets.PINGREQ:
		clear(p.buf)
		p.buf[0] = 0b11010000
		p.buf[1] = 0
		c.writeChan <- p.buf[:2]
	case packets.SUBSCRIBE:
		log.Printf("got sub\n")
		props := packets.Properties{}
		props.Zero()
		sub := packets.Subscribe{}
		sub.Zero()

		err := packets.DecodeSubscribe(
			&sub,
			&props,
			p.buf[offset:],
		)
		if err != nil {
			log.Fatalf(
				"%s: error decoding subscribe packet: %v\n",
				c.id,
				err,
			)
		}

		suback := packets.Suback{}
		suback.Zero()
		suback.PacketId = sub.PackedId

		for _, filter := range sub.TopicFilters {
			// TODO: filter cleaning
			sub := strings.Split(filter.Filter.String(), "/")
			c.server.topicTrie.AddSubscription(sub, c.id)
			suback.ReasonCodes = append(
				suback.ReasonCodes, 0,
			)
		}

		props.Zero()
		clear(p.buf)

		scratch := c.server.bp.GetBuf()
		i := packets.EncodeSuback(
			&suback,
			&props,
			p.buf,
			scratch,
		)
		c.server.bp.ReturnBuf(scratch)

		c.writeChan <- p.buf[:i]
	case packets.PUBLISH:
		c.handlePublish(p)
	case packets.DISCONNECT:
		log.Printf("%s: disconnected\n", c.id)

		// remove subs
		c.server.topicTrie.RemoveSubs(c.id)

		// remove from clients
		c.server.clientsLock.Lock()
		delete(c.server.clients, c.id)
		c.server.clientsLock.Unlock()

		// stop writePump
		close(c.writeChan)

		// yeet on outa here
		return
	default:
		log.Fatalf(
			"invalid packet type from %s: %s\n",
			c.id,
			p.fh.Pt.String(),
		)
	}
}

func (c *Client) handlePublish(p Packet) {
	pub := packets.Publish{}
	props := packets.Properties{}
	pub.Zero()
	props.Zero()

	offset := len(p.buf) - int(p.fh.RemLen)

	err := packets.DecodePublish(
		p.fh,
		&pub,
		&props,
		p.buf[offset:],
	)
	if err != nil {
		// TODO:
	}

}
