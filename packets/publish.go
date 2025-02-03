package packets

import (
	"encoding/binary"
	"errors"
	"strings"
)

type Publish struct {
	Topic    strings.Builder
	PacketId uint16
	Payload  []byte
}

var (
	MalPubPacket = errors.New("Malformed publish packet")
)

func DecodePublish(
	fh *FixedHeader,
	publish *Publish,
	props *Properties,
	data []byte,
) error {
	l := decodeUtf8(data, publish.Topic)
	if l == -1 {
		return MalPubPacket
	}
	rest := data[l:]

	if fh.Flags&0b00000010 > 0 || fh.Flags&0b00000100 > 0 {
		// extract packet id
		publish.PacketId = binary.BigEndian.Uint16(rest[:2])
		rest = rest[2:]
		l += 2
	}

	off := DecodeProps(props, rest)
	if off == -1 {
		return MalPubPacket
	}

	rest = rest[off:]
	l += off
	// TODO: would like copying here
	publish.Payload = append(publish.Payload, rest...)

	return nil
}

func (p *Publish) Zero() {
	p.Topic.Reset()
	p.PacketId = 0
	clear(p.Payload)
	p.Payload = p.Payload[:0]
}
