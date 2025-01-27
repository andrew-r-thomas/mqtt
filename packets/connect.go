package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Connect struct {
	Id          string
	Username    string
	Password    []byte
	Keepalive   uint16
	Flags       byte
	WillTopic   string
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
	protocolName, offset, err := decodeUtf8(data)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	if protocolName != "MQTT" {
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

	offset, err = DecodeProps(props, rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	rest = rest[offset:]

	// client id
	connect.Id, offset, err = decodeUtf8(rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalConnPacket, err)
	}
	rest = rest[offset:]

	// will props
	if connect.Flags&0b00000100 != 0 {
		offset, err = DecodeProps(willProps, rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
		}
		rest = rest[offset:]

		connect.WillTopic, offset, err = decodeUtf8(rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
		}
		rest = rest[offset:]

		offset = decodeBinary(rest, connect.WillPayload)
		rest = rest[offset:]
	}

	// username
	if connect.Flags&0b10000000 != 0 {
		connect.Username, offset, err = decodeUtf8(rest)
		if err != nil {
			return fmt.Errorf("%v: %v", MalConnPacket, err)
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
	c.Username = ""
	clear(c.Password)
	c.Password = c.Password[:0]
	c.Id = ""
	c.Keepalive = 0
	c.Flags = 0
	c.WillTopic = ""
	clear(c.WillPayload)
	c.WillPayload = c.WillPayload[:0]
}
