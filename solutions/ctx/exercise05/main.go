package main

import (
	"context"
	"sync"
)

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
// 任务队列长度要足够大，否则submit会一直阻塞
type WorkerPool struct {
	workerCount int
	taskChan    chan Job
	rateChan    chan Job
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	bucket      *TokenBucket // 引用独立的 TokenBucket
}

// taskChan的长度
const (
	taskChanCap = 100
	// 令牌桶的最大容量可以与速率挂钩，比如允许一秒的突发量
)

// NewWorkerPool 工作池初始化函数
func NewWorkerPool(ctx context.Context, workerCount int, ratePerSecond int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	// 初始化任务队列
	workerPool := &WorkerPool{
		workerCount: workerCount,
		taskChan:    make(chan Job, taskChanCap),
		rateChan:    make(chan Job, taskChanCap),
		wg:          sync.WaitGroup{},
		ctx:         ctx,
		cancel:      cancel,
		bucket:      NewTokenBucket(ratePerSecond),
	}

	// 创建一个中间chan控制速率
	workerPool.wg.Add(1)
	go workerPool.dispatcher()

	// 创建workerCount个goroutine监听任务队列
	for i := 0; i < workerPool.workerCount; i++ {
		workerPool.wg.Add(1)
		go workerPool.worker()
	}

	return workerPool
}

// dispatcher 是核心的调度器
// 它从 taskChan 中获取任务，并根据速率限制将任务推送到 rateChan
func (w *WorkerPool) dispatcher() {
	defer w.wg.Done()
	// 当 dispatcher 退出时，意味着不会再有任务被分发，可以安全关闭 rateChan
	defer close(w.rateChan)

	for {
		// 优先检查 context 是否被取消
		select {
		case <-w.ctx.Done():
			// 强制取消
			return
		case job, ok := <-w.taskChan:
			if !ok {
				// taskChan被关闭，这是优雅关闭的信号
				return
			}
			// 正常接收到任务，等待令牌
			if err := w.bucket.WaitAndTake(w.ctx); err != nil {
				// 在等待令牌时被强制取消
				return
			}

			// 将任务发送给 worker，同时也要能响应 shutdown 信号
			select {
			case w.rateChan <- job:
			case <-w.ctx.Done():
				// 在发送给 worker 时被强制取消
				return
			}
		}
	}
}

// worker 从 rateChan 中消费任务并执行
func (w *WorkerPool) worker() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			// 强制取消
			return
		case task, ok := <-w.rateChan:
			if !ok {
				// rateChan 被关闭，正常退出
				return
			}
			// 执行任务
			task()
		}
	}
}

// Submit 向工作池提交一个任务。如果任务队列已满，此方法可以阻塞。
func (w *WorkerPool) Submit(job Job) {
	select {
	case <-w.ctx.Done():
		return
	case w.taskChan <- job:
	}
}

// Shutdown 优雅地关闭工作池。它应该停止接收新任务，并等待所有已在队列中和正在执行的任务完成后再返回。
func (w *WorkerPool) Shutdown() {
	// 1. 关闭 taskChan，此后 Submit 方法会阻塞或 panic(如果池已关闭)
	// dispatcher 在读完所有 taskChan 中的任务后会自动退出。
	close(w.taskChan)

	// 2. 等待所有 goroutine (dispatcher 和 workers) 优雅退出
	w.wg.Wait()
}
