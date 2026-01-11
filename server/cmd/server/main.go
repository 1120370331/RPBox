package main

import (
	"log"

	"github.com/rpbox/server/internal/api"
	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/pkg/auth"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	log.Println("Database connected")

	// 初始化JWT
	auth.Init(cfg.JWT.Secret)

	// 启动服务器
	server := api.NewServer(cfg)
	log.Printf("Server starting on :%s", cfg.Server.Port)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
