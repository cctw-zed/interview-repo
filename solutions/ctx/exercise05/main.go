package main

import (
	"context"
	"sync"
	"time"
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

	// 令牌桶
	ratePerSecond int        // 令牌生成速率
	lastTimestamp time.Time  // 上次获取令牌的时间
	currentTokens float64    // 令牌数
	maxTokens     float64    // 最大令牌数
	mu            sync.Mutex // 获取令牌锁
}

// taskChan的长度
const (
	taskChanCap = 100
	// 令牌桶的最大容量可以与速率挂钩，比如允许一秒的突发量
)

// NewWorkerPool 工作池初始化函数
func NewWorkerPool(ctx context.Context, workerCount int, ratePerSecond int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	maxTokens := float64(ratePerSecond) // 允许1s的突发
	// 初始化任务队列
	workerPool := &WorkerPool{
		workerCount:   workerCount,
		ratePerSecond: ratePerSecond,
		taskChan:      make(chan Job, taskChanCap),
		rateChan:      make(chan Job, taskChanCap),
		wg:            sync.WaitGroup{},
		ctx:           ctx,
		cancel:        cancel,
		maxTokens:     maxTokens,
		lastTimestamp: time.Now(),
		currentTokens: maxTokens, // 启动时令牌桶是满的
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
			// Shutdown 被调用，停止接收新任务，但要处理完 taskChan 中已有的任务
			// 关闭 taskChan 是由 Shutdown 方法完成的
			// 我们在这里排空 taskChan
			for job := range w.taskChan {
				// 在排空时，我们仍然要遵守速率限制
				w.waitAndTakeToken()
				w.rateChan <- job
			}
			return // 所有排队任务处理完毕，退出 dispatcher
		case job, ok := <-w.taskChan:
			if !ok {
				// 当 taskChan 被关闭且为空时，ok 会是 false
				// 这种主要发生在 Shutdown 场景下，是正常退出路径
				return
			}
			// 正常接收到任务，等待令牌
			w.waitAndTakeToken()

			// 将任务发送给 worker，同时也要能响应 shutdown 信号
			select {
			case w.rateChan <- job:
			case <-w.ctx.Done():
				// 如果在等待发送给 worker 时收到了关闭信号
				// 我们需要处理这个"孤儿"任务
				// 这里选择仍然尝试遵守速率限制并发送它
				w.waitAndTakeToken()
				w.rateChan <- job
			}
		}
	}
}

// worker 从 rateChan 中消费任务并执行
func (w *WorkerPool) worker() {
	defer w.wg.Done()
	// 使用 for-range 会自动处理 channel 关闭的情况
	// 当 rateChan 被关闭且为空后，循环会自动结束
	for task := range w.rateChan {
		task()
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
	// 1. 发出关闭信号，让所有 goroutine 进入关闭流程
	w.cancel()

	// 2. 关闭 taskChan，此后 Submit 方法会阻塞或 panic(如果池已关闭)
	// 这个操作是安全的，因为只有 Submit 方法会向 taskChan 写
	close(w.taskChan)

	// 3. 等待所有 goroutine (dispatcher 和 workers) 优雅退出
	w.wg.Wait()
}

// waitAndTakeToken 检查并获取一个令牌，如果令牌不足则阻塞等待
func (w *WorkerPool) waitAndTakeToken() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 使用循环，以应对等待后可能的状态变化（虽然在本设计中不太可能，但更健壮）
	for {
		// 计算并补充令牌
		now := time.Now()
		tokensGenerated := now.Sub(w.lastTimestamp).Seconds() * float64(w.ratePerSecond)
		w.currentTokens += tokensGenerated
		if w.currentTokens > w.maxTokens {
			w.currentTokens = w.maxTokens
		}
		w.lastTimestamp = now

		// 如果令牌足够，取走一个并返回
		if w.currentTokens >= 1 {
			w.currentTokens--
			return
		}

		// 如果令牌仍然不足，计算需要等待多久
		// 计算需要等待的时间以获得一个完整的令牌
		timeToWait := time.Duration((1-w.currentTokens)/float64(w.ratePerSecond)) * time.Second
		// 在等待前临时解锁，以允许其他 goroutine (如果有的话) 操作
		w.mu.Unlock()
		time.Sleep(timeToWait)
		// 等待后重新加锁，进入下一次循环检查
		w.mu.Lock()
	}
}
