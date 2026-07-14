package main

import (
	"log"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	if err := database.SeedRPDBDemo(database.DB); err != nil {
		log.Fatalf("写入 RPDB 演示数据失败: %v", err)
	}
	log.Printf("RPDB 演示数据写入完成，账号=%s", "rpdb_demo")
}
