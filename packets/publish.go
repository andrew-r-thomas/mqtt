package packets

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Publish struct {
	topic    string
	packetId uint16
	props    PublishProps
	payload  []byte
}

type PublishProps struct {
	pfi byte         // payload format indicator
	mei uint32       // message expiry interval
	ta  uint16       // topic alias
	rt  string       // response topic
	cd  []byte       // correlation data
	up  []StringPair // user property
	si  uint32       // subscription identifier (var byte int)
	ct  string       // content type
}

var (
	MalPubPacket = errors.New("Malformed publish packet")
)

func DecodePublish(fh *FixedHeader, publish *Publish, data []byte) error {
	topic, l, err := decodeUtf8(data)
	if err != nil {
		return fmt.Errorf("%v: %v", MalPubPacket, err)
	}
	publish.topic = topic
	rest := data[l:]

	if fh.Flags&0b00000010 > 0 || fh.Flags&0b00000100 > 0 {
		// extract packet id
		publish.packetId = binary.BigEndian.Uint16(rest[:2])
		rest = rest[2:]
		l += 2
	}

	off, err := decodePublishProps(&publish.props, rest)
	if err != nil {
		return fmt.Errorf("%v: %v", MalPubPacket, err)
	}

	rest = rest[off:]
	l += off
	// TODO: would like copying here
	publish.payload = append(publish.payload, data[int(fh.RemLen)-l:]...)

	return nil
}

func decodePublishProps(props *PublishProps, data []byte) (int, error) {
	l, offset, err := decodeVarByteInt(data)
	if err != nil {
		return offset, fmt.Errorf("%v: %v", MalProps, err)
	}

	end := offset + int(l)
	for offset < end {
		switch data[offset] {
		case 1: // payload format indicator
			props.pfi = data[offset+1]
			offset += 2
		case 2: // message expiry interval
			props.mei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 35: // topic alias
			props.ta = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 8: // response topic
			rt, l, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.rt = rt
			offset += l + 1
		case 9: // correlation data
			l := decodeBinary(data[offset+1:], props.cd)
			offset += l + 1
		case 38: // user property
			namestr, noff, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			valstr, voff, err := decodeUtf8(data[offset+1+noff:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.up = append(props.up, StringPair{name: namestr, val: valstr})
			offset += 1 + noff + voff
		case 11: // subscription identifier
			si, off, err := decodeVarByteInt(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.si = si
			offset += off + 1
		case 3: // content type
			ct, off, err := decodeUtf8(data[offset+1:])
			if err != nil {
				return offset, fmt.Errorf("%v: %v", MalProps, err)
			}
			props.ct = ct
			offset += off + 1
		}
	}
	return offset, nil
}
