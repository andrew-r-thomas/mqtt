/*

 TODO:
 - try to implement things with copy wherever possible
 - figure out what we want to do about reseting strings

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

type StringPair struct {
	name string
	val  string
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

	remLen, offest, err := decodeVarByteInt(data[1:])
	fh.RemLen = remLen

	if err != nil {
		return offest, fmt.Errorf("Malformed fixed header: %v", err)
	}

	return offest, nil
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
			return offset, fmt.Errorf("%v: %v", MalProps, InvalidPropId)
		}
	}

	return offset, nil
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
func encodeVarByteInt(buf []byte, num uint32) {
	// TODO:
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
