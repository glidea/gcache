package sharding

// Sharding algorithm selector
type Sharding interface {
	Get(nodes []string, key string) (node string, ok bool)
}
