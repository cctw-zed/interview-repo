package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type MyStruct struct {
	Value int
}

// ❌ 错误的实现（原始代码）
var (
	badInstance *MyStruct
	badOnce     sync.Once
)

func getBadInstance() *MyStruct {
	if badInstance != nil { // 没有内存屏障保护
		return badInstance
	}

	badOnce.Do(func() {
		badInstance = &MyStruct{Value: 42}
	})
	return badInstance
}

// ✅ 正确的实现1：纯 sync.Once
var (
	goodInstance *MyStruct
	goodOnce     sync.Once
)

func getGoodInstance() *MyStruct {
	goodOnce.Do(func() {
		goodInstance = &MyStruct{Value: 42}
	})
	return goodInstance
}

// ✅ 正确的实现2：使用 atomic.Value
var atomicInstance atomic.Value

func getAtomicInstance() *MyStruct {
	if v := atomicInstance.Load(); v != nil {
		return v.(*MyStruct)
	}

	newInstance := &MyStruct{Value: 42}
	atomicInstance.Store(newInstance)
	return newInstance
}

// ✅ 正确的实现3：使用 sync.RWMutex
var (
	mutexInstance *MyStruct
	mu            sync.RWMutex
)

func getMutexInstance() *MyStruct {
	mu.RLock()
	if mutexInstance != nil {
		defer mu.RUnlock()
		return mutexInstance
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	if mutexInstance == nil {
		mutexInstance = &MyStruct{Value: 42}
	}
	return mutexInstance
}

// 并发测试：检查是否会出现nil结果
func TestConcurrentAccess(t *testing.T) {
	fmt.Println("=== 并发安全测试 ===")

	tests := []struct {
		name string
		fn   func() *MyStruct
	}{
		{"错误实现", getBadInstance},
		{"sync.Once", getGoodInstance},
		{"atomic.Value", getAtomicInstance},
		{"sync.RWMutex", getMutexInstance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nilCount int32
			var wg sync.WaitGroup

			for i := 0; i < 1000; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					instance := tt.fn()
					if instance == nil {
						atomic.AddInt32(&nilCount, 1)
					}
				}()
			}

			wg.Wait()

			if nilCount > 0 {
				t.Errorf("%s: 发现 %d 个 nil 实例", tt.name, nilCount)
			} else {
				fmt.Printf("%s: 通过测试 ✅\n", tt.name)
			}
		})
	}
}

// 性能基准测试
func BenchmarkSingletonMethods(b *testing.B) {
	fmt.Println("\n=== 性能测试 ===")

	// 重置所有单例
	badInstance = nil
	goodInstance = nil
	atomicInstance = atomic.Value{}
	mutexInstance = nil

	b.Run("错误实现", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				getBadInstance()
			}
		})
	})

	b.Run("sync.Once", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				getGoodInstance()
			}
		})
	})

	b.Run("atomic.Value", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				getAtomicInstance()
			}
		})
	})

	b.Run("sync.RWMutex", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				getMutexInstance()
			}
		})
	})
}

// 演示函数
func ExampleSingletonComparison() {
	fmt.Println("演示单例模式的正确与错误实现")

	// 简单的并发测试
	fmt.Println("\n=== 简单并发测试 ===")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 测试错误实现
			bad := getBadInstance()
			good := getGoodInstance()

			fmt.Printf("Goroutine %d: bad=%v, good=%v\n", id, bad != nil, good != nil)

			time.Sleep(time.Millisecond)
		}(i)
	}

	wg.Wait()
}
