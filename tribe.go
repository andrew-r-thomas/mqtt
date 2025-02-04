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

func StartTribeManager(recv <-chan TribeMsg) {
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
			sIdx := cidMap[msg.ClientId]
			for i := 0; i < sIdx; i++ {
				senders[i] <- msgData.Data
			}
			for i := sIdx + 1; i < len(senders); i++ {
				senders[i] <- msgData.Data
			}
		}
	}
}
