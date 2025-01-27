package packets

import "encoding/binary"

type Subscribe struct {
	TopicFilters []TopicFilter
	PackedId     uint16
}

type TopicFilter struct {
	Filter string
	Opts   byte
}

func DecodeSubscribe(s *Subscribe, props *Properties, data []byte) error {
	s.PackedId = binary.BigEndian.Uint16(data[0:2])
	rest := data[2:]

	offset, err := DecodeProps(props, rest)
	if err != nil {
		return err
	}
	rest = rest[offset:]

	offset = 0
	for offset < len(rest) {
		// decode topic filters
		tf, off, err := decodeUtf8(rest[offset:])
		if err != nil {
			return err
		}
		so := rest[offset+off]
		s.TopicFilters = append(
			s.TopicFilters,
			TopicFilter{
				Filter: tf,
				Opts:   so,
			},
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
