package packets

import (
	"encoding/binary"
)

type Unsubscribe struct {
	TopciFilters []string
	PacketId     uint16
}

func DecodeUnsubscribe(u *Unsubscribe, props *Properties, data []byte) error {
	u.PacketId = binary.BigEndian.Uint16(data[0:2])
	rest := data[2:]

	offset, err := DecodeProps(props, rest)
	if err != nil {
		return err
	}
	rest = rest[offset:]

	offset = 0
	for offset < len(rest) {
		tf, off, err := decodeUtf8(rest[offset:])
		if err != nil {
			return err
		}

		u.TopciFilters = append(
			u.TopciFilters,
			tf,
		)
		offset += off
	}

	return nil
}

func (u *Unsubscribe) Zero() {
	u.PacketId = 0
	clear(u.TopciFilters)
	u.TopciFilters = u.TopciFilters[:0]
}
