# Go 内存泄漏检测和修复演示

这个项目演示了常见的 Go 内存泄漏问题，以及如何使用 pprof 工具进行检测和分析。

## 项目结构

```
memory_leak_demo/
├── main.go        # 存在内存泄漏的示例程序
├── main_fixed.go  # 修复后的程序
├── demo.sh        # 自动化演示脚本
├── go.mod         # Go 模块文件
└── README.md      # 项目说明文档
```

## 内存泄漏类型

### 1. Goroutine 泄漏
**问题**：创建的 goroutine 没有正确的退出机制，导致持续消耗内存。

```go
// 问题代码
func processDataWithLeak(id string) {
    go func() {
        for {
            // 没有退出条件，goroutine 永远运行
            time.Sleep(100 * time.Millisecond)
            // 持续分配内存
        }
    }()
}
```

**修复方法**：使用 context 控制 goroutine 生命周期。

```go
// 修复代码
func processDataFixed(ctx context.Context, id string) {
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return  // 正确退出
            case <-ticker.C:
                // 处理逻辑
            }
        }
    }()
}
```

### 2. HTTP 连接泄漏
**问题**：HTTP 响应体未正确关闭，导致连接泄漏。

```go
// 问题代码
resp, err := client.Get("https://example.com")
if err != nil {
    return
}
// 没有关闭 response body
```

**修复方法**：确保关闭响应体。

```go
// 修复代码
resp, err := client.Get("https://example.com")
if err != nil {
    return
}
defer resp.Body.Close()  // 正确关闭
```

### 3. Slice 引用泄漏
**问题**：从大 slice 中切片时，保持了对原始大 slice 的引用。

```go
// 问题代码
largeSlice := make([]byte, 10*1024*1024)  // 10MB
smallSlice := largeSlice[:10]  // 只需要10字节，但引用了整个10MB
```

**修复方法**：复制需要的部分。

```go
// 修复代码
largeSlice := make([]byte, 10*1024*1024)
smallSlice := make([]byte, 10)
copy(smallSlice, largeSlice[:10])  // 复制后原始slice可以被GC
largeSlice = nil
```

## 使用方法

### 1. 快速演示

```bash
# 给脚本添加执行权限
chmod +x demo.sh

# 运行自动化演示
./demo.sh
```

### 2. 手动测试

#### 启动泄漏版本服务器
```bash
go run main.go
```

#### 启动修复版本服务器
```bash
go run main_fixed.go
```

#### 触发内存泄漏
```bash
# 触发泄漏
curl "http://localhost:8080/leak?id=test1"
curl "http://localhost:8080/leak?id=test2"

# 查看状态
curl "http://localhost:8080/status"
```

## pprof 分析技巧

### 1. 内存分析

#### 查看实时堆内存
```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

#### 生成内存分析图
```bash
go tool pprof -alloc_space -cum -svg http://localhost:8080/debug/pprof/heap > heap.svg
```

#### 查看文本格式内存统计
```bash
curl "http://localhost:8080/debug/pprof/heap?debug=1"
```

### 2. Goroutine 分析

#### 查看 goroutine 信息
```bash
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

#### 生成 goroutine 分析图
```bash
go tool pprof -svg http://localhost:8080/debug/pprof/goroutine > goroutine.svg
```

### 3. pprof 交互式命令

进入 pprof 交互模式后，可以使用以下命令：

```bash
# 查看内存分配最多的函数
(pprof) top10

# 查看调用图
(pprof) web

# 查看特定函数的内存分配
(pprof) list functionName

# 查看累积内存分配
(pprof) top -cum

# 查看goroutine数量
(pprof) top
```

## 关键指标解读

### 内存分析指标

- **alloc_objects**: 分配的对象数量
- **alloc_space**: 分配的内存空间
- **inuse_objects**: 正在使用的对象数量
- **inuse_space**: 正在使用的内存空间

### 异常模式识别

1. **内存泄漏**：`inuse_space` 持续增长
2. **Goroutine 泄漏**：goroutine 数量异常增加
3. **频繁 GC**：`alloc_space` 增长很快但 `inuse_space` 相对稳定

## 最佳实践

### 1. 预防措施

```go
// 1. 使用 context 控制 goroutine
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(time.Second):
            // 工作逻辑
        }
    }
}

// 2. 正确关闭资源
func httpRequest() {
    resp, err := http.Get("https://example.com")
    if err != nil {
        return
    }
    defer resp.Body.Close()
    
    // 处理响应
}

// 3. 设置合理的超时
client := &http.Client{
    Timeout: 10 * time.Second,
}

// 4. 使用对象池减少GC压力
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}
```

### 2. 监控建议

```go
// 定期检查内存统计
func printMemStats() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    fmt.Printf("Alloc = %d KB", m.Alloc/1024)
    fmt.Printf("TotalAlloc = %d KB", m.TotalAlloc/1024)
    fmt.Printf("Sys = %d KB", m.Sys/1024)
    fmt.Printf("NumGC = %d", m.NumGC)
}

// 监控goroutine数量
func printGoroutineCount() {
    fmt.Printf("Goroutines: %d", runtime.NumGoroutine())
}
```

## 排查步骤

1. **建立基线**：记录正常情况下的内存使用和 goroutine 数量
2. **监控趋势**：观察内存和 goroutine 数量的变化趋势
3. **分析热点**：使用 pprof 找出内存分配热点
4. **代码审查**：检查资源管理和 goroutine 生命周期
5. **验证修复**：对比修复前后的性能指标

## 注意事项

- pprof 会对性能有轻微影响，生产环境使用时需要注意
- 内存分析最好在负载相对稳定时进行
- 定期进行内存分析，建立长期监控机制
- 结合业务逻辑分析，避免误判正常的内存增长

## 扩展阅读

- [Go pprof 官方文档](https://golang.org/pkg/net/http/pprof/)
- [Go 内存管理和垃圾回收](https://golang.org/doc/gc-guide)
- [Go 性能优化最佳实践](https://github.com/dgryski/go-perfbook) 