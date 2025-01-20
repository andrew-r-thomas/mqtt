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

var MalProps = errors.New("Malformed properties")
var InvalidPropId = errors.New("Invalid property identifier")

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
