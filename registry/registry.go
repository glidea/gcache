package registry

// Registry is a client for registry
type Registry interface {
	Add(groupName string, self string) error
	Get(groupName string) (nodes []string, err error)
}
