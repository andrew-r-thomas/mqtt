package packets

import (
	"encoding/binary"
	"errors"
	"strings"
)

type Subscribe struct {
	TopicFilters []TopicFilter
	PackedId     uint16
}

type TopicFilter struct {
	Filter strings.Builder
	Opts   byte
}

var MalSubPacket = errors.New("Malformed subscribe packet")

func DecodeSubscribe(s *Subscribe, props *Properties, data []byte) error {
	s.PackedId = binary.BigEndian.Uint16(data[0:2])
	rest := data[2:]

	offset := DecodeProps(props, rest)
	if offset == -1 {
		return MalSubPacket
	}
	rest = rest[offset:]

	offset = 0
	for offset < len(rest) {
		var tf TopicFilter
		// decode topic filters
		off := decodeUtf8(rest[offset:], tf.Filter)
		if off == -1 {
			return MalSubPacket
		}
		tf.Opts = rest[offset+off]
		s.TopicFilters = append(
			s.TopicFilters,
			tf,
		)
		offset += off + 1
	}

	return nil
}

func (s *Subscribe) Zero() {
	s.PackedId = 0
	clear(s.TopicFilters)
	s.TopicFilters = s.TopicFilters[:0]
}
