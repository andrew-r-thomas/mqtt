package mqtt

import (
	"log"
	"net"
	"sync"

	"github.com/andrew-r-thomas/mqtt/packets"
)

type Server struct {
	addr string

	bp *BufPool
	fp *FHPool

	tribes *sync.Map
}

const DefaultBufSize = 1024

func NewServer(addr string) Server {
	tribes := new(sync.Map)
	bp := NewBufPool(100, DefaultBufSize)
	fp := NewFHPool(100)

	return Server{
		addr: addr,

		bp: &bp,
		fp: &fp,

		tribes: tribes,
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
	log.Printf("%s: connected\n", connect.Id.String())

	readChan := make(chan Packet, 100)
	writeChan := make(chan []byte, 100)
	go s.readPump(conn, readChan, connect.Id.String())
	go s.writePump(conn, writeChan, connect.Id.String())

	for p := range readChan {
		offset := len(p.buf) - int(p.fh.RemLen)

		switch p.fh.Pt {
		case packets.PINGREQ:
			clear(p.buf)
			p.buf[0] = 0b11010000
			p.buf[1] = 0
			log.Printf("%s: sending ping resp\n", connect.Id.String())
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
				log.Fatalf("%s: error decoding subscribe packet: %v\n", connect.Id.String(), err)
			} else {
				log.Printf("%s: decoded subscribe\n", connect.Id.String())
			}
			suback := packets.Suback{}
			suback.Zero()

			for _, filter := range sub.TopicFilters {
				var tribeChan = make(chan TribeMsg, 100)
				val, ok := s.tribes.LoadOrStore(
					filter.Filter.String(),
					tribeChan,
				)
				if !ok {
					go StartTribeManager(tribeChan)
				} else {
					tribeChan = val.(chan TribeMsg)
				}
				tribeChan <- TribeMsg{
					MsgType:  AddMember,
					ClientId: connect.Id.String(),
					MsgData: AddMemberMsg{
						Sender: writeChan,
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

			packets.DecodePublish(
				p.fh,
				&pub,
				&props,
				p.buf[offset:],
			)

			val, _ := s.tribes.Load(pub.Topic.String())
			tribeChan := val.(chan TribeMsg)
			tribeChan <- TribeMsg{
				MsgType:  SendMsg,
				ClientId: connect.Id.String(),
				MsgData: SendMsgMsg{
					Data: p.buf,
				},
			}
		case packets.DISCONNECT:
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

func (s *Server) readPump(conn net.Conn, send chan<- Packet, id string) {
	for {
		fh := s.fp.GetFH()
		buf := s.bp.GetBuf()

		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("error reading from %s conn! %v\n", id, err)
		}
		offset := packets.DecodeFixedHeader(&fh, buf)
		if offset == -1 {
			log.Fatalf("error decoding fixed header from %s\n", id)
		}

		for n < int(fh.RemLen)+offset {
			// didn't read enough
			b := s.bp.GetBuf()

			nn, err := conn.Read(b)
			if err != nil {
				log.Fatalf(
					"error reading from %s conn: %v\n",
					id,
					err,
				)
			}

			buf = append(buf, b...)
			n += nn

			s.bp.ReturnBuf(b)
		}

		send <- Packet{fh: &fh, buf: buf[:n]}

		if fh.Pt == packets.DISCONNECT {
			// TODO:
			return
		}
	}
}

func (s *Server) writePump(conn net.Conn, recv <-chan []byte, id string) {
	for buf := range recv {
		log.Printf("%s: writing buf\n", id)
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
