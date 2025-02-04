package packets

import (
	"encoding/binary"
	"errors"
	"strings"
)

type Unsubscribe struct {
	TopicFilters []string
	PacketId     uint16
}

var MalUnsubPacket = errors.New("Malformed unsubscribe packet")

func DecodeUnsubscribe(u *Unsubscribe, props *Properties, data []byte) error {
	u.PacketId = binary.BigEndian.Uint16(data[0:2])
	rest := data[2:]

	offset := DecodeProps(props, rest)
	if offset == -1 {
		return MalUnsubPacket
	}
	rest = rest[offset:]

	offset = 0
	for offset < len(rest) {
		var tf strings.Builder
		off := decodeUtf8(rest[offset:], &tf)
		if off == -1 {
			return MalUnsubPacket
		}

		u.TopicFilters = append(
			u.TopicFilters,
			tf.String(),
		)
		offset += off
	}

	return nil
}

func (u *Unsubscribe) Zero() {
	u.PacketId = 0
	clear(u.TopicFilters)
	u.TopicFilters = u.TopicFilters[:0]
}
