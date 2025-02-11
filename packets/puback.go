package packets

import "encoding/binary"

type Puback struct {
	reasonCode ReasonCode
	packetId   uint16
}

func EncodePuback(
	p *Puback,
	props *Properties,
	buf []byte,
	scratch []byte,
) int {
	buf[0] = 0b01000000

	binary.BigEndian.PutUint16(scratch, p.packetId)
	scratch[2] = byte(p.reasonCode)
	sl := 3
	sl += EncodeProps(props, scratch[sl:], buf[1:])
	bl := encodeVarByteInt(buf[1:], sl)
	copy(buf[bl+1:], scratch[:sl])

	return bl + sl + 1
}
