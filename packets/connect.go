package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Connect struct {
	id          string
	username    string
	password    []byte
	props       ConnectProps
	keepalive   uint16
	flags       byte
	willTopic   string
	willPayload []byte
	willProps   WillProps
}

type ConnectProps struct {
	sei uint32 // session expiry interval
	mps uint32 // maximum packet size
	rm  uint16 // receive maximum
	tam uint16 // topic alias maximum
	// TODO: change this to be a slice
	up  StringPair // user property
	am  string     // authentication method
	ad  []byte     // authentication data
	rri byte       // request response information
	rpi byte       // request problem information
}

type WillProps struct {
	pfi byte       // payload format indicator
	mei uint32     // message expiry interval
	ct  string     // content type
	rt  string     // response topic
	cd  []byte     // correlation data
	wdi uint32     // will delay interval
	up  StringPair // user property
}

var (
	UnsupProtoc   = errors.New("Unsupported protocol")
	UnsupProtocV  = errors.New("Unsupported protocol version")
	MalConnPacket = errors.New("Malformed connect packet")
)

func DecodeConnect(connect *Connect, data []byte) error {
	protocolName, offset, err := decodeUtf8(data)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	if protocolName != "MQTT" {
		return fmt.Errorf("%v: %v", MalConnPacket, UnsupProtoc)
	}

	rest := data[offset:]

	if rest[0] != 5 {
		return fmt.Errorf("%v: %v", MalConnPacket, UnsupProtocV)
	}

	connect.flags = rest[1]
	if connect.flags&1 == 1 {
		return fmt.Errorf("%v: reserved flag is 1", MalConnPacket)
	}
	// TODO: more checking on the flags

	connect.keepalive = binary.BigEndian.Uint16(rest[2:4])
	rest = rest[4:]

	offset, err = decodeConnectProps(&connect.props, rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	rest = rest[offset:]

	// client id
	connect.id, offset, err = decodeUtf8(rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	rest = rest[offset:]

	// will props
	if connect.flags&0b00000100 != 0 {
		offset, err = decodeWillProps(&connect.willProps, rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
		}
		rest = rest[offset:]

		connect.willTopic, offset, err = decodeUtf8(rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
		}
		rest = rest[offset:]

		offset = decodeBinary(rest, connect.willPayload)
		rest = rest[offset:]
	}

	// username
	if connect.flags&0b10000000 != 0 {
		connect.username, offset, err = decodeUtf8(rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
		}
		rest = rest[offset:]
	}

	// password
	if connect.flags&0b01000000 != 0 {
		decodeBinary(rest, connect.password)
	}
	return nil
}

func (c *Connect) Zero() {
	c.username = ""
	clear(c.password)
	c.password = c.password[:0]
	c.id = ""
	c.keepalive = 0
	c.flags = 0
	c.willTopic = ""
	clear(c.willPayload)
	c.willPayload = c.willPayload[:0]
	c.props.zero()
	c.willProps.zero()
}

func (p *ConnectProps) zero() {
	p.sei = 0
	p.mps = 0
	p.rm = 0
	p.tam = 0
	p.am = ""
	clear(p.ad)
	p.ad = p.ad[:0]
	p.rri = 0
	p.rpi = 0
	p.up.zero()
}

func (p *WillProps) zero() {
	p.pfi = 0
	p.mei = 0
	p.ct = ""
	p.rt = ""
	clear(p.cd)
	p.cd = p.cd[:0]
	p.wdi = 0
	p.up.zero()
}

func decodeConnectProps(props *ConnectProps, data []byte) (int, error) {
	l, offset, err := decodeVarByteInt(data)
	if err != nil {
		return offset, fmt.Errorf("%v: %v", MalProps, err)
	}
	end := offset + int(l)
	for offset < end {
		switch data[offset] {
		case 17: // session expiry interval
			props.sei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 21: // authentication method
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.am = str
			offset += off + 1
		case 22: // authentication data
			off := decodeBinary(data[offset+1:], props.ad)
			offset += off + 1
		case 23: // request problem information
			props.rpi = data[offset+1]
			offset += 2
		case 25: // request response information
			props.rri = data[offset+1]
			offset += 2
		case 33: // receive maximum
			props.rm = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 34: // topic alias maximum
			props.tam = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 38: // user property
			nameStr, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.up.name = nameStr
			offset += off + 1
			valStr, off, err := decodeUtf8(data[offset:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.up.val = valStr
			offset += off + 1
		case 39: // maximum packet size
			props.mps = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		default:
			return offset, fmt.Errorf("%v: %v", MalProps, InvalidPropId)
		}
	}
	return end, nil
}

func decodeWillProps(props *WillProps, data []byte) (int, error) {
	l, offset, err := decodeVarByteInt(data)
	if err != nil {
		return offset, fmt.Errorf("%v: %v", MalProps, err)
	}
	end := offset + int(l)

	for offset < end {
		switch data[offset] {
		case 1: // payload format indicator
			props.pfi = data[offset+1]
			offset += 2
		case 2: // message expiry interval
			props.mei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 3: // content type
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.ct = str
			offset += off + 1
		case 8: // response topic
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.rt = str
			offset += off + 1
		case 9: // correlation data
			off := decodeBinary(data[offset+1:], props.cd)
			offset += off + 1
		case 24: // will delay interval
			props.wdi = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 38: // user property
			nameStr, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.up.name = nameStr
			offset += off + 1
			valStr, off, err := decodeUtf8(data[offset:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.up.val = valStr
			offset += off + 1
		default:
			return offset, fmt.Errorf("%v: %v", MalProps, InvalidPropId)
		}
	}

	return offset, nil
}
