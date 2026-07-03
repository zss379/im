package sse

import (
	"sync/atomic"
)

// Pool 管理 SSE 连接数上限
type Pool struct {
	maxConns   int64
	active     int64
}

func NewPool(maxConns int) *Pool {
	return &Pool{
		maxConns: int64(maxConns),
	}
}

// TryAcquire 尝试获取一个连接槽位。如果达到上限返回 false
func (p *Pool) TryAcquire() bool {
	for {
		current := atomic.LoadInt64(&p.active)
		if current >= p.maxConns {
			return false
		}
		if atomic.CompareAndSwapInt64(&p.active, current, current+1) {
			return true
		}
	}
}

// Release 释放一个连接槽位
func (p *Pool) Release() {
	atomic.AddInt64(&p.active, -1)
}

// ActiveCount 当前活跃连接数
func (p *Pool) ActiveCount() int64 {
	return atomic.LoadInt64(&p.active)
}

// MaxConns 最大连接数
func (p *Pool) MaxConns() int64 {
	return p.maxConns
}
