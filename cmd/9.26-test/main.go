package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	serverAddr := "127.0.0.1:502" // modbus服务地址
	testIP := "127.0.0.1"
	
	fmt.Printf("开始测试认证限流功能...\n")
	fmt.Printf("目标服务器: %s\n", serverAddr)
	fmt.Printf("测试IP: %s\n", testIP)
	fmt.Println("配置: 失败3次封禁3分钟")
	fmt.Println()

	// 测试正常连接（应该被拒绝，因为没有有效的注册包）
	fmt.Println("=== 测试1: 发送无效注册包，触发限流 ===")
	for i := 1; i <= 5; i++ {
		fmt.Printf("第%d次尝试连接...\n", i)
		
		conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
		if err != nil {
			fmt.Printf("  连接失败: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// 发送无效的注册包
		invalidRegPkg := fmt.Sprintf("invalid_reg_pkg_%d", i)
		_, err = conn.Write([]byte(invalidRegPkg))
		if err != nil {
			fmt.Printf("  发送数据失败: %v\n", err)
		} else {
			fmt.Printf("  已发送无效注册包: %s\n", invalidRegPkg)
		}

		// 等待服务器响应或关闭连接
		time.Sleep(2 * time.Second)
		conn.Close()
		fmt.Printf("  连接已关闭\n")
		
		// 短暂间隔
		time.Sleep(1 * time.Second)
		fmt.Println()
	}

	fmt.Println("=== 测试2: 验证限流是否生效 ===")
	fmt.Println("等待5秒后尝试连接，应该被立即拒绝...")
	time.Sleep(5 * time.Second)

	for i := 1; i <= 3; i++ {
		fmt.Printf("限流期内第%d次尝试连接...\n", i)
		
		start := time.Now()
		conn, err := net.DialTimeout("tcp", serverAddr, 2*time.Second)
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("  连接失败: %v (耗时: %v)\n", err, duration)
		} else {
			fmt.Printf("  连接成功，但应该被服务器立即关闭 (耗时: %v)\n", duration)
			
			// 尝试发送数据
			_, err = conn.Write([]byte("test"))
			if err != nil {
				fmt.Printf("  发送失败: %v\n", err)
			}
			
			// 等待一下看是否被关闭
			time.Sleep(1 * time.Second)
			conn.Close()
		}
		
		time.Sleep(2 * time.Second)
		fmt.Println()
	}

	fmt.Println("=== 测试3: 等待限流解除 ===")
	fmt.Printf("等待3分钟让限流解除...\n")
	
	// 倒计时显示
	for i := 180; i > 0; i -= 10 {
		fmt.Printf("剩余等待时间: %d秒\n", i)
		time.Sleep(10 * time.Second)
	}

	fmt.Println("=== 测试4: 验证限流解除后可以正常连接 ===")
	for i := 1; i <= 2; i++ {
		fmt.Printf("限流解除后第%d次尝试连接...\n", i)
		
		conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
		if err != nil {
			fmt.Printf("  连接失败: %v\n", err)
		} else {
			fmt.Printf("  连接成功！限流已解除\n")
			
			// 发送无效注册包
			invalidRegPkg := fmt.Sprintf("test_after_unblock_%d", i)
			_, err = conn.Write([]byte(invalidRegPkg))
			if err != nil {
				fmt.Printf("  发送数据失败: %v\n", err)
			} else {
				fmt.Printf("  已发送注册包: %s\n", invalidRegPkg)
			}
			
			time.Sleep(2 * time.Second)
			conn.Close()
			fmt.Printf("  连接已关闭\n")
		}
		
		time.Sleep(2 * time.Second)
		fmt.Println()
	}

	fmt.Println("测试完成！")
	fmt.Println()
	fmt.Println("期望结果:")
	fmt.Println("1. 前3次无效连接应该成功建立但被服务器关闭（认证失败）")
	fmt.Println("2. 第4、5次连接应该被立即拒绝或快速关闭（触发限流）")
	fmt.Println("3. 等待3分钟后的连接应该可以正常建立（限流解除）")
	fmt.Println()
	fmt.Println("请检查服务器日志确认限流功能是否正常工作")
}