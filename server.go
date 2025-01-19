package mqtt

import "net"

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
	// TODO:

	// wait for connect packet

	// wait for other packets, or something from the topic hub
}
