#!/bin/bash

# 简化的内存泄漏演示测试脚本
echo "🔍 内存泄漏演示 - 快速测试"
echo "================================"

# 1. 启动泄漏版本服务器
echo "启动泄漏版本服务器..."
go run main.go &
LEAK_PID=$!
echo "服务器已启动 (PID: $LEAK_PID)"

# 等待启动
sleep 2

# 2. 测试基本功能
echo "测试基本功能..."
curl -s "http://localhost:8080/leak?id=test1" && echo " - 触发泄漏功能正常"
curl -s "http://localhost:8080/status" && echo " - 状态查询功能正常"

# 3. 测试pprof端点
echo "测试pprof端点..."
curl -s "http://localhost:8080/debug/pprof/" | head -5 && echo " - pprof端点正常"

# 4. 清理
echo "清理进程..."
kill $LEAK_PID
echo "✅ 测试完成" 