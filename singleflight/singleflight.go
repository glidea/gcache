package singleflight

import "sync"

// Group prevent repeated execution
type Group struct {
	mu       sync.Mutex
	promises map[string]*promise
}

func New() *Group {
	return &Group{promises: map[string]*promise{}}
}

// Do execute and return the result of fn,
// ensure that only one goroutine execute fn for same key at same time.
// Meanwhile, if other goroutines Do for same key,
// they will wait for that goroutine to complete fn,
// and then get the result directly.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if p, ok := g.promises[key]; ok {
		g.mu.Unlock()
		return p.get()
	}
	p := newPromise()
	g.promises[key] = p
	g.mu.Unlock()

	val, err := fn()
	p.done(val, err)
	g.mu.Lock()
	delete(g.promises, key)
	g.mu.Unlock()
	return val, err
}
