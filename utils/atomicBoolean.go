package utils

import "sync/atomic"

type TAtomBool struct{ Flag int32 }

func (b *TAtomBool) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.Flag), int32(i))
}

func (b *TAtomBool) Get() bool {
	if atomic.LoadInt32(&(b.Flag)) != 0 {
		return true
	}
	return false
}
