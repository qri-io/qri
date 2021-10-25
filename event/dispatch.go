package event

const (
	// ETDispatchMethodCall occurs when a method call is dispatched
	ETDispatchMethodCall = Type("dispatch:MethodCall")
)

// DispatchCall encapsulates fields from a dispatch method call
type DispatchCall struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}
