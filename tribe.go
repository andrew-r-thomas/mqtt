package mqtt

import "log"

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
			log.Printf("got send msg\n")
			msgData := msg.MsgData.(SendMsgMsg)
			// TODO: figure out if we're going to send or not
			for _, s := range senders {
				s <- msgData.Data
			}
		}
	}
}
