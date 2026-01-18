package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	apiURL = "http://localhost:8081/api/v1/test/send-notification"
	userID = 2 // test 用户的 ID
)

type NotificationRequest struct {
	UserID  uint   `json:"user_id"`
	Content string `json:"content"`
}

func main() {
	log.Println("开始每秒发送通知，按 Ctrl+C 停止...")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 计数器
	count := 0

	// 创建定时器
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// HTTP 客户端
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for {
		select {
		case <-ticker.C:
			count++

			// 构建请求
			req := NotificationRequest{
				UserID:  userID,
				Content: fmt.Sprintf("测试通知 #%d - %s", count, time.Now().Format("15:04:05")),
			}

			jsonData, err := json.Marshal(req)
			if err != nil {
				log.Printf("❌ 序列化请求失败: %v", err)
				continue
			}

			// 发送 HTTP 请求
			resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("❌ 发送请求失败 #%d: %v", count, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				log.Printf("✓ 已发送通知 #%d", count)
			} else {
				log.Printf("❌ 发送失败 #%d: HTTP %d", count, resp.StatusCode)
			}
			resp.Body.Close()

		case <-sigChan:
			log.Printf("\n收到停止信号，共发送 %d 条通知", count)
			return
		}
	}
}
