package mqtt

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// PERF: consider having the packet own a buf, and writing pointers into it
// for the various types for easy access, or functions to decode things on the fly
// this one is non obvious since a lot of the data has headers and stuff for indicating
// length and whatnot (and im not even sure if you can do it), so either way we will want
// to do some extensive profiling on both options
type Packet struct {
	fh    fixedHeader
	props Properties
}

type fixedHeader struct {
	pt     PacketType
	flags  byte
	remLen uint32
}

// PERF: consider implications of making function to decode this from byte,
// or encoding it directly in the Packet fixedHeader struct
type PacketType byte

const (
	Reserved PacketType = iota
	CONNECT
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
	AUTH
)

type Properties struct {
	pfi byte       // payload format indicator
	mei uint32     // message expiry interval
	ct  string     // content type
	rt  string     // response topic
	cd  []byte     // correlation data
	si  uint32     // subscription identifier
	sei uint32     // session expiry interval
	aci string     // assigned client identifier
	ska uint16     // server keep alive
	am  string     // authentication method
	ad  []byte     // authentication data
	rpi byte       // request problem information
	wdi uint32     // will delay interval
	rri byte       // request response information
	ri  string     // response information
	sr  string     // server reference
	rs  string     // reason string
	rm  uint16     // receive maximum
	tam uint16     // topic alias maximum
	ta  uint16     // topic alias
	mq  byte       // maximum QoS
	ra  byte       // retain available
	up  stringPair // user property
	mps uint32     // maximum packet size
	wsa byte       // wildcard subscription available
	sia byte       // subscription identifier available
	ssa byte       // shared subscription available
}

type stringPair struct {
	name string
	val  string
}

// this function assumes you have decoded the fixed header already
// this is because we want to be able to determine how many bytes to read
// from the connection for a packet by the fixed header
func DecodePacket(p *Packet, data []byte) error {
	// TODO: add checking for basic length requirements for different packets (using p.fh.remLen and p.fh.pt)
	// this is so that we can do unchecked indexing into the data without worrying
	switch p.fh.pt {
	case CONNECT:
		return decodeConnect(p, data)
	case CONNACK:
	case PUBLISH:
	case PUBACK:
	case PUBREC:
	case PUBREL:
	case PUBCOMP:
	case SUBSCRIBE:
	case SUBACK:
	case UNSUBSCRIBE:
	case UNSUBACK:
	case PINGREQ:
	case PINGRESP:
	case DISCONNECT:
	case AUTH:
	default:
		return fmt.Errorf("Malformed packet, packet type is %v", p.fh.pt)
	}

	return nil
}

func DecodeFixedHeader(fh *fixedHeader, data []byte) (int, error) {
	fh.pt = PacketType(data[0] >> 4)
	fh.flags = data[0] & 0b00001111

	remLen, offest, err := decodeVarByteInt(data[1:])
	if err != nil {
		return offest, fmt.Errorf("Malformed fixed header: %v", err)
	}
	fh.remLen = remLen

	return offest, nil
}

// TODO: pick reasonable defaults for sizes for arrays and stuff
var ZeroPacket = Packet{}

func (p *Packet) Zero() {
	*p = ZeroPacket
}

const multMax uint32 = 128 * 128 * 128

var InvalidVarByteInt = errors.New("Invalid variable byte integer")

func decodeVarByteInt(data []byte) (uint32, int, error) {
	var mult uint32 = 0
	var val uint32 = 0

	i := 0
	for {
		if mult > multMax {
			return 0, 0, InvalidVarByteInt
		}

		b := data[i]
		// PERF: making a u32 every time, maybe can keep things at byte level
		val += (uint32(b) & 127) * mult
		mult *= 128
		i += 1

		if (b & 128) == 0 {
			return val, i, nil
		}
	}

}

var InvalidUtf8 = errors.New("Invalid utf8 string")

func decodeUtf8(data []byte) (string, int, error) {
	var str strings.Builder
	l := int(binary.BigEndian.Uint16(data[0:2]))

	off := 0
	for {
		r, size := utf8.DecodeRune(data[off:])
		if r == utf8.RuneError {
			return str.String(), l, InvalidUtf8
		}
		// TODO: more checking for mqtt specifics around utf8
		str.WriteRune(r)
		off += size
		if off >= l {
			break
		}
	}

	return str.String(), l, nil
}

func decodeBinary(data []byte, buf []byte) int {
	l := binary.BigEndian.Uint16(data[0:2])
	buf = append(buf, data[2:l+2]...)
	return int(l) + 2
}

var MalProps = errors.New("Malformed properties")
var InvalidPropId = errors.New("Invalid property identifier")

func decodeProps(props *Properties, data []byte) (int, error) {
	// TODO:
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
		case 11: // subscription identifier
			si, off, err := decodeVarByteInt(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			offset += off + 1
			props.si = si
		case 17: // session expiry interval
			props.sei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 18: // assigned client identifier
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.aci = str
			offset += off + 1
		case 19: // server keep alive
			props.ska = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
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
		case 24: // will delay interval
			props.wdi = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 25: // request response information
			props.rri = data[offset+1]
			offset += 2
		case 26: // response information
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.ri = str
			offset += off + 1
		case 28: // server reference
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.sr = str
			offset += off + 1
		case 31: // reason string
			str, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.rs = str
			offset += off + 1
		case 33: // receive maximum
			props.rm = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 34: // topic alias maximum
			props.tam = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 35: // topic alias
			props.ta = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 36: // maximum QoS
			props.mq = data[offset+1]
			offset += 2
		case 37: // retain available
			props.ra = data[offset+1]
			offset += 2
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
		case 40: // wildcard subscription available
			props.wsa = data[offset+1]
			offset += 2
		case 41: // subscription identifier available
			props.sia = data[offset+1]
			offset += 2
		case 42: // shared subscription available
			props.ssa = data[offset+1]
			offset += 2
		default:
			return offset, fmt.Errorf("%v: %v", MalProps)
		}
	}

	return offset, nil
}

var UnsupProtoc = errors.New("Unsupported protocol")
var UnsupProtocV = errors.New("Unsupported protocol version")
var MalConnPacket = errors.New("Malformed connect packet")

// expects data to start at variable header
func decodeConnect(p *Packet, data []byte) error {
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

	connFlags := rest[1]
	if connFlags&1 == 1 {
		return fmt.Errorf("%v: reserved flag is 1")
	}

	keepAlive := binary.BigEndian.Uint16(rest[2:4])

	offset, err = decodeProps(&p.props, rest[4:])
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}

	rest = rest[offset:]
	// TODO: decode payload

	return nil
}
