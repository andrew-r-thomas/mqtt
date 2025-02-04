package mqtt

import (
	"sync"

	"github.com/andrew-r-thomas/mqtt/packets"
)

type BufPool struct {
	pool sync.Pool
}

func NewBufPool(bufCap int) *BufPool {
	return &BufPool{
		pool: sync.Pool{
			New: func() any {
				buf := make([]byte, bufCap)
				return &buf
			},
		},
	}
}
func (bp *BufPool) GetBuf() []byte {
	buf := bp.pool.Get().(*[]byte)
	return *buf
}
func (bp *BufPool) ReturnBuf(buf []byte) {
	clear(buf)
	bp.pool.Put(&buf)
}

type FHPool struct {
	pool sync.Pool
}

func NewFHPool() *FHPool {
	return &FHPool{
		pool: sync.Pool{
			New: func() any {
				fh := new(packets.FixedHeader)
				fh.Zero()
				return fh
			},
		},
	}
}

func (fp *FHPool) GetFH() *packets.FixedHeader {
	return fp.pool.Get().(*packets.FixedHeader)
}

func (fp *FHPool) ReturnFH(fh *packets.FixedHeader) {
	fh.Zero()
	fp.pool.Put(fh)
}
