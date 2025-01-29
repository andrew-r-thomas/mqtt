package mqtt

import (
	"log"
	"net"
	"strings"

	"github.com/andrew-r-thomas/mqtt/packets"
)

type Server struct {
	addr string

	tt TopicTree

	pubChan    chan<- PubMsg
	subChan    chan<- SubMsg
	unSubChan  chan<- UnSubMsg
	addCliChan chan<- AddCliMsg
	remCliChan chan<- RemCliMsg

	bp BufPool
}

func NewServer(addr string) Server {
	pubChan := make(chan PubMsg, 10)
	subChan := make(chan SubMsg, 10)
	unSubChan := make(chan UnSubMsg, 10)
	addCliChan := make(chan AddCliMsg, 10)
	remCliChan := make(chan RemCliMsg, 10)

	tt := NewTopicTree(
		pubChan,
		subChan,
		addCliChan,
		unSubChan,
		remCliChan,
	)

	bp := NewBufPool(1000, 1024) // 1mb

	return Server{
		addr: addr,

		tt: tt,

		pubChan:    pubChan,
		subChan:    subChan,
		unSubChan:  unSubChan,
		addCliChan: addCliChan,
		remCliChan: remCliChan,

		bp: bp,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go s.tt.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go handleClient(
			conn,

			s.pubChan,
			s.subChan,
			s.unSubChan,
			s.addCliChan,
			s.remCliChan,

			&s.bp,
		)
	}
}

func handleClient(
	conn net.Conn,

	pubChan chan<- PubMsg,
	subChan chan<- SubMsg,
	unSubChan chan<- UnSubMsg,
	addCliChan chan<- AddCliMsg,
	remCliChan chan<- RemCliMsg,

	bp *BufPool,
) {
	pl := packets.NewPacketLib()
	pl.Zero()

	connect, _ := setupConnection(
		conn,
		&pl.Properties,
		bp,
	)

	sender := make(chan []byte, 10)
	addCliChan <- AddCliMsg{ClientId: connect.Id, Sender: sender}

	readChan := make(chan Packet, 10)
	go readPump(conn, bp, readChan)

	for {
		select {
		case pub := <-sender:
			conn.Write(pub)
			bp.ReturnBuf(pub)
		case packet := <-readChan:
			offset := len(packet.buf) - int(packet.fh.RemLen)
			switch packet.fh.Pt {
			case packets.PUBLISH:
				pl.Properties.Zero()
				packets.DecodePublish(
					&packet.fh,
					&pl.Publish,
					&pl.Properties,
					packet.buf[offset:],
				)
				pubChan <- PubMsg{
					Topic: pl.Publish.Topic,
					Msg:   packet.buf,
				}
			case packets.PINGREQ:
				clear(packet.buf)
				packet.buf[0] = 0b11010000
				packet.buf[1] = 0
				_, err := conn.Write(packet.buf[:2])
				if err != nil {
					log.Fatalf("ahhh! %v\n", err)
				}
				bp.ReturnBuf(packet.buf)
			case packets.SUBSCRIBE:
				pl.Properties.Zero()
				pl.Subscribe.Zero()
				packets.DecodeSubscribe(
					&pl.Subscribe,
					&pl.Properties,
					packet.buf[offset:],
				)

				pl.Suback.Zero()
				tfs := [][]string{}
				for _, filter := range pl.Subscribe.TopicFilters {
					// TODO: validate filter in here
					pl.Suback.ReasonCodes = append(
						pl.Suback.ReasonCodes, 0,
					)
					tfs = append(
						tfs,
						strings.Split(
							filter.Filter,
							"/",
						),
					)
				}
				subChan <- SubMsg{
					ClientId:     connect.Id,
					TopicFilters: tfs,
				}

				pl.Suback.PacketId = pl.Subscribe.PackedId
				pl.Properties.Zero()
				clear(packet.buf)
				scratch := bp.GetBuf()
				i := packets.EncodeSuback(
					&pl.Suback,
					&pl.Properties,
					packet.buf,
					scratch,
				)
				_, err := conn.Write(packet.buf[:i])
				if err != nil {
					log.Fatalf(
						"error writing suback: %v\n",
						err,
					)
				}
				bp.ReturnBuf(scratch)
				bp.ReturnBuf(packet.buf)
			case packets.UNSUBSCRIBE:
				for _, b := range packet.buf {
					log.Printf("%08b\n", b)
				}

				pl.Properties.Zero()
				pl.Unsubscribe.Zero()
				packets.DecodeUnsubscribe(
					&pl.Unsubscribe,
					&pl.Properties,
					packet.buf[offset:],
				)

				pl.Unsuback.Zero()
				tfs := [][]string{}
				for _, filter := range pl.Unsubscribe.TopciFilters {
					pl.Unsuback.ReasonCodes = append(pl.Unsuback.ReasonCodes, 0)
					tfs = append(tfs, strings.Split(filter, "/"))
				}
				unSubChan <- UnSubMsg{
					ClientId:     connect.Id,
					TopicFilters: tfs,
				}

				pl.Unsuback.PacketId = pl.Unsubscribe.PacketId
				pl.Properties.Zero()
				clear(packet.buf)
				scratch := bp.GetBuf()
				i := packets.EncodeUnsuback(&pl.Unsuback, &pl.Properties, packet.buf, scratch)
				_, err := conn.Write(packet.buf[:i])
				if err != nil {
					log.Fatalf(
						"error writing unsuback: %v\n",
						err,
					)
				}
				bp.ReturnBuf(scratch)
				bp.ReturnBuf(packet.buf)
			default:
				log.Printf("bad packet\n")
				bp.ReturnBuf(packet.buf)
			}
		}
	}
}

func setupConnection(
	conn net.Conn,
	props *packets.Properties,
	bp *BufPool,
) (packets.Connect, packets.Properties) {
	buf := bp.GetBuf()
	scratch := bp.GetBuf()
	// wait for connect packet
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("ahhh! %v", err)
	}

	fh := packets.FixedHeader{}
	fh.Zero()

	offset, err := packets.DecodeFixedHeader(&fh, buf)
	if err != nil {
		log.Fatalf("ahhh! %v", err)
	}
	if n < int(fh.RemLen)+offset {
		log.Fatalf("ahhh! didn't read enough!")
	}

	connect := packets.Connect{}
	connect.Zero()
	willProps := packets.Properties{}
	willProps.Zero()
	err = packets.DecodeConnect(
		&connect,
		buf[offset:n],
		props,
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

	// encode connack packet and write to connection
	bl := packets.EncodeConnack(&connack, props, buf, scratch)
	n, err = conn.Write(buf[:bl])
	if err != nil {
		log.Fatalf("ahh error writing to conn: %v\n", err)
	}

	bp.ReturnBuf(buf)
	bp.ReturnBuf(scratch)

	return connect, willProps
}

type Packet struct {
	fh  packets.FixedHeader
	buf []byte // this is the whole buffer, including the fixed header
}

func readPump(conn net.Conn, bp *BufPool, sender chan<- Packet) {
	fh := packets.FixedHeader{}

	for {
		fh.Zero()
		buf := bp.GetBuf()

		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("error reading from conn! %v\n", err)
		}

		offset, err := packets.DecodeFixedHeader(&fh, buf)

		for n < int(fh.RemLen)+offset {
			// didn't read enough
			b := bp.GetBuf()
			defer bp.ReturnBuf(b)

			nn, err := conn.Read(b)
			if err != nil {
				log.Fatalf("error reading from conn: %v\n", err)
			}

			n += nn
			buf = append(buf, b...)
		}

		sender <- Packet{fh: fh, buf: buf[:n]}
	}
}
