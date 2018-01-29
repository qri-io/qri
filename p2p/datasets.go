package p2p

// import (
// 	"fmt"
// 	"github.com/qri-io/qri/repo"
// )

// // RequestDatasetInfo get's qri profile information from a PeerInfo
// func (n *QriNode) RequestDatasetInfo(ref *repo.DatasetRef) error {
// 	// Get this repo's profile information
// 	profile, err := n.Repo.Profile()
// 	if err != nil {
// 		fmt.Println("error getting node profile info:", err)
// 		return err
// 	}

// 	res, err := n.SendMessage(pinfo.ID, &Message{
// 		Type:    MtPeerInfo,
// 		Payload: profile,
// 	})
// 	if err != nil {
// 		fmt.Println("send profile message error:", err.Error())
// 		return err
// 	}

// 	if res.Phase == MpResponse {
// 		if err := n.handleProfileResponse(pinfo, res); err != nil {
// 			fmt.Println("profile response error", err.Error())
// 			return err
// 		}
// 	}

// 	return nil
// }
