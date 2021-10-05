package singleflight

import "sync"

// promise is a container for safely get or put result
type promise struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

func newPromise() *promise {
	p := &promise{}
	p.wg.Add(1)
	return p
}

// get blockly until promise done
func (p *promise) get() (val interface{}, err error) {
	p.wg.Wait()
	return p.val, p.err
}

// done set result and notify waiter
func (p *promise) done(val interface{}, err error) {
	p.val = val
	p.err = err
	p.wg.Done()
}
