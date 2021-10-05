package gcache

import (
	"fmt"
	"gcache/cache"
	"gcache/protocol"
	"gcache/registry"
	"gcache/sharding"
	"gcache/singleflight"
	"time"
)

// GCache is the core struct of the framework,
// a ServerBootstrap for users
type GCache struct {
	groups   map[string]*group // which groups this node joined
	registry registry.Registry
	protocol protocol.Protocol
	self     string // addr of this node. eg: 200.198.131.111
	err      error  // an error that occurs when setting parameters. It is deferred until the startup phase
}

func New() *GCache {
	return &GCache{groups: map[string]*group{}}
}

func (c *GCache) Group(name string, maxBytes int64, evictionAlgo string, timeout time.Duration, shardingAlgo string, db DBGetter) *GCache {
	g := &group{
		c:    c,
		name: name,
		db:   db,
		sfg:  singleflight.New(),
	}
	switch evictionAlgo {
	case EvictionLru:
		g.cache = cache.New(cache.NewLRU(), timeout, maxBytes)
	default:
		c.err = fmt.Errorf("eviction no support" + evictionAlgo)
	}
	switch shardingAlgo {
	case ShardingConsistenthash:
		g.sharding = sharding.NewConsistentHash(100, nil)
	default:
		c.err = fmt.Errorf("sharding no support" + shardingAlgo)
	}
	c.groups[name] = g
	return c
}

func (c *GCache) Registry(what, addr string) *GCache {
	switch what {
	case RegistryZK:
		c.registry, c.err = registry.NewZK(addr)
	default:
		c.err = fmt.Errorf("registry no support " + what)
	}
	return c
}

func (c *GCache) Protocol(p string) *GCache {
	switch p {
	case ProtocolHTTP:
		c.protocol = protocol.NewHTTP(protocol.ResourceGetFunc(func(groupName string, key string) (val []byte, err error) {
			if g, ok := c.groups[groupName]; ok {
				return g.Get(key)
			}
			return nil, nil
		}))
	default:
		c.err = fmt.Errorf("protocol no support " + p)
	}
	return c
}

func (c *GCache) Start(addr string) error {
	switch {
	case c.err != nil:
		return c.err
	case len(c.groups) == 0:
		return fmt.Errorf("group empty")
	case c.registry == nil:
		return fmt.Errorf("registry nil")
	case c.protocol == nil:
		return fmt.Errorf("protocol nil")
	}
	if addr[:1] == ":" {
		addr = "127.0.0.1" + addr
	}
	c.self = addr
	for _, g := range c.groups {
		err := c.registry.Add(g.name, c.self)
		if err != nil {
			return err
		}
	}
	return c.protocol.Serve(c.self)
}

type group struct {
	c        *GCache
	name     string
	cache    *cache.Cache
	db       DBGetter
	sharding sharding.Sharding
	sfg      *singleflight.Group
}

func (g *group) Get(key string) (val []byte, err error) {
	if val, ok := g.cache.Get(key); ok {
		return val, nil
	}

	v, err := g.sfg.Do(key, func() (interface{}, error) {
		nodes, err := g.c.registry.Get(g.name)
		if err != nil {
			return nil, err
		}
		node, ok := g.sharding.Get(nodes, key)
		if ok && node != g.c.self {
			return g.c.protocol.GetFromRemote(node, g.name, key)
		}
		return g.getFromLocal(key)
	})

	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (g *group) getFromLocal(key string) (val []byte, err error) {
	val, err = g.db.Get(key)
	if err != nil {
		return nil, err
	}
	g.cache.Add(key, val)
	return val, err
}

type DBGetter interface {
	Get(key string) (val []byte, err error)
}

type DBGetterFunc func(key string) (val []byte, err error)

func (f DBGetterFunc) Get(key string) (val []byte, err error) {
	return f(key)
}

const (
	EvictionLru            = "lru"
	ShardingConsistenthash = "consistenthash"
	RegistryZK             = "zk"
	ProtocolHTTP           = "http"
)
