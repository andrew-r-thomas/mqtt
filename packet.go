package mqtt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

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

var MalVarByteInt = errors.New("Malformed variable byte integer")

func decodeVarByteInt(data []byte) (uint32, int, error) {
	var mult uint32 = 0
	var val uint32 = 0

	i := 0
	for {
		if mult > multMax {
			return 0, 0, MalVarByteInt
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

func decodeProps(props *Properties, data []byte) (int, error) {
	// TODO:
	l, offset, err := decodeVarByteInt(data)
	if err != nil {

	}
	return 0, nil
}

var UnsupProtoc = errors.New("Unsupported protocol")
var UnsupProtocV = errors.New("Unsupported protocol version")
var MalConnPacket = errors.New("Malformed connect packet")

// expects data to start at variable header
func decodeConnect(p *Packet, data []byte) error {
	// TODO:
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

	keepAlive := binary.BigEndian.Uint16(rest[2:])

	err = decodeProps(&p.props, rest[4:])

	return nil
}
