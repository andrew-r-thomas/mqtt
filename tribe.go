package mqtt

type TribeMsgType byte

const (
	AddMember TribeMsgType = iota
	SendMsg
)

type TribeMsg struct {
	MsgType  TribeMsgType
	ClientId string
	MsgData  any
}

type AddMemberMsg struct {
	Sender chan<- []byte
}
type SendMsgMsg struct {
	Data []byte
}

func StartTribeManager(recv <-chan TribeMsg, bp *BufPool) {
	senders := []chan<- []byte{}
	cidMap := map[string]int{}

	for msg := range recv {
		switch msg.MsgType {
		case AddMember:
			cidMap[msg.ClientId] = len(senders)
			msgData := msg.MsgData.(AddMemberMsg)
			senders = append(senders, msgData.Sender)
		case SendMsg:
			msgData := msg.MsgData.(SendMsgMsg)
			// TODO: figure out if we're going to send or not
			for _, s := range senders {
				b := bp.GetBuf()
				n := copy(b, msgData.Data)
				if n < len(msgData.Data) {
					b = append(b, msgData.Data[n:]...)
				}
				s <- b[:len(msgData.Data)]
			}
			bp.ReturnBuf(msgData.Data)
		}
	}
}

/*

ok so the current problem is that we're sending the same slice to all of the
subscribers, so when their writer returns the byte slice to the bufferpool,
the pool sets the slice to zero, and the other subscribers get a packet of
zeros

*/
