package cache

import (
	"container/list"
)

// lru is a cache that implements lru eviction algorithm
type lru struct {
	m  map[string]*list.Element
	dl *list.List
}

func NewLRU() *lru {
	return &lru{
		m:  map[string]*list.Element{},
		dl: list.New(),
	}
}

func (l *lru) insert(k string, vw *vWarp) {
	e := l.dl.PushBack(&entry{k, vw})
	l.m[k] = e
}

func (l *lru) remove(k string) {
	e := l.m[k]
	if e == nil {
		return
	}
	delete(l.m, k)
	l.dl.Remove(e)
}

func (l *lru) update(k string, vw *vWarp) {
	e := l.m[k]
	e.Value.(*entry).vw = vw
	l.dl.MoveToBack(e)
}

func (l *lru) get(k string) (vw *vWarp, ok bool) {
	if e, ok := l.m[k]; ok {
		l.dl.MoveToBack(e)
		return e.Value.(*entry).vw, true
	}
	return
}

func (l *lru) onFull(overBytes int64) (decrBytes int64) {
	t := overBytes
	for overBytes > 0 {
		decr := l.removeOldest()
		if decr == 0 {
			break
		}
		overBytes -= decr
	}
	return t - overBytes
}

func (l *lru) removeOldest() (decrBytes int64) {
	front := l.dl.Front()
	if front == nil {
		return 0
	}
	l.dl.Remove(front)
	kv := front.Value.(*entry)
	delete(l.m, kv.key)
	return int64(len(kv.key) + len(kv.vw.v))
}

// entry is the type of lru.m.list.Element.Value
type entry struct {
	key string
	vw  *vWarp
}
