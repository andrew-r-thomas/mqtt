package packets

type Puback struct{}

func EncodePuback(
	p *Puback,
	props *Properties,
	buf []byte,
	scratch []byte,
) int {
	return 0
}
