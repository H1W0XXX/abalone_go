// internal/search/cancel.go

package search

import "sync/atomic"

type cancelToken struct{ f int32 }

func (c *cancelToken) Abort() {
	atomic.StoreInt32(&c.f, 1)
}
func (c *cancelToken) IsAborted() bool {
	return atomic.LoadInt32(&c.f) == 1
}

// ↓↓↓ 加两个包级函数 ↓↓↓
var defaultToken = &cancelToken{}

// 直接调用这两个函数，就等于调用 defaultToken 的方法
func Abort() {
	defaultToken.Abort()
}
func IsAborted() bool {
	return defaultToken.IsAborted()
}
