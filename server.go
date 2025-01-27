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

	bp := NewBufPool(1000, 1024)

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
	// set up packet lib and read buf
	pl := packets.NewPacketLib()
	pl.Zero()

	connect, _ := setupConnection(
		conn,
		&pl.FixedHeader,
		&pl.Properties,
		bp,
	)

	sender := make(chan []byte, 10)
	addCliChan <- AddCliMsg{ClientId: connect.Id, Sender: sender}

	readChan := make(chan []byte, 10)
	go readPackets(conn, bp, readChan)

	// wait for packets
	for {
		select {
		case pub := <-sender:
			log.Printf("got a pub\n")
			conn.Write(pub)
			bp.ReturnBuf(pub)
		case read := <-readChan:
			log.Printf("packet bytes: %v\n", read[:10])
			// read fixed header (and maybe more) in
			pl.FixedHeader.Zero()
			offset, err := packets.DecodeFixedHeader(&pl.FixedHeader, read)
			if err != nil {
				log.Fatalf("ahhh! %v", err)
			}
			if len(read) < int(pl.FixedHeader.RemLen)+offset {
				log.Fatalf("ahhh! didn't read enough!")
			}
			log.Printf("packet header: %#v\n", pl.FixedHeader)

			switch pl.FixedHeader.Pt {
			case packets.PUBLISH:
				pl.Properties.Zero()
				packets.DecodePublish(
					&pl.FixedHeader,
					&pl.Publish,
					&pl.Properties,
					read[offset:],
				)
				pubChan <- PubMsg{
					Topic: pl.Publish.Topic,
					Msg:   read,
				}
			case packets.PINGREQ:
				log.Printf("ping\n")
				clear(read)
				read[0] = 0b11010000
				read[1] = 0
				_, err = conn.Write(read[:2])
				if err != nil {
					log.Fatalf("ahhh! %v\n", err)
				}
			case packets.SUBSCRIBE:
				log.Printf("got a subscribe\n")
				pl.Properties.Zero()
				pl.Subscribe.Zero()
				packets.DecodeSubscribe(
					&pl.Subscribe,
					&pl.Properties,
					read[offset:],
				)
				log.Printf("subscribe packet id: %d\n", pl.Subscribe.PackedId)

				pl.Suback.Zero()
				tfs := [][]string{}
				for _, filter := range pl.Subscribe.TopicFilters {
					// TODO: validate filter in here
					log.Printf("topic filter: %s\n", filter.Filter)
					pl.Suback.ReasonCodes = append(pl.Suback.ReasonCodes, 0)
					tfs = append(tfs, strings.Split(filter.Filter, "/"))
				}
				subChan <- SubMsg{
					ClientId:     connect.Id,
					TopicFilters: tfs,
				}

				pl.Suback.PacketId = pl.Subscribe.PackedId
				pl.Properties.Zero()
				clear(read)
				scratch := bp.GetBuf()
				i := packets.EncodeSuback(&pl.Suback, &pl.Properties, read, scratch)
				_, err := conn.Write(read[:i])
				if err != nil {
					log.Fatalf("error writing suback: %v\n", err)
				}
				bp.ReturnBuf(scratch)
			default:
				log.Printf("bad packet\n")
			}
			bp.ReturnBuf(read)
		}
	}
}

func setupConnection(
	conn net.Conn,
	fh *packets.FixedHeader,
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

	offset, err := packets.DecodeFixedHeader(fh, buf)
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

func readPackets(conn net.Conn, bp *BufPool, sender chan<- []byte) {
	for {
		buf := bp.GetBuf()
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("error reading from conn! %v\n", err)
		}
		log.Printf("packet bytes in reader: %v\n", buf[:10])
		sender <- buf[:n]
	}
}
