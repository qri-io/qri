package p2p

import (
	"context"
	disc "github.com/libp2p/go-libp2p/p2p/discovery"
	"time"
)

// TODO - major work in progress
func (qn *QriNode) DiscoverPeers(notifee disc.Notifee) error {
	_, err := disc.NewMdnsService(context.Background(), qn.Host, time.Second*10)
	return err
}
