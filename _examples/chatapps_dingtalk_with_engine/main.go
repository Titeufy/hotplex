package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hrygo/hotplex"
	"github.com/hrygo/hotplex/chatapps"
	"github.com/hrygo/hotplex/chatapps/dingtalk"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	addr := os.Getenv("HOTPLEX_CHATAPPS_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	workDir := os.Getenv("HOTPLEX_WORK_DIR")
	if workDir == "" {
		workDir = "/tmp/hotplex-chatapps"
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		logger.Error("Failed to create work directory", "error", err, "dir", workDir)
		os.Exit(1)
	}

	engineOpts := hotplex.EngineOptions{
		Timeout:     10 * time.Minute,
		IdleTimeout: 5 * time.Minute,
		Logger:      logger,
	}

	engine, err := hotplex.NewEngine(engineOpts)
	if err != nil {
		logger.Error("Failed to create engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close()

	logger.Info("HotPlex Engine initialized")

	adapter := dingtalk.NewAdapter(dingtalk.Config{
		ServerAddr:    addr,
		MaxMessageLen: 5000,
	}, logger)

	adapter.SetSender(func(ctx context.Context, sessionID string, msg *chatapps.ChatMessage) error {
		robotCode, _ := msg.Metadata["robot_code"].(string)
		if robotCode == "" {
			robotCode = "test"
		}

		chunks := adapter.ChunkMessage(msg.Content)
		for i, chunk := range chunks {
			content := chunk
			if len(chunks) > 1 {
				content = fmt.Sprintf("[%d/%d]\n%s", i+1, len(chunks), chunk)
			}

			payload := map[string]any{
				"msgtype": "text",
				"text":    map[string]string{"content": content},
			}

			token, err := adapter.GetAccessToken()
			if err != nil {
				return err
			}

			body, _ := json.Marshal(payload)
			url := fmt.Sprintf("https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend?robotCode=%s", robotCode)
			req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-acs-dingtalk-access-token", token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if i < len(chunks)-1 {
				time.Sleep(200 * time.Millisecond)
			}
		}
		return nil
	})

	adapter.SetHandler(func(ctx context.Context, msg *chatapps.ChatMessage) error {
		logger.Info("Received message", "user", msg.UserID, "content", msg.Content[:min(50, len(msg.Content))])

		userWorkDir := filepath.Join(workDir, msg.UserID)
		if err := os.MkdirAll(userWorkDir, 0755); err != nil {
			logger.Error("Failed to create user work dir", "error", err, "user", msg.UserID)
			return err
		}

		thinkingMsg := &chatapps.ChatMessage{
			Platform:  "dingtalk",
			SessionID: msg.SessionID,
			Content:   "🤔 正在思考...",
			Metadata:  msg.Metadata,
		}
		if err := adapter.SendMessage(ctx, msg.SessionID, thinkingMsg); err != nil {
			logger.Error("Failed to send thinking message", "error", err)
		}

		cfg := &hotplex.Config{
			WorkDir:          userWorkDir,
			SessionID:        msg.SessionID,
			TaskInstructions: "You are a helpful AI assistant. Respond in Chinese.",
		}

		var responseContent string
		err := engine.Execute(ctx, cfg, msg.Content, func(eventType string, data any) error {
			switch eventType {
			case "thinking":
			case "tool_use":
			case "answer":
				responseContent += fmt.Sprintf("%v", data)
			case "error":
				return fmt.Errorf("%v", data)
			}
			return nil
		})

		if err != nil {
			logger.Error("Engine execution failed", "error", err)
			responseContent = fmt.Sprintf("❌ 处理失败: %v", err)
		}

		responseMsg := &chatapps.ChatMessage{
			Platform:  "dingtalk",
			SessionID: msg.SessionID,
			Content:   responseContent,
			RichContent: &chatapps.RichContent{
				ParseMode: chatapps.ParseModeMarkdown,
			},
			Metadata: msg.Metadata,
		}

		if err := adapter.SendMessage(ctx, msg.SessionID, responseMsg); err != nil {
			logger.Error("Failed to send response", "error", err)
		} else {
			logger.Info("Response sent", "user", msg.UserID)
		}

		return nil
	})

	if err := adapter.Start(context.Background()); err != nil {
		logger.Error("Failed to start adapter", "error", err)
		os.Exit(1)
	}

	fmt.Println("🎉 ChatApps + HotPlex Engine 已启动!")
	fmt.Printf("   监听地址: http://localhost%s\n", addr)
	fmt.Println("   回调端点: /webhook")
	fmt.Println("   健康检查: /health")
	fmt.Printf("   工作目录: %s\n", workDir)
	fmt.Println("\n按 Ctrl+C 退出")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n👋 正在关闭...")
	adapter.Stop()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
