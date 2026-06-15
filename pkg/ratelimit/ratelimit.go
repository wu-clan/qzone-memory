// Package ratelimit 提供一个出站请求限速器：限制最大并发，并保证相邻请求之间
// 至少间隔 interval（外加随机抖动）。抓取与媒体下载共用同一个实例，统一克制对
// 腾讯的请求频率，避免触发风控。
package ratelimit

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// Limiter 并发安全的限速器。
type Limiter struct {
	sem      chan struct{}
	mu       sync.Mutex
	next     time.Time
	interval time.Duration
	jitter   time.Duration
}

// New 创建限速器。
//   - maxConcurrency <= 0 时退化为 1；
//   - interval / jitter < 0 时视为 0。
func New(interval, jitter time.Duration, maxConcurrency int) *Limiter {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	if interval < 0 {
		interval = 0
	}
	if jitter < 0 {
		jitter = 0
	}
	return &Limiter{
		sem:      make(chan struct{}, maxConcurrency),
		interval: interval,
		jitter:   jitter,
	}
}

// Acquire 获取一个许可：先占用并发名额，再等待到允许发出下一次请求的时刻。
// ctx 取消时返回其错误，并归还已占用的并发名额。每次成功的 Acquire 都应对应
// 一次 Release。
func (l *Limiter) Acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case l.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	wait := l.reserve()
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		l.Release() // 归还名额
		return ctx.Err()
	}
}

// reserve 预约下一个发送时隙，返回需要等待的时长。
func (l *Limiter) reserve() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	wait := time.Duration(0)
	if l.next.After(now) {
		wait = l.next.Sub(now)
	}

	gap := l.interval
	if l.jitter > 0 {
		gap += time.Duration(rand.Int63n(int64(l.jitter)))
	}
	l.next = now.Add(wait).Add(gap)
	return wait
}

// Release 归还一个并发名额。多余的 Release 会被安全忽略。
func (l *Limiter) Release() {
	select {
	case <-l.sem:
	default:
	}
}

// shared 进程级共享限速器：抓取与媒体下载共用同一个闸门，统一节流（见设计 D6）。
// 默认是一个串行、无间隔的限速器，应在启动时用 Configure 按配置替换。
var shared = New(0, 0, 1)

// Configure 用配置初始化共享限速器，应在程序启动时调用一次。
func Configure(interval, jitter time.Duration, maxConcurrency int) {
	shared = New(interval, jitter, maxConcurrency)
}

// Shared 返回进程级共享限速器。
func Shared() *Limiter {
	return shared
}
