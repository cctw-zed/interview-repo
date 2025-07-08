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

// 存在内存泄漏的数据结构
type DataProcessor struct {
	mu    sync.RWMutex
	cache map[string]*LargeData
}

type LargeData struct {
	ID   string
	Data []byte // 模拟大数据
}

var processor = &DataProcessor{
	cache: make(map[string]*LargeData),
}

// 问题1: goroutine泄漏 - 没有正确使用context取消
func processDataWithLeak(id string) {
	go func() {
		for {
			// 模拟处理数据，但goroutine永远不会退出
			time.Sleep(100 * time.Millisecond)

			// 创建大量数据对象
			data := &LargeData{
				ID:   id,
				Data: make([]byte, 1024*1024), // 1MB数据
			}

			processor.mu.Lock()
			processor.cache[id] = data
			processor.mu.Unlock()

			// 问题：缓存永远不会被清理
		}
	}()
}

// 问题2: HTTP客户端连接泄漏
func httpClientLeak() {
	for i := 0; i < 10; i++ {
		go func(index int) {
			// 创建HTTP客户端但不复用，也不设置超时
			client := &http.Client{}

			for {
				// 持续发送请求但不处理响应体
				resp, err := client.Get("https://httpbin.org/delay/1")
				if err != nil {
					continue
				}
				// 问题：没有关闭response body
				_ = resp
				time.Sleep(time.Second)
			}
		}(i)
	}
}

// 问题3: slice泄漏
func sliceLeak() {
	largeSlice := make([]byte, 10*1024*1024) // 10MB

	// 从大slice中取小片段，但保持对原slice的引用
	go func() {
		for {
			// 这会导致整个10MB的slice无法被GC
			smallSlice := largeSlice[:10]

			// 模拟使用smallSlice
			fmt.Printf("Using slice of length: %d\n", len(smallSlice))

			time.Sleep(time.Second)
		}
	}()
}

// API处理函数
func leakHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = fmt.Sprintf("data_%d", time.Now().Unix())
	}

	// 触发内存泄漏
	processDataWithLeak(id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Started processing for ID: %s", id)))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	processor.mu.RLock()
	count := len(processor.cache)
	processor.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Cache size: %d items", count)))
}

func main() {
	// 启动各种泄漏场景
	go httpClientLeak()
	go sliceLeak()

	// 设置HTTP路由
	http.HandleFunc("/leak", leakHandler)
	http.HandleFunc("/status", statusHandler)

	// pprof已经通过import自动注册到默认ServeMux
	log.Println("Server starting on :8080")
	log.Println("pprof debug endpoints available at:")
	log.Println("  - http://localhost:8080/debug/pprof/")
	log.Println("  - http://localhost:8080/debug/pprof/heap")
	log.Println("  - http://localhost:8080/debug/pprof/goroutine")
	log.Fatal(http.ListenAndServe(":8080", nil))
	context.WithTimeout()
}
