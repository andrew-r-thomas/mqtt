package mqtt

import "github.com/andrew-r-thomas/mqtt/packets"

type BufPool struct {
	bufCap int
	pool   chan []byte
}

func NewBufPool(capacity int, bufCap int) BufPool {
	pool := make(chan []byte, capacity)
	for range capacity {
		pool <- make([]byte, bufCap)
	}
	return BufPool{pool: pool, bufCap: bufCap}
}

func (bp *BufPool) GetBuf() []byte {
	select {
	case buf := <-bp.pool:
		return buf
	default:
		return make([]byte, bp.bufCap)
	}
}

func (bp *BufPool) ReturnBuf(buf []byte) {
	clear(buf)
	select {
	case bp.pool <- buf[:min(bp.bufCap, len(buf))]:
	default:
	}
}

type FHPool struct {
	pool chan packets.FixedHeader
}

func NewFHPool(capacity int) FHPool {
	pool := make(chan packets.FixedHeader, capacity)
	for range capacity {
		fh := packets.FixedHeader{}
		fh.Zero()
		pool <- fh
	}
	return FHPool{pool: pool}
}

func (fp *FHPool) GetFH() packets.FixedHeader {
	select {
	case fh := <-fp.pool:
		return fh
	default:
		fh := packets.FixedHeader{}
		fh.Zero()
		return fh
	}
}

func (fp *FHPool) ReturnFH(fh packets.FixedHeader) {
	fh.Zero()
	select {
	case fp.pool <- fh:
	default:
	}
}
