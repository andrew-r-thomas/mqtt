package mqtt

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
