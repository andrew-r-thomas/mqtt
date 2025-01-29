package packets

// TODO: think about trimming down stuff to only the *data* that we need
// scratch space for, for example Sub/Unsub and Suback/Unsuback have the same
// exact fields, and we only use one at a time

type PacketLib struct {
	Publish     Publish
	Subscribe   Subscribe
	Suback      Suback
	Unsubscribe Unsubscribe
	Unsuback    Unsuback

	Properties Properties
}

func NewPacketLib() *PacketLib {
	pl := PacketLib{}
	pl.Zero()
	return &pl
}

func (pl *PacketLib) Zero() {
	pl.Publish.Zero()
	pl.Subscribe.Zero()
	pl.Suback.Zero()
	pl.Properties.Zero()
	pl.Unsubscribe.Zero()
	pl.Unsuback.Zero()
}
