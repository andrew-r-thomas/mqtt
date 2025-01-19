package packets

type Connack struct {
	flags      byte
	reasonCode ReasonCode
}

// these buffers are assumed to be empty (zero len, some capacity)
func EncodeConnack(connack *Connack, buf []byte, scratch []byte) error {
	// encode the packet type, no flags for this one
	buf = append(buf, byte(2<<4))

	remLen := 2
	// encode variable header into scratch
	scratch = append(scratch, connack.flags)
	scratch = append(scratch, byte(connack.reasonCode))

	return nil
}
