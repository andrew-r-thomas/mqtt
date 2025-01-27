package packets

import "encoding/binary"

type Suback struct {
	PacketId    uint16
	ReasonCodes []byte
}

func EncodeSuback(
	p *Suback,
	props *Properties,
	buf []byte,
	scratch []byte,
) int {
	buf[0] = 9 << 4

	binary.BigEndian.PutUint16(scratch[0:2], p.PacketId)
	sl := 2
	sl += EncodeProps(props, scratch[sl:], buf[1:])
	copy(scratch[sl:], p.ReasonCodes)
	sl += len(p.ReasonCodes)

	bl := encodeVarByteInt(buf[1:], sl)
	copy(buf[bl+1:], scratch[:sl])

	return bl + sl + 1
}

func (p *Suback) Zero() {
	p.PacketId = 0
	clear(p.ReasonCodes)
	p.ReasonCodes = p.ReasonCodes[:0]
}
