package packets

import (
	"encoding/binary"
)

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
func EncodeConnack(connack *Connack, buf []byte, scratch []byte) int {
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
	sl += encodeConnackProps(&connack.props, scratch[sl:], buf[1:])
	bl := encodeVarByteInt(buf[1:], sl)
	copy(buf[bl+1:], scratch[:sl])

	return bl + sl + 1
}

func encodeConnackProps(props *ConnackProps, buf []byte, scratch []byte) int {
	l := 0
	if props.rm != 0 {
		scratch[l] = 33
		binary.BigEndian.PutUint16(scratch[l+1:l+3], props.rm)
		l += 3
	}
	if props.mps != 0 {
		scratch[l] = 39
		binary.BigEndian.PutUint32(scratch[l+1:l+5], props.mps)
		l += 5
	}
	if props.aci != "" {
		scratch[l] = 18
		ll := encodeUtf8(scratch[l+1:], props.aci)
		l += ll + 1
	}
	if props.tam != 0 {
		scratch[l] = 34
		binary.BigEndian.PutUint16(scratch[l+1:l+3], props.tam)
		l += 3
	}
	if props.rs != "" {
		scratch[l] = 31
		ll := encodeUtf8(scratch[l+1:], props.rs)
		l += ll + 1
	}
	for _, up := range props.up {
		scratch[l] = 38
		lln := encodeUtf8(scratch[l+1:], up.name)
		llv := encodeUtf8(scratch[lln+1:], up.val)
		l += lln + llv + 1
	}
	if props.ri != "" {
		scratch[l] = 26
		ll := encodeUtf8(scratch[l+1:], props.ri)
		l += ll + 1
	}
	if props.sr != "" {
		scratch[l] = 28
		ll := encodeUtf8(scratch[l+1:], props.sr)
		l += ll + 1
	}
	if props.am != "" {
		scratch[l] = 21
		ll := encodeUtf8(scratch[l+1:], props.sr)
		l += ll + 1
	}
	if len(props.ad) != 0 {
		scratch[l] = 22
		binary.BigEndian.AppendUint16(scratch[l+1:l+3], uint16(len(props.ad)))
		copy(scratch[l+3:], props.ad)
		l += len(props.ad) + 3
	}
	if props.mq != 2 {
		scratch[l] = 36
		scratch[l+1] = props.mq
		l += 2
	}
	if !props.ra {
		scratch[l] = 37
		scratch[l+1] = 0
		l += 2
	}
	if !props.wsa {
		scratch[l] = 40
		scratch[l+1] = 0
		l += 2
	}
	if !props.sia {
		scratch[l] = 41
		scratch[l+1] = 0
		l += 2
	}
	if !props.ssa {
		scratch[l] = 42
		scratch[l+1] = 0
		l += 2
	}
	if props.ska != -1 {
		scratch[l] = 19
		binary.BigEndian.PutUint16(scratch[l+1:l+3], uint16(props.ska))
		l += 3
	}
	if props.sei != -1 {
		scratch[l] = 17
		binary.BigEndian.PutUint32(scratch[l+1:l+5], uint32(props.sei))
		l += 5
	}

	bl := encodeVarByteInt(buf, l)
	copy(buf[bl:], scratch[:l])
	clear(scratch)

	return bl + l
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
	p.mq = 2
	p.ra = true
	p.wsa = true
	p.sia = true
	p.ssa = true
	p.ska = -1
	p.sei = -1
}
