package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strings"
)

type Connect struct {
	Id          strings.Builder
	Username    strings.Builder
	Password    []byte
	Keepalive   uint16
	Flags       byte
	WillTopic   strings.Builder
	WillPayload []byte
}

var (
	UnsupProtoc   = errors.New("Unsupported protocol")
	UnsupProtocV  = errors.New("Unsupported protocol version")
	MalConnPacket = errors.New("Malformed connect packet")
)

func DecodeConnect(
	connect *Connect,
	data []byte,
	props *Properties,
	willProps *Properties,
) error {
	var protocolName strings.Builder
	offset := decodeUtf8(data, &protocolName)
	if offset == -1 {
		return MalConnPacket
	}
	log.Printf("protocol name: %s\n", protocolName.String())
	if protocolName.String() != "MQTT" {
		return fmt.Errorf("%v: %v", MalConnPacket, UnsupProtoc)
	}

	rest := data[offset:]

	if rest[0] != 5 {
		return fmt.Errorf("%v: %v", MalConnPacket, UnsupProtocV)
	}

	connect.Flags = rest[1]
	if connect.Flags&1 == 1 {
		return fmt.Errorf("%v: reserved flag is 1", MalConnPacket)
	}
	// TODO: more checking on the flags

	connect.Keepalive = binary.BigEndian.Uint16(rest[2:4])
	rest = rest[4:]

	offset = DecodeProps(props, rest)
	if offset == -1 {
		return MalConnPacket
	}
	rest = rest[offset:]

	// client id
	offset = decodeUtf8(rest, &connect.Id)
	if offset == -1 {
		return MalConnPacket
	}
	rest = rest[offset:]

	// will props
	if connect.Flags&0b00000100 != 0 {
		offset = DecodeProps(willProps, rest)
		if offset == -1 {
			return MalConnPacket
		}
		rest = rest[offset:]

		offset = decodeUtf8(rest, &connect.WillTopic)
		if offset == -1 {
			return MalConnPacket
		}
		rest = rest[offset:]

		offset = decodeBinary(rest, connect.WillPayload)
		rest = rest[offset:]
	}

	// username
	if connect.Flags&0b10000000 != 0 {
		offset = decodeUtf8(rest, &connect.Username)
		if offset == -1 {
			return MalConnPacket
		}
		rest = rest[offset:]
	}

	// password
	if connect.Flags&0b01000000 != 0 {
		decodeBinary(rest, connect.Password)
	}
	return nil
}

func (c *Connect) Zero() {
	c.Username.Reset()
	clear(c.Password)
	c.Password = c.Password[:0]
	c.Id.Reset()
	c.Keepalive = 0
	c.Flags = 0
	c.WillTopic.Reset()
	clear(c.WillPayload)
	c.WillPayload = c.WillPayload[:0]
}
