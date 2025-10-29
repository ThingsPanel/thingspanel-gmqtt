package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("========================================")
	log.Println("纯 TCP 连接测试 - 验证服务器端 TCP Keep-Alive")
	log.Println("========================================")

	// 连接到 MQTT 服务器
	addr := "127.0.0.1:1883"
	log.Printf("连接到: %s\n", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("连接失败: %v\n", err)
	}
	defer conn.Close()

	log.Println("[✓] TCP 连接已建立")
	log.Printf("本地地址: %s\n", conn.LocalAddr())
	log.Printf("远程地址: %s\n", conn.RemoteAddr())

	// 检查连接类型
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		log.Println("========================================")
		log.Println("这是一个 TCP 连接")
		log.Println("注意: 客户端这边我们不禁用 TCP Keep-Alive")
		log.Println("这样可以观察:")
		log.Println("  - 客户端 → 服务器: 会有 TCP Keep-Alive (约15秒)")
		log.Println("  - 服务器 → 客户端: 应该没有 TCP Keep-Alive")
		log.Println("========================================")
		_ = tcpConn
	}

	log.Println("")
	log.Println("现在请在另一个终端运行 tshark 观察:")
	log.Println("  tshark -i 8 -f \"tcp port 1883\" -t ad")
	log.Println("")
	log.Println("预期观察到:")
	log.Println("  ✓ 客户端 → 服务器: 有 [TCP Keep-Alive] 包")
	log.Println("  ✓ 服务器 → 客户端: 没有 [TCP Keep-Alive] 包")
	log.Println("")
	log.Println("按 Ctrl+C 停止测试")
	log.Println("========================================")

	// 每 10 秒打印一次运行时间
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			elapsed := time.Since(startTime)
			log.Printf("[运行时间] %v - 连接仍然活跃\n", elapsed.Round(time.Second))
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("")
	log.Println("[退出] 关闭连接...")
	elapsed := time.Since(startTime)
	log.Printf("[测试结束] 总运行时间: %v\n", elapsed.Round(time.Second))
}
