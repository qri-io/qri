package p2p

// MtLogDiff gets info on a dataset
const MtLogDiff = MsgType("log_diff")

func (n *QriNode) handleLogDiff(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":
		res, err := msg.UpdateJSON(struct {
			Err string
		}{
			"this peer does not support log_diff",
		})
		if err != nil {
			return
		}
		res = res.WithHeaders("phase", "response")
		if err := ws.sendMessage(res); err != nil {
			log.Debug(err.Error())
			return
		}
	}

	return
}
