package packets

import "encoding/binary"

type Unsuback struct {
	PacketId    uint16
	ReasonCodes []byte
}

func EncodeUnsuback(
	u *Unsuback,
	props *Properties,
	buf []byte,
	scratch []byte,
) int {
	buf[0] = 11 << 4

	binary.BigEndian.PutUint16(scratch[0:2], u.PacketId)
	sl := 2
	sl += EncodeProps(props, scratch[sl:], buf[1:])
	copy(scratch[sl:], u.ReasonCodes)
	sl += len(u.ReasonCodes)

	bl := encodeVarByteInt(buf[1:], sl)
	copy(buf[bl+1:], scratch[:sl])

	return bl + sl + 1
}

func (u *Unsuback) Zero() {
	u.PacketId = 0
	clear(u.ReasonCodes)
	u.ReasonCodes = u.ReasonCodes[:0]
}
