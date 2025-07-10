package main

import (
	"context"
	"fmt"
	"time"
)

/*
题目描述：
模拟一个向外部服务请求数据的场景。创建一个函数 fetchData(ctx context.Context)，该函数内部会模拟一个耗时操作（例如：time.Sleep(3 * time.Second)）来代表网络请求。
main 函数需要调用 fetchData，但要求该请求必须在 2 秒内完成。如果 fetchData 在 2 秒内成功返回，则打印 "data fetched successfully"；
如果超时，则打印 "fetch data timeout"。
考察点：
使用 context.WithTimeout 来创建一个有超时时间的 context。
在 fetchData 函数中，使用 select 结构同时等待耗时操作完成和 ctx.Done() 事件。
正确处理超时后 ctx.Err() 返回的错误类型 (context.DeadlineExceeded)。
*/

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := fetchData(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("fetch data timeout")
		} else {
			fmt.Println("fetch data failed")
		}
		return
	}

	fmt.Println("fetch data successfully")
}

func fetchData(ctx context.Context) error {

	longRunningTask := func() <-chan bool {
		done := make(chan bool)
		go func() {
			time.Sleep(3 * time.Second)
			done <- true
		}()
		return done
	}

	select {
	case <-longRunningTask():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
