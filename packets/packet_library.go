package packets

type PacketLib struct {
	FixedHeader FixedHeader

	Publish   Publish
	Subscribe Subscribe
	Suback    Suback

	Properties Properties
}

func NewPacketLib() *PacketLib {
	pl := PacketLib{}
	pl.Zero()
	return &pl
}

func (pl *PacketLib) Zero() {
	pl.FixedHeader.Zero()
	pl.Publish.Zero()
	pl.Subscribe.Zero()
	pl.Suback.Zero()
	pl.Properties.Zero()
}
