package p2p

import (
	"fmt"
)

func (n *QriNode) handleProfileRequest(r *Message) *Message {
	p, err := n.repo.Profile()
	if err != nil {
		fmt.Println(err.Error())
	}
	return &Message{
		Type:    MtProfile,
		Phase:   MpResponse,
		Payload: p,
	}
}
