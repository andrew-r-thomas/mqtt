package packets

import (
	"encoding/binary"
	"errors"
	"strings"
)

type Properties struct {
	// if zero value, don't send
	Mps uint32          // maximum packet size
	Ad  []byte          // authentication data
	Am  strings.Builder // authentication method
	Rm  uint16          // receive maximum
	Aci strings.Builder // assigned client identifier
	Tam uint16          // topic alias maximum
	Rs  strings.Builder // reason string
	Sr  strings.Builder // server reference
	Ri  strings.Builder // response information
	Up  []StringPair    // user property
	Sei uint32          // session expiry interval
	Rri byte            // request response information
	Ska uint16          // server keep alive
	Wdi uint32          // will delay interval
	Pfi byte            // payload format indicator
	Mei uint32          // message expiry interval
	Ct  strings.Builder // content type
	Rt  strings.Builder // response topic
	Cd  []byte          // correlation data
	Si  uint32          // subscription identifier (var byte int)
	Ta  uint16          // topic alias

	// if 1, don't send
	Wsa byte // wildcard subscription available
	Sia byte // subscription identifier available
	Ssa byte // shared subscription available
	Ra  byte // retain available
	Rpi byte // request problem information

	// if 2, don't send
	Mq byte // maximum qos
}

var MalProps = errors.New("Malformed properties")
var InvalidPropId = errors.New("Invalid property identifier")

func DecodeProps(p *Properties, data []byte) int {
	l, offset, err := decodeVarByteInt(data)
	if err != nil {
		return -1
	}

	end := offset + int(l)
	for offset < end {
		switch data[offset] {
		case 1: // payload format indicator
			p.Pfi = data[offset+1]
			offset += 2
		case 2: // message expiry interval
			p.Mei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 3: // content type
			off := decodeUtf8(data[offset+1:], p.Ct)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 8: // response topic
			off := decodeUtf8(data[offset+1:], p.Rt)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 9: // correlation data
			off := decodeBinary(data[offset+1:], p.Cd)
			offset += off + 1
		case 11: // subscription identifier
			si, off, err := decodeVarByteInt(data[offset+1:])
			if err != nil {
				return -1
			}
			p.Si = si
			offset += off + 1
		case 17: // session expiry interval
			p.Sei = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 18: // assigned client identifier
			off := decodeUtf8(data[offset+1:], p.Aci)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 19: // server keep alive
			p.Ska = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 21: // authentication method
			off := decodeUtf8(data[offset+1:], p.Am)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 22: // authentication data
			off := decodeBinary(data[offset+1:], p.Ad)
			offset += off + 1
		case 23: // request problem information
			p.Rpi = data[offset+1]
			offset += 2
		case 24: // will delay interval
			p.Wdi = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 25: // request response information
			p.Rri = data[offset+1]
			offset += 2
		case 26: // response information
			off := decodeUtf8(data[offset+1:], p.Ri)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 28: // server reference
			off := decodeUtf8(data[offset+1:], p.Sr)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 31: // reason string
			off := decodeUtf8(data[offset+1:], p.Rs)
			if off == -1 {
				return -1
			}
			offset += off + 1
		case 33: // receive maximum
			p.Rm = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 34: // topic alias maximum
			p.Tam = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 35: // topic alias
			p.Ta = binary.BigEndian.Uint16(data[offset+1 : offset+3])
			offset += 3
		case 36: // maximum qos
			p.Mq = data[offset+1]
			offset += 2
		case 37: // retain available
			p.Ra = data[offset+1]
			offset += 2
		case 38: // user property
			var sp StringPair
			off := decodeUtf8(data[offset+1:], sp.name)
			if off == -1 {
				return -1
			}
			offset += off + 1
			off = decodeUtf8(data[offset:], sp.val)
			if off == -1 {
				return -1
			}
			offset += off + 1
			p.Up = append(p.Up, sp)
		case 39: // maximum packet size
			p.Mps = binary.BigEndian.Uint32(data[offset+1 : offset+5])
			offset += 5
		case 40: // wildcard subscription available
			p.Wsa = data[offset+1]
			offset += 2
		case 41: // subscription identifier available
			p.Sia = data[offset+1]
			offset += 2
		case 42: // shared subscription available
			p.Ssa = data[offset+1]
			offset += 2
		}
	}

	return offset
}

func EncodeProps(p *Properties, buf []byte, scratch []byte) int {
	l := 0
	if p.Mps != 0 {
		scratch[l] = 39
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Mps)
		l += 5
	}
	if len(p.Ad) != 0 {
		scratch[l] = 22
		binary.BigEndian.PutUint16(scratch[l+1:l+3], uint16(len(p.Ad)))
		copy(scratch[l+3:], p.Ad)
		l += len(p.Ad) + 3
	}
	if p.Am.Len() > 0 {
		scratch[l] = 21
		ll := encodeUtf8(scratch[l+1:], p.Am.String())
		l += ll + 1
	}
	if p.Rm != 0 {
		scratch[l] = 33
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Rm)
		l += 3
	}
	if p.Aci.Len() > 0 {
		scratch[l] = 18
		ll := encodeUtf8(scratch[l+1:], p.Aci.String())
		l += ll + 1
	}
	if p.Tam != 0 {
		scratch[l] = 34
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Tam)
		l += 3
	}
	if p.Rs.Len() > 0 {
		scratch[l] = 31
		ll := encodeUtf8(scratch[l+1:], p.Rs.String())
		l += ll + 1
	}
	if p.Sr.Len() > 0 {
		scratch[l] = 28
		ll := encodeUtf8(scratch[l+1:], p.Sr.String())
		l += ll + 1
	}
	if p.Ri.Len() > 0 {
		scratch[l] = 26
		ll := encodeUtf8(scratch[l+1:], p.Ri.String())
		l += ll + 1
	}
	for _, up := range p.Up {
		scratch[l] = 38
		lln := encodeUtf8(scratch[l+1:], up.name.String())
		llv := encodeUtf8(scratch[lln+1:], up.val.String())
		l += lln + llv + 1
	}
	if p.Sei != 0 {
		scratch[l] = 17
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Sei)
		l += 5
	}
	if p.Rri != 0 {
		scratch[l] = 25
		scratch[l+1] = p.Rri
		l += 2
	}
	if p.Ska != 0 {
		scratch[l] = 19
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Ska)
		l += 3
	}
	if p.Wdi != 0 {
		scratch[l] = 24
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Wdi)
		l += 5
	}
	if p.Pfi != 0 {
		scratch[l] = 1
		scratch[l+1] = p.Pfi
		l += 2
	}
	if p.Mei != 0 {
		scratch[l] = 2
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Mei)
		l += 5
	}
	if p.Ct.Len() > 0 {
		scratch[l] = 3
		ll := encodeUtf8(scratch[l+1:], p.Ct.String())
		l += ll + 1
	}
	if p.Rt.Len() > 0 {
		scratch[l] = 8
		ll := encodeUtf8(scratch[l+1:], p.Rt.String())
		l += ll + 1
	}
	if len(p.Cd) != 0 {
		scratch[l] = 9
		copy(scratch[l+1:], p.Cd)
		l += len(p.Cd) + 1
	}
	if p.Si != 0 {
		scratch[l] = 11
		binary.BigEndian.PutUint32(scratch[l+1:l+5], p.Si)
		l += 5
	}
	if p.Ta != 0 {
		scratch[l] = 35
		binary.BigEndian.PutUint16(scratch[l+1:l+3], p.Ta)
		l += 3
	}
	if p.Wsa != 1 {
		scratch[l] = 40
		scratch[l+1] = p.Wsa
		l += 2
	}
	if p.Sia != 1 {
		scratch[l] = 41
		scratch[l+1] = p.Sia
		l += 2
	}
	if p.Ssa != 1 {
		scratch[l] = 42
		scratch[l+1] = p.Ssa
		l += 2
	}
	if p.Ra != 1 {
		scratch[l] = 37
		scratch[l+1] = p.Ra
		l += 2
	}
	if p.Rpi != 1 {
		scratch[l] = 23
		scratch[l+1] = p.Rpi
		l += 2
	}
	if p.Mq != 2 {
		scratch[l] = 36
		scratch[l+1] = p.Mq
		l += 2
	}

	bl := encodeVarByteInt(buf, l)
	copy(buf[bl:], scratch[:l])
	clear(scratch)

	return bl + l
}

func (p *Properties) Zero() {
	p.Mps = 0
	clear(p.Ad)
	p.Ad = p.Ad[:0]
	p.Am.Reset()
	p.Rm = 0
	p.Aci.Reset()
	p.Tam = 0
	p.Rs.Reset()
	p.Sr.Reset()
	p.Ri.Reset()
	clear(p.Up)
	p.Up = p.Up[:0]
	p.Sei = 0
	p.Rri = 0
	p.Ska = 0
	p.Wdi = 0
	p.Pfi = 0
	p.Mei = 0
	p.Ct.Reset()
	p.Rt.Reset()
	clear(p.Cd)
	p.Cd = p.Cd[:0]
	p.Si = 0
	p.Ta = 0

	p.Wsa = 1
	p.Sia = 1
	p.Ssa = 1
	p.Ra = 1
	p.Rpi = 1

	p.Mq = 2
}
