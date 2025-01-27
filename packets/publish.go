package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Publish struct {
	Topic    string
	PacketId uint16
	Payload  []byte
}

var (
	MalPubPacket = errors.New("Malformed publish packet")
)

func DecodePublish(fh *FixedHeader, publish *Publish, props *Properties, data []byte) error {
	topic, l, err := decodeUtf8(data)
	if err != nil {
		return fmt.Errorf("%v: %v", MalPubPacket, err)
	}
	publish.Topic = topic
	rest := data[l:]

	if fh.Flags&0b00000010 > 0 || fh.Flags&0b00000100 > 0 {
		// extract packet id
		publish.PacketId = binary.BigEndian.Uint16(rest[:2])
		rest = rest[2:]
		l += 2
	}

	off, err := DecodeProps(props, rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalPubPacket, err)
	}

	rest = rest[off:]
	l += off
	// TODO: would like copying here
	publish.Payload = append(publish.Payload, rest...)

	return nil
}

func (p *Publish) Zero() {
	p.Topic = ""
	p.PacketId = 0
	clear(p.Payload)
	p.Payload = p.Payload[:0]
}
