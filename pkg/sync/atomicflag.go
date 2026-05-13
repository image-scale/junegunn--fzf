package sync

import (
	"sync/atomic"
)

type AtomicFlag struct {
	val int32
}

func NewAtomicFlag(initial bool) *AtomicFlag {
	v := int32(0)
	if initial {
		v = 1
	}
	return &AtomicFlag{val: v}
}

func (f *AtomicFlag) Get() bool {
	return atomic.LoadInt32(&f.val) == 1
}

func (f *AtomicFlag) Set(b bool) bool {
	v := int32(0)
	if b {
		v = 1
	}
	atomic.StoreInt32(&f.val, v)
	return b
}
