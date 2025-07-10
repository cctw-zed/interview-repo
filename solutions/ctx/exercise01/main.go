package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

/*
题目描述：
编写一个程序，main 函数启动一个 worker goroutine。worker goroutine 会在一个无限循环中每秒打印一次 "working..."。
main 函数在运行 3 秒后，需要通知 worker goroutine 停止工作并优雅地退出。
worker 在接收到退出信号后，应打印 "worker gracefully stopped" 然后退出。
考察点：
使用 context.WithCancel 来创建一个可取消的 context。
在 main goroutine 中调用 cancel() 函数来发送取消信号。
在 worker goroutine 中使用 select 和 ctx.Done() 来监听取消事件。
*/

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker(ctx)
	}()

	time.Sleep(3 * time.Second)
	cancel()
	wg.Wait()
	fmt.Println("main gracefully stopped")

}

func worker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("worker gracefully stopped")
			return
		case <-ticker.C:
			fmt.Println("working...")
		}
	}
}
