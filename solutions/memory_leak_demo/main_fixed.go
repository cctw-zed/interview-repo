package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

// 修复后的数据结构
type FixedDataProcessor struct {
	mu    sync.RWMutex
	cache map[string]*FixedLargeData
}

type FixedLargeData struct {
	ID       string
	Data     []byte
	ExpireAt time.Time
}

var fixedProcessor = &FixedDataProcessor{
	cache: make(map[string]*FixedLargeData),
}

// 修复1: 使用context控制goroutine生命周期
func processDataFixed(ctx context.Context, id string) {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// 正确退出goroutine
				return
			case <-ticker.C:
				// 创建数据对象，设置过期时间
				data := &FixedLargeData{
					ID:       id,
					Data:     make([]byte, 1024*1024), // 1MB数据
					ExpireAt: time.Now().Add(5 * time.Minute),
				}

				fixedProcessor.mu.Lock()
				fixedProcessor.cache[id] = data
				fixedProcessor.mu.Unlock()
			}
		}
	}()
}

// 修复2: 正确使用HTTP客户端
func httpClientFixed(ctx context.Context) {
	// 使用连接池复用HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	for i := 0; i < 10; i++ {
		go func(index int) {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// 正确处理HTTP请求
					resp, err := client.Get("https://httpbin.org/delay/1")
					if err != nil {
						continue
					}
					// 关键：正确关闭response body
					resp.Body.Close()
				}
			}
		}(i)
	}
}

// 修复3: 正确处理slice引用
func sliceFixed(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 创建大slice
				largeSlice := make([]byte, 10*1024*1024) // 10MB

				// 正确方式：复制需要的部分，避免引用整个大slice
				smallSlice := make([]byte, 10)
				copy(smallSlice, largeSlice[:10])

				// 现在largeSlice可以被GC回收
				largeSlice = nil

				fmt.Printf("Using slice of length: %d\n", len(smallSlice))
			}
		}
	}()
}

// 缓存清理goroutine
func startCacheCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fixedProcessor.mu.Lock()
				now := time.Now()
				for id, data := range fixedProcessor.cache {
					if now.After(data.ExpireAt) {
						delete(fixedProcessor.cache, id)
					}
				}
				fixedProcessor.mu.Unlock()
			}
		}
	}()
}

// 全局context用于控制所有goroutine
var globalCtx context.Context
var globalCancel context.CancelFunc

// API处理函数
func fixedLeakHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = fmt.Sprintf("data_%d", time.Now().Unix())
	}

	// 使用修复版本
	processDataFixed(globalCtx, id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Started processing for ID: %s", id)))
}

func fixedStatusHandler(w http.ResponseWriter, r *http.Request) {
	fixedProcessor.mu.RLock()
	count := len(fixedProcessor.cache)
	fixedProcessor.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Cache size: %d items", count)))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	globalCancel()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All goroutines stopped"))
}

func runFixed() {
	// 创建全局context
	globalCtx, globalCancel = context.WithCancel(context.Background())

	// 启动修复版本的各种功能
	go httpClientFixed(globalCtx)
	go sliceFixed(globalCtx)
	go startCacheCleanup(globalCtx)

	// 设置HTTP路由
	http.HandleFunc("/leak", fixedLeakHandler)
	http.HandleFunc("/status", fixedStatusHandler)
	http.HandleFunc("/stop", stopHandler)

	// pprof已经通过import自动注册到默认ServeMux
	log.Println("Fixed server starting on :8081")
	log.Println("pprof debug endpoints available at:")
	log.Println("  - http://localhost:8081/debug/pprof/")
	log.Println("  - http://localhost:8081/debug/pprof/heap")
	log.Println("  - http://localhost:8081/debug/pprof/goroutine")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
