package packets

import "encoding/binary"

type Connack struct {
	sessionPresent bool
	reasonCode     ReasonCode
	props          ConnackProps
}

type ConnackProps struct {
	// don't send on zero value
	rm  uint16       // receive maximum
	mps uint32       // maximum packet size
	aci string       // assigned client identifier
	tam uint16       // topic alias maximum
	rs  string       // reason string
	up  []StringPair // user property
	ri  string       // response information
	sr  string       // server reference
	am  string       // authentication method
	ad  []byte       // authentication data

	// special rules
	mq byte // maximum QoS (if 2, don't send)

	// if false, send 0 byte, otherwise don't send
	ra  bool // retain available
	wsa bool // wildcard subscription supported
	sia bool // subscription identifiers supported
	ssa bool // shared subscriptions available

	// if -1, don't send
	ska int32 // server keep alive (uint16)
	sei int64 // session expiry interval (uint32)
}

// these buffers are assumed to be empty (zero len, some capacity)
func EncodeConnack(connack *Connack, buf []byte, scratch []byte) []byte {
	// encode the packet type, no flags for this one
	buf = append(buf, byte(2<<4))

	remLen := 2
	// encode variable header into scratch
	if connack.sessionPresent {
		scratch = append(scratch, 1)
	} else {
		scratch = append(scratch, 0)
	}
	scratch = append(scratch, byte(connack.reasonCode))
	scratch = encodeConnackProps(&connack.props, scratch, buf[1:])
	remLen += len(scratch)
	buf = encodeVarByteInt(buf, remLen)
	buf = append(buf, scratch...)

	return buf
}

func encodeConnackProps(props *ConnackProps, buf []byte, scratch []byte) []byte {
	if props.rm != 0 {
		scratch = append(scratch, 33)
		scratch = binary.BigEndian.AppendUint16(scratch, props.rm)
	}
	if props.mps != 0 {
		scratch = append(scratch, 39)
		scratch = binary.BigEndian.AppendUint32(scratch, props.mps)
	}
	if props.aci != "" {
		scratch = append(scratch, 18)
		scratch = encodeUtf8(scratch, props.aci)
	}
	if props.tam != 0 {
		scratch = append(scratch, 34)
		scratch = binary.BigEndian.AppendUint16(scratch, props.tam)
	}
	if props.rs != "" {
		scratch = append(scratch, 31)
		scratch = encodeUtf8(scratch, props.rs)
	}
	for _, up := range props.up {
		scratch = append(scratch, 38)
		scratch = encodeUtf8(scratch, up.name)
		scratch = encodeUtf8(scratch, up.val)
	}
	if props.ri != "" {
		scratch = append(scratch, 26)
		scratch = encodeUtf8(scratch, props.ri)
	}
	if props.sr != "" {
		scratch = append(scratch, 28)
		scratch = encodeUtf8(scratch, props.sr)
	}
	if props.am != "" {
		scratch = append(scratch, 21)
		scratch = encodeUtf8(scratch, props.am)
	}
	if len(props.ad) != 0 {
		scratch = append(scratch, 22)
		scratch = binary.BigEndian.AppendUint16(scratch, uint16(len(props.ad)))
		scratch = append(scratch, props.ad...)
	}
	if props.mq != 2 {
		scratch = append(scratch, 36)
		scratch = append(scratch, props.mq)
	}
	if !props.ra {
		scratch = append(scratch, 37)
		scratch = append(scratch, 0)
	}
	if !props.wsa {
		scratch = append(scratch, 40)
		scratch = append(scratch, 0)
	}
	if !props.sia {
		scratch = append(scratch, 41)
		scratch = append(scratch, 0)
	}
	if !props.ssa {
		scratch = append(scratch, 42)
		scratch = append(scratch, 0)
	}
	if props.ska != -1 {
		scratch = append(scratch, 19)
		scratch = binary.BigEndian.AppendUint16(scratch, uint16(props.ska))
	}
	if props.sei != -1 {
		scratch = append(scratch, 17)
		scratch = binary.BigEndian.AppendUint32(scratch, uint32(props.sei))
	}

	buf = encodeVarByteInt(buf, len(scratch))
	buf = append(buf, scratch...)
	clear(scratch)

	return buf
}

func (c *Connack) Zero() {
	c.sessionPresent = false
	c.reasonCode = 0
	c.props.zero()
}

func (p *ConnackProps) zero() {
	p.rm = 0
	p.mps = 0
	p.aci = ""
	p.tam = 0
	p.rs = ""
	clear(p.up)
	p.up = p.up[:0]
	p.ri = ""
	p.sr = ""
	p.am = ""
	clear(p.ad)
	p.ad = p.ad[:0]
	p.mq = 0
	p.ra = true
	p.wsa = true
	p.sia = true
	p.ssa = true
	p.ska = -1
	p.sei = -1
}
