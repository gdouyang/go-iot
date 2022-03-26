package wasmhost

type Context interface {
	ClientID() string
	UserName() string
	Done() <-chan struct{}
}
