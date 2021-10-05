package registry

import (
	"fmt"
	"gcache/singleflight"
	"github.com/samuel/go-zookeeper/zk"
	"sync"
	"time"
)

// ZK is a client for zookeeper
type ZK struct {
	mu         sync.RWMutex
	sfg        *singleflight.Group
	conn       *zk.Conn
	nodesCache map[string][]string
}

func NewZK(addr string) (z *ZK, err error) {
	conn, _, err := zk.Connect([]string{addr}, time.Second*5)
	if err != nil {
		return nil, err
	}
	_, err = conn.Create(pathPre, []byte{},
		0, zk.WorldACL(zk.PermAll))
	if err != nil && err.Error() != nodeExistErr {
		return nil, err
	}
	return &ZK{
		sfg:        singleflight.New(),
		conn:       conn,
		nodesCache: map[string][]string{},
	}, nil
}

func (z *ZK) Add(groupName, self string) error {
	err := z.createSelfNode(groupName, self)
	if err == nil {
		return nil
	}
	if err.Error() == nodeNoExistErr {
		err := z.createGroupNode(groupName)
		if err != nil {
			return err
		}
	}
	return z.createSelfNode(groupName, self)
}

func (z *ZK) Get(groupName string) (nodes []string, err error) {
	z.mu.RLock()
	if nodes, ok := z.nodesCache[groupName]; ok {
		z.mu.RUnlock()
		return nodes, nil
	}
	z.mu.RUnlock()

	ns, err := z.sfg.Do(groupName, func() (interface{}, error) {
		nodes, _, evtChan, err := z.conn.ChildrenW(pathPre + "/" + groupName)
		if err != nil {
			return nil, err
		}
		go func() {
			for {
				e := <-evtChan
				nodes, _, evtChan, err = z.conn.ChildrenW(pathPre + "/" + groupName)
				if err == nil && e.Type == zk.EventNodeChildrenChanged {
					z.atomicUpdateNodesCache(groupName, nodes)
				}
			}
		}()
		z.atomicUpdateNodesCache(groupName, nodes)
		return nodes, nil
	})
	return ns.([]string), err
}

func (z *ZK) createGroupNode(groupName string) error {
	return z.createNode(fmt.Sprintf(pathPre+"/%s", groupName), 0)
}

func (z *ZK) createSelfNode(groupName, self string) error {
	return z.createNode(fmt.Sprintf(pathPre+"/%s/%s", groupName, self), zk.FlagEphemeral)
}

func (z *ZK) createNode(path string, flags int32) error {
	_, err := z.conn.Create(path,
		[]byte{}, flags, zk.WorldACL(zk.PermAll))
	if err == nil || err.Error() == nodeExistErr {
		return nil
	}
	return err
}

func (z *ZK) atomicUpdateNodesCache(groupName string, nodes []string) {
	z.mu.Lock()
	z.nodesCache[groupName] = nodes
	z.mu.Unlock()
}

const (
	pathPre        = "/gcache"
	nodeExistErr   = "zk: node already exists"
	nodeNoExistErr = "zk: node does not exist"
)

// TODO 断开连接后支持重连，并重注册
