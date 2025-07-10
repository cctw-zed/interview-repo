package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestWorkerPool_BasicExecutionAndShutdown tests if the pool processes all submitted jobs
// and shuts down gracefully, ensuring all work is done.
func TestWorkerPool_BasicExecutionAndShutdown(t *testing.T) {
	t.Run("should execute all submitted jobs", func(t *testing.T) {
		// 1. 设置 (Setup)
		const numJobs = 100
		const workerCount = 10
		const rateLimit = 500 // 设置一个较高的速率限制，确保不影响本测试

		// 使用原子计数器来安全地统计已完成的任务数
		var jobsDoneCounter int64

		pool := NewWorkerPool(context.Background(), workerCount, rateLimit)

		// 2. 执行 (Act)
		// 提交 numJobs 个任务，每个任务完成时都使计数器加一
		for i := 0; i < numJobs; i++ {
			pool.Submit(func() {
				atomic.AddInt64(&jobsDoneCounter, 1)
			})
		}

		// 调用 Shutdown，它应该会阻塞，直到所有任务都完成
		pool.Shutdown()

		// 3. 断言 (Assert)
		// 检查计数器的值是否等于我们提交的任务总数
		if atomic.LoadInt64(&jobsDoneCounter) != numJobs {
			t.Errorf("expected %d jobs to be done, but got %d", numJobs, atomic.LoadInt64(&jobsDoneCounter))
		}
	})
}

// TestWorkerPool_RateLimiting tests if the rate limiting mechanism is working correctly.
func TestWorkerPool_RateLimiting(t *testing.T) {
	t.Run("should respect the rate limit", func(t *testing.T) {
		// 1. 设置
		const numJobs = 50
		const workerCount = 5
		const rateLimit = 10 // 每秒 10 个任务

		// 使用原子计数器
		var jobsDoneCounter int64
		pool := NewWorkerPool(context.Background(), workerCount, rateLimit)

		// 2. 执行
		startTime := time.Now()

		for i := 0; i < numJobs; i++ {
			pool.Submit(func() {
				atomic.AddInt64(&jobsDoneCounter, 1)
			})
		}

		pool.Shutdown()
		elapsedTime := time.Since(startTime)

		// 3. 断言
		if atomic.LoadInt64(&jobsDoneCounter) != numJobs {
			t.Errorf("expected %d jobs to be done, but got %d", numJobs, atomic.LoadInt64(&jobsDoneCounter))
		}

		// 核心断言：验证执行时间
		// 20个任务，速率是10个/秒。理论上至少需要 (20-1)/10 = 1.9 秒。
		// 我们设置一个略小于理论值的下限，来容忍调度误差。
		// 如果执行时间远小于这个值，说明速率限制没有生效。
		minExpectedDuration := ((numJobs - 1) / rateLimit) * time.Second
		if elapsedTime < minExpectedDuration {
			t.Errorf("rate limiting failed: expected to take at least %v, but took %v", minExpectedDuration, elapsedTime)
		}

		t.Logf("Processed %d jobs in %v, respecting rate limit of %d/s.", numJobs, elapsedTime, rateLimit)
	})
}

// TestWorkerPool_ContextCancellation tests if the pool stops processing
// when the parent context is canceled.
func TestWorkerPool_ContextCancellation(t *testing.T) {
	t.Run("should stop immediately and discard queued jobs", func(t *testing.T) {
		// 1. 设置
		const workerCount = 5
		const rateLimit = 10 // 较慢的速率限制

		var jobsStartedCounter int64
		var jobsDoneCounter int64

		// 创建一个可以手动取消的 context
		ctx, cancel := context.WithCancel(context.Background())
		pool := NewWorkerPool(ctx, workerCount, rateLimit)

		// 2. 执行
		// 提交大量任务，远超 worker 数量和速率限制
		// 每个任务启动时增加 started 计数器，完成后增加 done 计数器
		for i := 0; i < 100; i++ {
			pool.Submit(func() {
				atomic.AddInt64(&jobsStartedCounter, 1)
				time.Sleep(50 * time.Millisecond) // 模拟任务耗时
				atomic.AddInt64(&jobsDoneCounter, 1)
			})
		}

		// 等待一小段时间，确保有一些任务已经被 dispatcher 推送到 rateChan，
		// 并且 worker 已经开始执行其中一部分。
		time.Sleep(150 * time.Millisecond)

		// 此刻，由于速率限制 (10/s) 和 worker 数量 (5)，
		// 应该只有少数任务 (大概 1-2 个) 已经开始执行。
		// 队列 taskChan 和 rateChan 中应该还有很多任务。

		// 取消 context
		cancel()

		// 调用 Shutdown。因为 context 已被取消，所有 goroutine 应该会迅速退出，
		// Shutdown 会很快返回。
		pool.Shutdown()

		// 3. 断言
		startedCount := atomic.LoadInt64(&jobsStartedCounter)
		doneCount := atomic.LoadInt64(&jobsDoneCounter)

		// 断言完成的任务数等于开始的任务数，确保没有任务中途退出
		if startedCount != doneCount {
			t.Errorf("started jobs (%d) should equal done jobs (%d)", startedCount, doneCount)
		}

		// 关键断言：断言开始执行的任务数非常少，远小于提交的总数。
		// 这个数字的上限取决于上面的 Sleep 时间和速率。
		// 150ms / (1000ms/10) = 1.5 个令牌，所以最多启动 2 个。我们放宽到 5 个。
		if startedCount > 5 {
			t.Errorf("expected a small number of jobs to have started before cancellation, but got %d", startedCount)
		}

		t.Logf("Jobs submitted: 100. Jobs started/done after cancellation: %d", startedCount)
	})
}
