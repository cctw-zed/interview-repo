package main

import (
	"context"
	"sync"
	"time"
)

// TokenBucket 实现了一个令牌桶算法的速率限制器
type TokenBucket struct {
	ratePerSecond int       // 每秒生成的令牌数
	maxTokens     float64   // 桶的最大容量
	currentTokens float64   // 当前桶中的令牌数
	lastTimestamp time.Time // 上次取令牌的时间
	mu            sync.Mutex
}

// NewTokenBucket 创建一个新的令牌桶实例
func NewTokenBucket(ratePerSecond int) *TokenBucket {
	maxTokens := float64(ratePerSecond)
	return &TokenBucket{
		ratePerSecond: ratePerSecond,
		maxTokens:     maxTokens,
		currentTokens: maxTokens, // 启动时令牌桶是满的
		lastTimestamp: time.Now(),
	}
}

// WaitAndTake 会阻塞直到从桶中获取一个令牌，或者 context 被取消。
// 如果成功获取令牌，返回 nil。如果因 context 取消而中断，返回 ctx.Err()。
func (tb *TokenBucket) WaitAndTake(ctx context.Context) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 使用循环，以应对等待后可能的状态变化
	for {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 计算并补充令牌
		now := time.Now()
		tokensGenerated := now.Sub(tb.lastTimestamp).Seconds() * float64(tb.ratePerSecond)
		tb.currentTokens += tokensGenerated
		if tb.currentTokens > tb.maxTokens {
			tb.currentTokens = tb.maxTokens
		}
		tb.lastTimestamp = now

		// 如果令牌足够，取走一个并返回
		if tb.currentTokens >= 1 {
			tb.currentTokens--
			return nil
		}

		// 如果令牌仍然不足，计算需要等待多久
		timeToWait := time.Duration((1-tb.currentTokens)/float64(tb.ratePerSecond)) * time.Second

		// 在等待时，同时监听 context 的取消信号
		tb.mu.Unlock()
		select {
		case <-time.After(timeToWait):
			// 等待结束，重新加锁并进入下一次循环检查
			tb.mu.Lock()
			continue
		case <-ctx.Done():
			// 在等待期间被取消，重新加锁以保护 defer 的 Unlock，然后返回错误
			tb.mu.Lock()
			return ctx.Err()
		}
	}
}
