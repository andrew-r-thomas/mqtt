package mqtt

import "sync/atomic"

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
	Sender Sender
}
type SendMsgMsg struct {
	Data []byte
}

type Sender struct {
	c    chan<- []byte
	live *atomic.Bool
}

func StartTribeManager(recv <-chan TribeMsg, bp *BufPool) {
	senders := map[string]Sender{}

	for msg := range recv {
		switch msg.MsgType {
		case AddMember:
			msgData := msg.MsgData.(AddMemberMsg)
			senders[msg.ClientId] = msgData.Sender
		case SendMsg:
			msgData := msg.MsgData.(SendMsgMsg)
			// TODO: figure out if we're going to send or not
			for id, s := range senders {
				if s.live.Load() {
					b := bp.GetBuf()
					n := copy(b, msgData.Data)
					if n < len(msgData.Data) {
						b = append(b, msgData.Data[n:]...)
					}
					s.c <- b[:len(msgData.Data)]
				} else {
					delete(senders, id)
				}
			}
			bp.ReturnBuf(msgData.Data)
		}
	}
}
