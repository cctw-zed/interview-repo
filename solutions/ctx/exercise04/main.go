package exercise04

import (
	"context"
	"fmt"
	"sync"
)

/*
第一题：并发任务的扇入（Fan-in）与优雅关闭
题目描述：
编写一个函数 Merge(ctx context.Context, sources ...<-chan int) <-chan int。
这个函数将多个 int 类型的 channel (sources) 合并成一个输出 channel。函数需要满足以下条件：
所有从 sources channels 中流出的数据，都应该被无序地发送到返回的输出 channel 中。
如果传入的 ctx 被取消，Merge 函数应该停止所有合并工作，并确保其内部启动的所有 goroutine 都能被清理和退出。
Merge 函数返回的那个输出 channel，应该在所有 sources channels 都被关闭并且所有数据都被合并完毕后，再被关闭。
考察点：
扇入（Fan-in）模式：如何将多个 channel 的输出合并到一个 channel。
sync.WaitGroup 的使用：如何等待一组不确定数量的 goroutine 全部执行完毕。
select 与 ctx.Done()：如何在 channel 操作中处理取消信号，实现优雅退出。
Goroutine 生命周期管理：如何确保在函数返回后，没有 goroutine 泄漏。
*/

func Merge(ctx context.Context, sources ...<-chan int) <-chan int {
	var (
		capacity int
		wg       sync.WaitGroup
		res      chan int
	)
	for _, source := range sources {
		wg.Add(1)
		capacity += cap(source)
	}
	res = make(chan int, capacity)

	for _, source := range sources {
		go func(source <-chan int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					fmt.Println("ctx canceled, exist")
					return
				case v, ok := <-source:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case res <- v:
					}
				}
			}
		}(source)
	}

	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}
