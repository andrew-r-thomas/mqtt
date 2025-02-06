package mqtt

import (
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/andrew-r-thomas/mqtt/packets"
)

type Server struct {
	addr string

	bp *BufPool
	fp *FHPool

	tribesLock sync.RWMutex
	tribes     map[string]chan<- TribeMsg
}

const DefaultBufSize = 1024

func NewServer(addr string) Server {
	bp := NewBufPool(100, DefaultBufSize)
	fp := NewFHPool(100)

	tribes := make(map[string]chan<- TribeMsg)

	return Server{
		addr: addr,

		bp: &bp,
		fp: &fp,

		tribes:     tribes,
		tribesLock: sync.RWMutex{},
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	connect, _ := s.setupClient(conn)

	readChan := make(chan Packet, 100)
	writeChan := make(chan []byte, 100)
	active := atomic.Bool{}
	active.Store(true)
	go s.readPump(conn, readChan, connect.Id.String())
	go s.writePump(conn, writeChan, connect.Id.String())

	for p := range readChan {
		offset := len(p.buf) - int(p.fh.RemLen)

		switch p.fh.Pt {
		case packets.PINGREQ:
			clear(p.buf)
			p.buf[0] = 0b11010000
			p.buf[1] = 0
			writeChan <- p.buf[:2]
		case packets.SUBSCRIBE:
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

			for _, filter := range sub.TopicFilters {
				s.tribesLock.RLock()
				tribeChan, ok := s.tribes[filter.Filter.String()]
				s.tribesLock.RUnlock()

				if !ok {
					tc := make(chan TribeMsg, 10)
					go StartTribeManager(tc, s.bp)
					s.tribesLock.Lock()
					tribeChan = tc
					s.tribes[filter.Filter.String()] = tribeChan
					s.tribesLock.Unlock()
				}
				tribeChan <- TribeMsg{
					MsgType:  AddMember,
					ClientId: connect.Id.String(),
					MsgData: AddMemberMsg{
						Sender: Sender{
							c:       writeChan,
							live:    &active,
							noLocal: filter.NoLocal,
						},
					},
				}
				suback.ReasonCodes = append(
					suback.ReasonCodes, 0,
				)
			}
			suback.PacketId = sub.PackedId

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
			s.tribesLock.RLock()
			tribeChan := s.tribes[pub.Topic.String()]
			s.tribesLock.RUnlock()
			tribeChan <- TribeMsg{
				MsgType:  SendMsg,
				ClientId: connect.Id.String(),
				MsgData: SendMsgMsg{
					Data: p.buf,
				},
			}
		case packets.DISCONNECT:
			log.Printf("%s: disconnected\n", connect.Id.String())
			active.Store(false)
			close(writeChan)
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
func (s *Server) readPump(conn net.Conn, send chan<- Packet, id string) {
	buf := make([]byte, 1024)
	accum := 0

pump:
	for {
		if accum >= 1024 {
			log.Fatalf("ahh! read pump buf is too full\n")
		}

		// read some data into a buffer
		n, err := conn.Read(buf[accum:])
		if err != nil {
			log.Fatalf(
				"%s: error reading from conn: %v\n",
				id,
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
						"%s: error decoding fixed header\n",
						id,
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
				b = append(b, buf[nn:offset+int(fh.RemLen)]...)
			}

			send <- Packet{
				fh:  &fh,
				buf: b[:offset+int(fh.RemLen)],
			}

			if fh.Pt == packets.DISCONNECT {
				return
			}

			clear(buf[:offset+int(fh.RemLen)])
			copy(buf, buf[offset+int(fh.RemLen):])
			accum -= offset + int(fh.RemLen)
		}
	}
}

func (s *Server) writePump(conn net.Conn, recv <-chan []byte, id string) {
	for buf := range recv {
		n, err := conn.Write(buf)
		if err != nil {
			log.Fatalf("error writting to %s conn: %v\n", id, err)
		}
		if n != len(buf) {
			log.Fatalf("did not write full buf to %s\n", id)
		}
		s.bp.ReturnBuf(buf)
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
