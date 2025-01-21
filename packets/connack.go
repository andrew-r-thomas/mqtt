package packets

type Connack struct {
	sessionPresent bool
	reasonCode     ReasonCode
}

// these buffers are assumed to be empty (zero len, some capacity)
func EncodeConnack(connack *Connack, props *Properties, buf []byte, scratch []byte) int {
	// encode the packet type, no flags for this one
	buf[0] = 2 << 4

	// encode variable header into scratch
	if connack.sessionPresent {
		scratch[0] = 1
	} else {
		scratch[0] = 0
	}
	scratch[1] = byte(connack.reasonCode)
	sl := 2
	sl += EncodeProps(props, scratch[sl:], buf[1:])
	bl := encodeVarByteInt(buf[1:], sl)
	copy(buf[bl+1:], scratch[:sl])

	return bl + sl + 1
}

func (c *Connack) Zero() {
	c.sessionPresent = false
	c.reasonCode = 0
}
