package mqtt

import (
	"database/sql"
	"log"
	"net"
	"strings"

	"github.com/andrew-r-thomas/mqtt/packets"
	"github.com/go-webauthn/webauthn/webauthn"

	_ "github.com/mattn/go-sqlite3"
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
	fp FHPool

	auth *webauthn.WebAuthn
	db   *sql.DB
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
	fp := NewFHPool(1000)

	db, err := sql.Open("sqlite3", "./db.db")
	// TODO: handle creating tables and such
	if err != nil {
		log.Fatalf("error opening db: %v\n", err)
	}

	return Server{
		addr: addr,

		tt: tt,

		pubChan:    pubChan,
		subChan:    subChan,
		unSubChan:  unSubChan,
		addCliChan: addCliChan,
		remCliChan: remCliChan,

		bp: bp,
		fp: fp,

		db: db,
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

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	s.setupClient(conn)
}

func (s *Server) setupClient(conn net.Conn) {
	// timeout and wait for CONNECT packet

	// ... probably some more stuff

	// authenticate
	// if login (need to get this from connect packet or auth packet)
	// get the user from data store
	// TODO: const strings for the sql
	stmt, err := s.db.Prepare("select * from users where id = ?")
	if err != nil {
		log.Fatalf("error preparing statement: %v\n", err)
	}
	var user User
	// TODO: get id from connect or something
	err = stmt.QueryRow("").Scan(&user)
	if err != nil {
		log.Fatalf("error querying row: %v\n", err)
	}
	stmt.Close()

	cred, session, err := s.auth.BeginLogin(&user)
}

type User struct{}

func (u *User) WebAuthnID() []byte {
	// TODO:
	return nil
}
func (u *User) WebAuthnName() string {
	// TODO:
	return ""
}
func (u *User) WebAuthnDisplayName() string {
	// TODO:
	return ""
}
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	// TODO:
	return nil
}

func handleClient(
	conn net.Conn,

	pubChan chan<- PubMsg,
	subChan chan<- SubMsg,
	unSubChan chan<- UnSubMsg,
	addCliChan chan<- AddCliMsg,
	remCliChan chan<- RemCliMsg,

	bp *BufPool,
	fp *FHPool,
	auth *webauthn.WebAuthn,
) {
	connect, _ := setupConnection(
		conn,
		&pl.Properties,
		bp,
		fp,
	)

	readChan := make(chan Packet, 10)
	writeChan := make(chan []byte, 10)
	addCliChan <- AddCliMsg{
		ClientId: connect.Id.String(),
		Sender:   writeChan,
	}
	go readPump(conn, bp, fp, readChan)
	go writePump(conn, bp, writeChan)

	for packet := range readChan {
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
			writeChan <- packet.buf[:2]
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
			bp.ReturnBuf(scratch)
			writeChan <- packet.buf[:i]
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
				pl.Unsuback.ReasonCodes = append(
					pl.Unsuback.ReasonCodes,
					0,
				)
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
			i := packets.EncodeUnsuback(
				&pl.Unsuback,
				&pl.Properties,
				packet.buf,
				scratch,
			)
			bp.ReturnBuf(scratch)
			writeChan <- packet.buf[:i]
		case packets.DISCONNECT:
			remCliChan <- RemCliMsg{
				ClientId: connect.Id.String(),
			}
			close(writeChan)
			return
		default:
			log.Printf("bad packet\n")
			bp.ReturnBuf(packet.buf)
		}
	}
}

func setupConnection(
	conn net.Conn,
	props *packets.Properties,
	bp *BufPool,
	fp *FHPool,
) (packets.Connect, packets.Properties) {
	// wait for connect packet
	buf := bp.GetBuf()
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("ahhh! %v", err)
	}

	// decode fixed header
	fh := packets.FixedHeader{}
	fh.Zero()
	offset := packets.DecodeFixedHeader(&fh, buf)
	if offset == -1 {
		log.Fatalf("ahhh! %v", err)
	}
	if n < int(fh.RemLen)+offset {
		log.Fatalf("ahhh! didn't read enough!")
	}
	if fh.Pt != packets.CONNECT {
		log.Fatalf(
			"sent packet other than connect first: %s\n",
			fh.Pt.String(),
		)
	}

	// decode connect packet
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
	scratch := bp.GetBuf()

	// encode connack packet and write to connection
	l := packets.EncodeConnack(&connack, props, buf, scratch)
	n, err = conn.Write(buf[:l])
	if err != nil {
		log.Fatalf("ahh error writing to conn: %v\n", err)
	}

	// cleanup
	bp.ReturnBuf(scratch)
	bp.ReturnBuf(buf)

	// TODO: we want to authenticate on connect, will need a separate ep for admin stuff

	return connect, willProps
}

type Packet struct {
	fh  packets.FixedHeader
	buf []byte // this is the whole buffer, including the fixed header
}

func readPump(
	conn net.Conn,
	bp *BufPool,
	fp *FHPool,
	sender chan<- Packet,
) {
	for {
		fh := fp.GetFH()
		buf := bp.GetBuf()

		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("error reading from conn! %v\n", err)
		}

		offset, err := packets.DecodeFixedHeader(&fh, buf)

		for n < int(fh.RemLen)+offset {
			// didn't read enough
			b := bp.GetBuf()

			nn, err := conn.Read(b)
			if err != nil {
				log.Fatalf(
					"error reading from conn: %v\n",
					err,
				)
			}

			n += nn
			buf = append(buf, b...)

			bp.ReturnBuf(b)
		}

		sender <- Packet{fh: fh, buf: buf[:n]}

		if fh.Pt == packets.DISCONNECT {
			return
		}
	}
}

func writePump(conn net.Conn, bp *BufPool, c <-chan []byte) {
	for buf := range c {
		n, err := conn.Write(buf)
		if err != nil {
			log.Fatalf("error writting to conn: %v\n", err)
		}
		if n != len(buf) {
			log.Fatalf("did not write full buf\n")
		}
		bp.ReturnBuf(buf)
	}
}
