package protocol

// Protocol is a rpc server and client distinguished by protocol
type Protocol interface {
	Serve(addr string) error
	GetFromRemote(addr, groupName, key string) (val []byte, err error)
}

type absProtocol struct {
	r ServeResource
}

type ServeResource interface {
	Get(groupName string, key string) (val []byte, err error)
}

type ResourceGetFunc func(groupName string, key string) (val []byte, err error)

func (f ResourceGetFunc) Get(groupName string, key string) (val []byte, err error) {
	return f(groupName, key)
}
