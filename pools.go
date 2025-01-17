package mqtt

import (
	"errors"
)

// PERF: consider just putting things in the channel, with no slice buffer
type PacketPool struct {
	freeList chan int
	packets  []Packet
}
type PoolPacket struct {
	id     int
	Packet *Packet
}

func NewPacketPool(capacity int) PacketPool {
	freeList := make(chan int, capacity)
	packets := make([]Packet, capacity)

	for i := range capacity {
		freeList <- i
	}

	return PacketPool{freeList: freeList, packets: packets}
}

var PacketPoolEmpty = errors.New("packet pool does not have any available packets")

func (pp *PacketPool) GetPacket() (PoolPacket, error) {
	select {
	case i := <-pp.freeList:
		return PoolPacket{id: i, Packet: &pp.packets[i]}, nil
	default:
		return PoolPacket{}, PacketPoolEmpty
	}
}

func (pp *PacketPool) ReturnPacket(poolPacket PoolPacket) {
	poolPacket.Packet.Zero()
	pp.freeList <- poolPacket.id
}

type BufPool struct {
	freeList chan int
	bufs     [][]byte
}
type PoolBuf struct {
	id  int
	Buf []byte
}

func NewBufPool(capacity int, bufCap int) BufPool {
	freeList := make(chan int, capacity)
	bufs := make([][]byte, capacity)

	for i := range capacity {
		freeList <- i
		bufs[i] = make([]byte, bufCap)
	}

	return BufPool{freeList: freeList, bufs: bufs}
}

var BufPoolEmpty = errors.New("buffer pool does not have any available buffers")

func (bp *BufPool) GetBuf() (PoolBuf, error) {
	select {
	case i := <-bp.freeList:
		return PoolBuf{id: i, Buf: bp.bufs[i]}, nil
	default:
		return PoolBuf{}, BufPoolEmpty
	}
}

func (bp *BufPool) ReturnBuf(poolBuf PoolBuf) {
	for i := range poolBuf.Buf {
		poolBuf.Buf[i] = 0
	}
	bp.freeList <- poolBuf.id
}
