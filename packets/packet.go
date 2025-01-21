/*

 TODO:
 - try to implement things with copy wherever possible
 - figure out what we want to do about reseting strings
   (kinda thinking we should just use a byte slice
    or maybe there's some helpful stuff in the standard lib)
 - extract out some common functionality for props
 - we're gonna do everything with copy, but we need to add bounds checks and stuff

*/

package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type FixedHeader struct {
	Pt     PacketType
	Flags  byte
	RemLen uint32
}

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

func (pt PacketType) String() string {
	switch pt {
	case Reserved:
		return "Reserved"
	case CONNECT:
		return "Connect"
	case CONNACK:
		return "Connack"
	case PUBLISH:
		return "Publish"
	case PUBACK:
		return "Puback"
	case PUBREL:
		return "Pubrel"
	case PUBCOMP:
		return "Pubcomp"
	case SUBSCRIBE:
		return "Subscribe"
	case SUBACK:
		return "Suback"
	case UNSUBSCRIBE:
		return "Unsubscribe"
	case UNSUBACK:
		return "Unsuback"
	case PINGREQ:
		return "Pingreq"
	case PINGRESP:
		return "Pingresp"
	case DISCONNECT:
		return "Disconnect"
	case AUTH:
		return "Auth"
	default:
		return "Invalid packet type"
	}
}

type StringPair struct {
	name string
	val  string
}

func (sp *StringPair) zero() {
	sp.name = ""
	sp.val = ""
}

type ReasonCode byte

const (
	S     ReasonCode = 0   // Success
	ND    ReasonCode = 0   // Normal Disconnection
	GQ0   ReasonCode = 0   // Granted QoS 0
	GQ1   ReasonCode = 1   // Granted QoS 1
	GQ2   ReasonCode = 2   // Granted QoS 2
	DwWM  ReasonCode = 4   // Disconnect with will message
	NMS   ReasonCode = 16  // No matching subscribers
	NSE   ReasonCode = 17  // No subscription existed
	CA    ReasonCode = 24  // Continue authentication
	RA    ReasonCode = 25  // Re-authenticate
	UE    ReasonCode = 128 // Unspecified error
	MP    ReasonCode = 129 // Malformed packet
	PE    ReasonCode = 130 // Protocol error
	ISE   ReasonCode = 131 // Implementation specific error
	UPV   ReasonCode = 132 // Unsupported protocol version
	CInV  ReasonCode = 133 // Client identifier not valid
	BUNoP ReasonCode = 134 // Bad username or password
	NA    ReasonCode = 135 // Not authorized
	SU    ReasonCode = 136 // Server unavailable
	SB    ReasonCode = 137 // Server busy
	B     ReasonCode = 138 // Banned
	SSD   ReasonCode = 139 // Server shutting down
	BAM   ReasonCode = 140 // Bad authentication method
	KAT   ReasonCode = 141 // Keep alive timeout
	STO   ReasonCode = 142 // Session taken over
	TFI   ReasonCode = 143 // Topic filter invalid
	TNI   ReasonCode = 144 // Topic name invalid
	PIiU  ReasonCode = 145 // Packet identifier in use
	PInF  ReasonCode = 146 // Packet identifier not found
	RME   ReasonCode = 147 // Receive maximum exceeded
	TAI   ReasonCode = 148 // Topic alias invalid
	PTL   ReasonCode = 149 // Packet too large
	MRtH  ReasonCode = 150 // Message rate too high
	QE    ReasonCode = 151 // Quota exceeded
	AA    ReasonCode = 152 // Administrative action
	PFI   ReasonCode = 153 // Payload format invalid
	RNS   ReasonCode = 154 // Retain not supported
	QNS   ReasonCode = 155 // QoS not supported
	UAS   ReasonCode = 156 // Use another server
	SM    ReasonCode = 157 // Server moved
	SSnS  ReasonCode = 158 // Shared subscriptions not supported
	CRE   ReasonCode = 159 // Connection rate exceeded
	MCT   ReasonCode = 160 // Maximum connect time
	SInS  ReasonCode = 161 // Subscription identifiers not supported
	WSnS  ReasonCode = 162 // Wildcard subscriptions not supported
)

func DecodeFixedHeader(fh *FixedHeader, data []byte) (int, error) {
	fh.Pt = PacketType(data[0] >> 4)
	fh.Flags = data[0] & 0b00001111

	remLen, offset, err := decodeVarByteInt(data[1:])
	fh.RemLen = remLen

	if err != nil {
		return offset + 1, fmt.Errorf("Malformed fixed header: %v", err)
	}

	return offset + 1, nil
}

const multMax uint32 = 128 * 128 * 128

var InvalidVarByteInt = errors.New("Invalid variable byte integer")

func decodeVarByteInt(data []byte) (uint32, int, error) {
	var mult uint32 = 1
	var val uint32 = 0

	i := 0
	for {
		if mult > multMax {
			return val, i, InvalidVarByteInt
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
func encodeVarByteInt(buf []byte, num int) int {
	i := 0
	for {
		b := byte(num) % 128
		num = num / 128
		if num > 0 {
			b = b | 128
		}
		buf[i] = b
		i += 1
		if num <= 0 {
			return i
		}
	}
}

var InvalidUtf8 = errors.New("Invalid utf8 string")

func decodeUtf8(data []byte) (string, int, error) {
	var str strings.Builder
	l := int(binary.BigEndian.Uint16(data[0:2]))

	off := 2
	for {
		r, size := utf8.DecodeRune(data[off:])
		if r == utf8.RuneError {
			return str.String(), l, InvalidUtf8
		}
		// TODO: more checking for mqtt specifics around utf8
		str.WriteRune(r)
		off += size
		if off >= l+2 {
			break
		}
	}

	return str.String(), off, nil
}

func encodeUtf8(data []byte, str string) int {
	binary.BigEndian.PutUint16(data[:2], uint16(len(str)))
	copy(data[2:2+len(str)], str)
	return 2 + len(str)
}

func decodeBinary(data []byte, buf []byte) int {
	l := binary.BigEndian.Uint16(data[0:2])
	buf = append(buf, data[2:l+2]...)
	return int(l) + 2
}

type Properties struct {
	// if zero value, don't send
	Mps uint32       // maximum packet size
	Ad  []byte       // authentication data
	Am  string       // authentication method
	Rm  uint16       // receive maximum
	Aci string       // assigned client identifier
	Tam uint16       // topic alias maximum
	Rs  string       // reason string
	Sr  string       // server reference
	Ri  string       // response information
	Up  []StringPair // user property
	Sei uint32       // session expiry interval
	Rri byte         // request response information
	Ska uint16       // server keep alive
	Wdi uint32       // will delay interval
	Pfi byte         // payload format indicator
	Mei uint32       // message expiry interval
	Ct  string       // content type
	Rt  string       // response topic
	Cd  []byte       // correlation data
	Si  uint32       // subscription identifier (var byte int)
	Ta  uint16       // topic alias

	// if 1, don't send
	Wsa byte // wildcard subscription available
	Sia byte // subscription identifier available
	Ssa byte // shared subscription available
	Ra  byte // retain available
	Rpi byte // request problem information

	// if 2, don't send
	Mq byte // maximum qos
}

var MalProps = errors.New("Malformed properties")
var InvalidPropId = errors.New("Invalid property identifier")

func DecodeProps(p *Properties, data []byte) (int, error) {
	l, offset, err := decodeVarByteInt(data)
	if err != nil {
		return offset, fmt.Errorf("%v: %v", MalProps, err)
	}

	end := offset + int(l)
	for offset < end {
		switch data[offset] {
		case 1: // payload format indicator
			p.Pfi = data[offset+1]
			offset += 2
		case 2: // message expiry interval
			p.Mei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 3: // content type
			ct, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Ct = ct
			offset += off + 1
		case 8: // response topic
			rt, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Rt = rt
			offset += off + 1
		case 9: // correlation data
			off := decodeBinary(data[offset+1:], p.Cd)
			offset += off + 1
		case 11: // subscription identifier
			si, off, err := decodeVarByteInt(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Si = si
			offset += off + 1
		case 17: // session expiry interval
			p.Sei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 18: // assigned client identifier
			aci, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Aci = aci
			offset += off + 1
		case 19: // server keep alive
			p.Ska = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 21: // authentication method
			am, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Am = am
			offset += off + 1
		case 22: // authentication data
			off := decodeBinary(data[offset+1:], p.Ad)
			offset += off + 1
		case 23: // request problem information
			p.Rpi = data[offset+1]
			offset += 2
		case 24: // will delay interval
			p.Wdi = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 25: // request response information
			p.Rri = data[offset+1]
			offset += 2
		case 26: // response information
			ri, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Ri = ri
			offset += off + 1
		case 28: // server reference
			sr, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Sr = sr
			offset += off + 1
		case 31: // reason string
			rs, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			p.Rs = rs
			offset += off + 1
		case 33: // receive maximum
			p.Rm = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 34: // topic alias maximum
			p.Tam = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 35: // topic alias
			p.Ta = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 36: // maximum qos
			p.Mq = data[offset+1]
			offset += 2
		case 37: // retain available
			p.Ra = data[offset+1]
			offset += 2
		case 38: // user property
			nameStr, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			offset += off + 1
			valStr, off, err := decodeUtf8(data[offset:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			offset += off + 1
			p.Up = append(p.Up, StringPair{name: nameStr, val: valStr})
		case 39: // maximum packet size
			p.Mps = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 40: // wildcard subscription available
			p.Wsa = data[offset+1]
			offset += 2
		case 41: // subscription identifier available
			p.Sia = data[offset+1]
			offset += 2
		case 42: // shared subscription available
			p.Ssa = data[offset+1]
			offset += 2
		}
	}

	return offset, nil
}

func EncodeProps(p *Properties, buf []byte, scratch []byte) int {
	l := 0
	if p.Mps != 0 {
		scratch[l] = 39
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Mps)
		l += 5
	}
	if len(p.Ad) != 0 {
		scratch[l] = 22
		binary.BigEndian.PutUint16(scratch[l+1:l+3], uint16(len(p.Ad)))
		copy(scratch[l+3:], p.Ad)
		l += len(p.Ad) + 3
	}
	if p.Am != "" {
		scratch[l] = 21
		ll := encodeUtf8(scratch[l+1:], p.Am)
		l += ll + 1
	}
	if p.Rm != 0 {
		scratch[l] = 33
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Rm)
		l += 3
	}
	if p.Aci != "" {
		scratch[l] = 18
		ll := encodeUtf8(scratch[l+1:], p.Aci)
		l += ll + 1
	}
	if p.Tam != 0 {
		scratch[l] = 34
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Tam)
		l += 3
	}
	if p.Rs != "" {
		scratch[l] = 31
		ll := encodeUtf8(scratch[l+1:], p.Rs)
		l += ll + 1
	}
	if p.Sr != "" {
		scratch[l] = 28
		ll := encodeUtf8(scratch[l+1:], p.Sr)
		l += ll + 1
	}
	if p.Ri != "" {
		scratch[l] = 26
		ll := encodeUtf8(scratch[l+1:], p.Ri)
		l += ll + 1
	}
	for _, up := range p.Up {
		scratch[l] = 38
		lln := encodeUtf8(scratch[l+1:], up.name)
		llv := encodeUtf8(scratch[lln+1:], up.val)
		l += lln + llv + 1
	}
	if p.Sei != 0 {
		scratch[l] = 17
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Sei)
		l += 5
	}
	if p.Rri != 0 {
		scratch[l] = 25
		scratch[l+1] = p.Rri
		l += 2
	}
	if p.Ska != 0 {
		scratch[l] = 19
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Ska)
		l += 3
	}
	if p.Wdi != 0 {
		scratch[l] = 24
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Wdi)
		l += 5
	}
	if p.Pfi != 0 {
		scratch[l] = 1
		scratch[l+1] = p.Pfi
		l += 2
	}
	if p.Mei != 0 {
		scratch[l] = 2
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Mei)
		l += 5
	}
	if p.Ct != "" {
		scratch[l] = 3
		ll := encodeUtf8(scratch[l+1:], p.Ct)
		l += ll + 1
	}
	if p.Rt != "" {
		scratch[l] = 8
		ll := encodeUtf8(scratch[l+1:], p.Rt)
		l += ll + 1
	}
	if len(p.Cd) != 0 {
		scratch[l] = 9
		copy(scratch[l+1:], p.Cd)
		l += len(p.Cd) + 1
	}
	if p.Si != 0 {
		scratch[l] = 11
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Si)
		l += 5
	}
	if p.Ta != 0 {
		scratch[l] = 35
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Ta)
		l += 3
	}
	if p.Wsa != 1 {
		scratch[l] = 40
		scratch[l+1] = p.Wsa
		l += 2
	}
	if p.Sia != 1 {
		scratch[l] = 41
		scratch[l+1] = p.Sia
		l += 2
	}
	if p.Ssa != 1 {
		scratch[l] = 42
		scratch[l+1] = p.Ssa
		l += 2
	}
	if p.Ra != 1 {
		scratch[l] = 37
		scratch[l+1] = p.Ra
		l += 2
	}
	if p.Rpi != 1 {
		scratch[l] = 23
		scratch[l+1] = p.Rpi
		l += 2
	}
	if p.Mq != 2 {
		scratch[l] = 36
		scratch[l+1] = p.Mq
		l += 2
	}

	bl := encodeVarByteInt(buf, l)
	copy(buf[bl:], scratch[:l])
	clear(scratch)

	return bl + l
}

func (p *Properties) Zero() {
	p.Mps = 0
	clear(p.Ad)
	p.Ad = p.Ad[:0]
	p.Am = ""
	p.Rm = 0
	p.Aci = ""
	p.Tam = 0
	p.Rs = ""
	p.Sr = ""
	p.Ri = ""
	clear(p.Up)
	p.Up = p.Up[:0]
	p.Sei = 0
	p.Rri = 0
	p.Ska = 0
	p.Wdi = 0
	p.Pfi = 0
	p.Mei = 0
	p.Ct = ""
	p.Rt = ""
	clear(p.Cd)
	p.Cd = p.Cd[:0]
	p.Si = 0
	p.Ta = 0

	p.Wsa = 1
	p.Sia = 1
	p.Ssa = 1
	p.Ra = 1
	p.Rpi = 1

	p.Mq = 2
}
