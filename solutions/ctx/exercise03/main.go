package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

/*
题目描述：
在一个 Web 服务中，我们希望为每个请求生成一个唯一的 request_id，并在整个请求处理链路中传递它，以便于日志追踪。
请实现一个 HTTP 中间件 withRequestID，它会执行以下操作：
从传入的 HTTP 请求中（或新生成一个）获取一个 request_id。
使用 context.WithValue 将 request_id 存入请求的 context 中。
调用下一个 http.Handler。
然后，编写一个最终的 http.Handler 函数 finalHandler，它能从 context 中读取 request_id 并将其打印到 HTTP 响应中。
考察点：
理解 context.WithValue 的作用和适用场景（传递请求范围的数据）。
如何在 http.Handler 和中间件中操作 http.Request.Context()。
从 context 中安全地读取值并进行类型断言。
*/

type requestIDKey int

const key requestIDKey = 0

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), key, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func finalHandler(w http.ResponseWriter, r *http.Request) {
	requestID, ok := r.Context().Value(key).(string)
	if !ok {
		log.Println("requestID not found")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Request-ID", requestID)
	fmt.Fprintf(w, "http request done, traceID: %s\n", requestID)
}

func main() {
	final := http.HandlerFunc(finalHandler)
	handler := withRequestID(final)

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
