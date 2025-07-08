#!/bin/bash

# 内存泄漏演示和排查脚本
# 这个脚本演示如何使用pprof来检测和分析内存泄漏问题

echo "🔍 内存泄漏检测和修复演示"
echo "================================="

# 检查是否安装了必要的工具
if ! command -v go &> /dev/null; then
    echo "❌ 错误：请先安装 Go 语言环境"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    echo "❌ 错误：请先安装 curl"
    exit 1
fi

echo "✅ 准备开始演示..."

# 1. 启动有内存泄漏的服务器
echo ""
echo "📊 第1步：启动有内存泄漏的服务器"
echo "--------------------------------"
go run main.go &
LEAK_PID=$!
echo "✅ 泄漏服务器已启动 (PID: $LEAK_PID, 端口: 8080)"

# 等待服务器启动
sleep 3

# 2. 模拟负载，触发内存泄漏
echo ""
echo "🔥 第2步：模拟负载触发内存泄漏"
echo "--------------------------------"
for i in {1..10}; do
    curl -s "http://localhost:8080/leak?id=test_$i" > /dev/null
    echo "✅ 触发泄漏 $i/10"
    sleep 1
done

# 3. 收集内存分析数据
echo ""
echo "🔍 第3步：使用pprof收集内存分析数据"
echo "--------------------------------"
echo "正在收集堆内存分析数据..."
go tool pprof -alloc_space -cum -svg http://localhost:8080/debug/pprof/heap > heap_leak.svg 2>/dev/null
echo "✅ 堆内存分析数据已保存到 heap_leak.svg"

echo "正在收集goroutine分析数据..."
go tool pprof -svg http://localhost:8080/debug/pprof/goroutine > goroutine_leak.svg 2>/dev/null
echo "✅ Goroutine分析数据已保存到 goroutine_leak.svg"

# 4. 获取实时内存统计
echo ""
echo "📈 第4步：获取实时内存统计"
echo "--------------------------------"
echo "当前缓存状态："
curl -s "http://localhost:8080/status"
echo ""

echo "内存分配统计："
curl -s "http://localhost:8080/debug/pprof/heap?debug=1" | head -20

# 5. 停止泄漏服务器
echo ""
echo "🛑 第5步：停止泄漏服务器"
echo "--------------------------------"
kill $LEAK_PID
echo "✅ 泄漏服务器已停止"

# 6. 启动修复版本的服务器
echo ""
echo "✅ 第6步：启动修复版本的服务器"
echo "--------------------------------"
go run main_fixed.go &
FIXED_PID=$!
echo "✅ 修复版服务器已启动 (PID: $FIXED_PID, 端口: 8081)"

# 等待服务器启动
sleep 3

# 7. 对比测试修复版本
echo ""
echo "🔍 第7步：对比测试修复版本"
echo "--------------------------------"
for i in {1..10}; do
    curl -s "http://localhost:8081/leak?id=test_$i" > /dev/null
    echo "✅ 测试修复版本 $i/10"
    sleep 1
done

echo "修复版本内存分析："
go tool pprof -alloc_space -cum -svg http://localhost:8081/debug/pprof/heap > heap_fixed.svg 2>/dev/null
echo "✅ 修复版本内存分析数据已保存到 heap_fixed.svg"

echo "修复版本缓存状态："
curl -s "http://localhost:8081/status"
echo ""

# 8. 停止修复版本服务器
echo ""
echo "🛑 第8步：停止修复版本服务器"
echo "--------------------------------"
curl -s "http://localhost:8081/stop" > /dev/null
kill $FIXED_PID
echo "✅ 修复版本服务器已停止"

# 9. 总结
echo ""
echo "📊 第9步：分析总结"
echo "--------------------------------"
echo "✅ 演示完成！"
echo ""
echo "📁 生成的分析文件："
echo "   - heap_leak.svg: 泄漏版本的堆内存分析图"
echo "   - heap_fixed.svg: 修复版本的堆内存分析图"
echo "   - goroutine_leak.svg: 泄漏版本的goroutine分析图"
echo ""
echo "🔍 pprof 常用命令："
echo "   - go tool pprof http://localhost:8080/debug/pprof/heap"
echo "   - go tool pprof http://localhost:8080/debug/pprof/goroutine"
echo "   - go tool pprof http://localhost:8080/debug/pprof/profile"
echo ""
echo "💡 内存泄漏检测要点："
echo "   1. 观察堆内存持续增长"
echo "   2. 检查goroutine数量异常增加"
echo "   3. 分析内存分配热点"
echo "   4. 检查资源未正确释放"
echo "   5. 使用context正确管理goroutine生命周期" 