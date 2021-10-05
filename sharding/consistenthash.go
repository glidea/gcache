package sharding

import (
	"bytes"
	"encoding/gob"
	"hash/crc32"
	"sort"
	"strconv"
)

// ConsistentHash implements Sharding
type ConsistentHash struct {
	sl       *selector
	replicas int
	fn       Hash
}

func NewConsistentHash(replicas int, fn Hash) *ConsistentHash {
	if fn == nil {
		fn = crc32.ChecksumIEEE
	}
	return &ConsistentHash{
		sl:       &selector{}, // just avoid nil panic in get
		fn:       fn,
		replicas: replicas,
	}
}

func (s *ConsistentHash) Get(nodes []string, key string) (node string, ok bool) {
	if len(nodes) == 0 {
		return "", false
	}
	newNodesId := toBytes(nodes)
	sl := s.sl
	if !eq(sl.nodesId, newNodesId) {
		sl = newSelector(s, newNodesId, nodes)
		s.sl = sl
	}
	return sl.choice(key), true
}

type selector struct {
	s       *ConsistentHash
	nodesId []byte
	circle  map[int]string
	idxes   []int // sorted
}

func newSelector(s *ConsistentHash, nodeId []byte, nodes []string) *selector {
	sl := &selector{
		s:       s,
		nodesId: nodeId,
		circle:  map[int]string{},
		idxes:   make([]int, 0, len(nodes)),
	}

	for _, node := range nodes {
		for i := 0; i < s.replicas; i++ {
			idx := int(s.fn([]byte(node + strconv.Itoa(i))))
			sl.circle[idx] = node
			sl.idxes = append(sl.idxes, idx)
		}
	}

	sort.Ints(sl.idxes)
	return sl
}

func (sl *selector) choice(key string) (node string) {
	idx := int(sl.s.fn([]byte(key)))
	i := sort.Search(len(sl.idxes), func(i int) bool {
		return sl.idxes[i] >= idx
	})

	return sl.circle[sl.idxes[i%len(sl.idxes)]]
}

func eq(a, b []byte) bool {
	c := 0
	for _, x := range a {
		c ^= int(x)
	}
	for _, x := range b {
		c ^= int(x)
	}
	return c == 0
}

func toBytes(v interface{}) []byte {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(v)
	if err != nil {
		return []byte{}
	}
	return b.Bytes()
}

type Hash func(data []byte) uint32
