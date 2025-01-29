package packets

import "fmt"

type FixedHeader struct {
	Pt     PacketType
	Flags  byte
	RemLen uint32
}

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

func (fh *FixedHeader) Zero() {
	fh.Pt = 0
	fh.Flags = 0
	fh.RemLen = 0
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
