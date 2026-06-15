package ratelimit

import (
	"context"
	"testing"
	"time"
)

// 相邻请求应被 interval 拉开：3 次串行获取至少经过约 2 个 interval。
func TestAcquireRespectsInterval(t *testing.T) {
	l := New(40*time.Millisecond, 0, 1)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := l.Acquire(ctx); err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		l.Release()
	}

	if elapsed := time.Since(start); elapsed < 70*time.Millisecond {
		t.Fatalf("期望相邻请求间隔累计 >= ~80ms，实际 %v", elapsed)
	}
}

// 并发名额占满后，新的获取应阻塞直至有名额或上下文超时。
func TestMaxConcurrencyBlocks(t *testing.T) {
	l := New(0, 0, 2)
	ctx := context.Background()

	if err := l.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.Acquire(ctx); err != nil {
		t.Fatal(err)
	}

	blocked, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := l.Acquire(blocked); err == nil {
		t.Fatal("并发已满，第三次获取本应阻塞至超时")
	}

	l.Release()
	if err := l.Acquire(ctx); err != nil {
		t.Fatalf("释放名额后应能再次获取: %v", err)
	}
}

// 已取消的上下文应立即返回错误。
func TestAcquireCanceledContext(t *testing.T) {
	l := New(0, 0, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := l.Acquire(ctx); err == nil {
		t.Fatal("已取消的上下文应返回错误")
	}
}
