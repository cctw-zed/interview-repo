package main

import "context"

/*
第二题：带速率限制的并发工作池
题目描述：
实现一个并发工作池 WorkerPool，它可以并发执行任务，但同时要受到全局的速率限制。
你需要实现以下结构和方法：
type Job func(): 定义要执行的任务类型。
NewWorkerPool(ctx context.Context, workerCount int, ratePerSecond int) *WorkerPool: 构造函数。
workerCount：池中有多少个 worker goroutine。
ratePerSecond：所有 worker 每秒钟总共执行的 Job 不能超过这个数量。
ctx：用于整体关闭整个工作池。当此 ctx 被取消时，所有 worker 都应停止接收新任务。
Submit(job Job): 向工作池提交一个任务。如果任务队列已满，此方法可以阻塞。
Shutdown(): 优雅地关闭工作池。它应该停止接收新任务，并等待所有已在队列中和正在执行的任务完成后再返回。
工作流程提示：
Worker 从一个内部的 jobs channel 中获取任务。
为了实现速率限制，可以创建一个单独的 goroutine，它使用 time.NewTicker 来控制任务的分发节奏。
或者，每个 worker 在处理任务前，都去询问一个共享的“令牌桶”或“节拍器”。
考察点：
工作池（Worker Pool）模式：经典的并发设计模式。
速率限制：如何使用 time.Ticker 或类似机制来控制事件发生的频率。
生产者-消费者模型：Submit 是生产者，worker 是消费者。
复杂的优雅关闭：如何协调多个 worker goroutine、一个任务分发器以及主 goroutine，在收到 context 取消信号后，按部就班地完成收尾工作。
*/

// Job 自定义任务类型
type Job func()

// WorkerPool 工作池
type WorkerPool struct {
	workerCount   int
	ratePerSecond int
	ctx           context.Context
}

// NewWorkerPool 工作池初始化函数
func NewWorkerPool(ctx context.Context, workerCount int, ratePerSecond int) *WorkerPool {
	return &WorkerPool{
		workerCount:   workerCount,
		ratePerSecond: ratePerSecond,
		ctx:           ctx,
	}
}

// Submit 向工作池提交一个任务。如果任务队列已满，此方法可以阻塞。
func (w *WorkerPool) Submit(job Job) {

}

// Shutdown 优雅地关闭工作池。它应该停止接收新任务，并等待所有已在队列中和正在执行的任务完成后再返回。
func (w *WorkerPool) Shutdown() {

}
