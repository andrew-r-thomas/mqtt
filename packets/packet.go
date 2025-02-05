/*

 TODO:
 - try to implement things with copy wherever possible
 - figure out what we want to do about reseting strings
   (kinda thinking we should just use a byte slice
    or maybe there's some helpful stuff in the standard lib)
 - we're gonna do everything with copy, but we need to add bounds checks and stuff

*/

package packets

import (
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf8"
)

type StringPair struct {
	name strings.Builder
	val  strings.Builder
}

func (sp *StringPair) zero() {
	sp.name.Reset()
	sp.val.Reset()
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

const multMax uint32 = 128 * 128 * 128

func decodeVarByteInt(data []byte) (uint32, int) {
	var mult uint32 = 1
	var val uint32 = 0

	i := 0
	for {
		if mult > multMax {
			return val, -1
		}

		b := data[i]
		// PERF: making a u32 every time, maybe can keep things at byte level
		val += uint32(b&127) * mult
		mult *= 128
		i += 1

		if (b & 128) == 0 {
			return val, i
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

// str must be empty when passed to this function
func decodeUtf8(data []byte, str *strings.Builder) int {
	// check for safe slice indexing
	if len(data) <= 2 {
		return -1
	}
	l := int(binary.BigEndian.Uint16(data[0:2]))
	if len(data) < l+2 {
		return -1
	}

	off := 2
	for off < l+2 {
		r, size := utf8.DecodeRune(data[off:])
		if r == utf8.RuneError {
			return -1
		}
		// NOTE: looks like go already throws out U+D800 and U+DFFF,
		// but we should test this
		// also looks like the "zero width no-break space" is cut out by go
		if r == '\u0000' {
			return -1
		}

		str.WriteRune(r)
		off += size
	}

	return off
}

// TODO: think about if we want to take a string builder here
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
