package mqtt

import (
	"log"
	"net"

	"github.com/andrew-r-thomas/mqtt/packets"
)

type Server struct {
	addr string

	// tt TopicTree
	//
	// pubChan    chan<- PubMsg
	// subChan    chan<- SubMsg
	// unSubChan  chan<- UnSubMsg
	// addCliChan chan<- AddCliMsg
	// remCliChan chan<- RemCliMsg

	bp BufPool
}

func NewServer(addr string) Server {
	// pubChan := make(chan PubMsg, 10)
	// subChan := make(chan SubMsg, 10)
	// unSubChan := make(chan UnSubMsg, 10)
	// addCliChan := make(chan AddCliMsg, 10)
	// remCliChan := make(chan RemCliMsg, 10)
	//
	// tt := NewTopicTree(
	// 	pubChan,
	// 	subChan,
	// 	addCliChan,
	// 	unSubChan,
	// 	remCliChan,
	// )

	bp := NewBufPool(1000, 1024)

	return Server{
		addr: addr,

		// tt: tt,
		//
		// pubChan:    pubChan,
		// subChan:    subChan,
		// unSubChan:  unSubChan,
		// addCliChan: addCliChan,
		// remCliChan: remCliChan,

		bp: bp,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	// go s.tt.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go handleClient(
			conn,

			// s.pubChan,
			// s.subChan,
			// s.unSubChan,
			// s.addCliChan,
			// s.remCliChan,

			&s.bp,
		)
	}
}

func handleClient(
	conn net.Conn,

	// pubChan chan<- PubMsg,
	// subChan chan<- SubMsg,
	// unSubChan chan<- UnSubMsg,
	// addCliChan chan<- AddCliMsg,
	// remCliChan chan<- RemCliMsg,

	bp *BufPool,
) {
	fh := packets.FixedHeader{}
	buf := bp.GetBuf()

	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("ahhh! %v", err)
	}

	offset, err := packets.DecodeFixedHeader(&fh, buf)
	if err != nil {
		log.Fatalf("ahhh! %v", err)
	}

	if n < int(fh.RemLen)+offset {
		log.Fatalf("ahhh! didn't read enough!")
	}

	connect := packets.Connect{}

	err = packets.DecodeConnect(&connect, buf[offset:])
	if err != nil {
		log.Fatalf("ahh! error decoding connect:\n%v", err)
	}

	log.Printf("connect packet:\n %#v", connect)
}
