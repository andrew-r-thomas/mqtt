package mqtt

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/andrew-r-thomas/mqtt/packets"
)

const KB = 1024
const MB = 1024 * KB
const DefaultBufSize = KB

type Server struct {
	bp BufPool
	fp FHPool

	clientsLock sync.RWMutex
	clients     map[string]Session

	topicTrie TopicTrie

	maxPacketSize int
	connDeadline  time.Duration

	ctx context.Context
}

// TODO: instantiate from persistent storage
func NewServer() Server {
	bp := NewBufPool(100, DefaultBufSize)
	fp := NewFHPool(100)

	return Server{
		bp: bp,
		fp: fp,

		clientsLock: sync.RWMutex{},
		clients:     make(map[string]Session, 8),

		topicTrie: NewTopicTrie(),

		maxPacketSize: 4 * KB,
	}
}

func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	// wait for connect packet
	conn.SetDeadline(time.Now().Add(time.Second))
	buf := s.bp.GetBuf()
	n, err := conn.Read(buf)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
		} else {
			log.Fatalf(
				"unexpected error setting up conn: %v\n",
				err,
			)
		}
	}
	// TODO: read until n is enough for fixedheader len
	fh := s.fp.GetFH()
	off := packets.DecodeFixedHeader(&fh, buf)
	if off == -1 {
		// TODO:
		log.Fatalf("error decoding fixed header\n")
	}
	for n < off+int(fh.RemLen) {
		// TODO: validate that the len isnt crazy
		b := s.bp.GetBuf()
		nn, err := conn.Read(b)
		if err != nil {
			// TODO:
		}
		buf = append(buf, b...)
		s.bp.ReturnBuf(b)
		n += nn
	}

	ctx, cancel := context.WithCancel(s.ctx)
	readChan := make(chan Packet, 100)
	writeChan := make(chan []byte, 100)

	go s.readPump(conn, ctx, readChan)
	go s.writePump(conn, ctx, writeChan)

	select {
	case packet := <-readChan:
		if packet.fh.Pt != packets.CONNECT {
			// TODO:
		}
		// TODO: do all the setup stuff
	case <-time.After(time.Second):
		cancel()
		return
	}
	// s.clientsLock.Lock()
	// s.clients[connect.Id.String()] = Session{
	// 	writeChan: writeChan,
	// }
	// s.clientsLock.Unlock()

	for p := range readChan {
		offset := len(p.buf) - int(p.fh.RemLen)

		switch p.fh.Pt {
		case packets.PINGREQ:
			clear(p.buf)
			p.buf[0] = 0b11010000
			p.buf[1] = 0
			writeChan <- p.buf[:2]
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
					connect.Id.String(),
					err,
				)
			}

			suback := packets.Suback{}
			suback.Zero()
			suback.PacketId = sub.PackedId

			for _, filter := range sub.TopicFilters {
				// TODO: filter cleaning
				sub := strings.Split(filter.Filter.String(), "/")
				s.topicTrie.AddSubscription(sub, connect.Id.String())
				suback.ReasonCodes = append(
					suback.ReasonCodes, 0,
				)
			}

			props.Zero()
			clear(p.buf)

			scratch := s.bp.GetBuf()
			i := packets.EncodeSuback(
				&suback,
				&props,
				p.buf,
				scratch,
			)
			s.bp.ReturnBuf(scratch)

			writeChan <- p.buf[:i]
		case packets.PUBLISH:
			log.Printf("got pub\n")
			props := packets.Properties{}
			props.Zero()
			pub := packets.Publish{}
			pub.Zero()

			err := packets.DecodePublish(
				p.fh,
				&pub,
				&props,
				p.buf[offset:],
			)
			if err != nil {
				log.Fatalf(
					"%s: error decoding pub: %v\n",
					connect.Id.String(),
					err,
				)
			}
			// TODO: topic cleaning
			topic := strings.Split(pub.Topic.String(), "/")
			matches := s.topicTrie.FindMatches(topic)
			s.clientsLock.RLock()
			for _, match := range matches {
				log.Printf("sending pub to %s\n", match)
				s.clients[match].writeChan <- p.buf
			}
			s.clientsLock.RUnlock()
		case packets.DISCONNECT:
			log.Printf("%s: disconnected\n", connect.Id.String())

			// remove subs
			s.topicTrie.RemoveSubs(connect.Id.String())

			// remove from clients
			s.clientsLock.Lock()
			delete(s.clients, connect.Id.String())
			s.clientsLock.Unlock()

			// stop writePump
			close(writeChan)

			// yeet on outa here
			return
		default:
			log.Fatalf(
				"invalid packet type from %s: %s\n",
				connect.Id.String(),
				p.fh.Pt.String(),
			)
		}
	}
}

// TODO: make all these errors graceful
func (s *Server) readPump(
	conn net.Conn,
	ctx context.Context,
	send chan<- Packet,
) {
	buf := make([]byte, s.maxPacketSize)
	accum := 0

pump:
	for {
		select {
		case <-ctx.Done():
			close(send)
			return
		default:
			// read some data into a buffer
			n, err := conn.Read(buf[accum:])
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

				fh := s.fp.GetFH()

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
					s.fp.ReturnFH(fh)
					continue pump
				}

				if offset+int(fh.RemLen) > accum {
					// PERF: this means we're doing extra fh decoding
					continue pump
				}

				b := s.bp.GetBuf()
				nn := copy(b, buf[:offset+int(fh.RemLen)])
				if nn < offset+int(fh.RemLen) {
					b = append(
						b,
						buf[nn:offset+int(fh.RemLen)]...,
					)
				}

				send <- Packet{
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

func (s *Server) writePump(
	conn net.Conn,
	ctx context.Context,
	recv <-chan []byte,
) {
	for {
		select {
		case <-ctx.Done():
		case buf := <-recv:
			n, err := conn.Write(buf)
			if err != nil {
				log.Fatalf("error writting to conn: %v\n", err)
			}
			if n != len(buf) {
				log.Fatalf("did not write full buf\n")
			}
			s.bp.ReturnBuf(buf)
		}
	}
}

func (s *Server) setupClient(
	conn net.Conn,
) (packets.Connect, packets.Properties) {
	// timeout and wait for CONNECT packet
	buf := s.bp.GetBuf()
	fh := s.fp.GetFH()

	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("error reading from conn: %v\n", err)
	}
	offset := packets.DecodeFixedHeader(&fh, buf)
	if offset == -1 {
		log.Fatalf("error decoding fixed header\n")
	}
	if fh.Pt != packets.CONNECT {
		log.Fatalf("client sent packet other than connect first\n")
	}
	for n < int(fh.RemLen)+offset {
		b := s.bp.GetBuf()
		nn, err := conn.Read(b)
		if err != nil {
			log.Fatalf("error readind from conn: %v\n", err)
		}
		buf = append(buf, b...)
		n += nn
		s.bp.ReturnBuf(b)
	}

	connect := packets.Connect{}
	connect.Zero()
	willProps := packets.Properties{}
	willProps.Zero()
	props := packets.Properties{}
	props.Zero()
	err = packets.DecodeConnect(
		&connect,
		buf[offset:n],
		&props,
		&willProps,
	)
	if err != nil {
		log.Fatalf("ahh! error decoding connect:\n%v", err)
	}

	// set up buf and packets for connack
	clear(buf)
	props.Zero()
	connack := packets.Connack{}
	connack.Zero()
	scratch := s.bp.GetBuf()

	// encode connack packet and write to connection
	l := packets.EncodeConnack(&connack, &props, buf, scratch)
	n, err = conn.Write(buf[:l])
	if err != nil {
		log.Fatalf("ahh error writing to conn: %v\n", err)
	}

	// cleanup
	s.bp.ReturnBuf(scratch)
	s.bp.ReturnBuf(buf)

	return connect, willProps

}

type Packet struct {
	fh  *packets.FixedHeader
	buf []byte // this is the whole buffer, including the fixed header
}

type Session struct {
	pendingMsgs [][]byte
	unackMsgs   [][]byte
	willMsg     []byte
	willDelay   uint32
	sessionEnd  time.Time
}
