package main

import (
	"fmt"
	"os"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: setadmin <用户名>")
		fmt.Println("示例: setadmin admin")
		os.Exit(1)
	}

	username := os.Args[1]

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 连接数据库
	if err := database.Init(&cfg.Database); err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 查找用户
	var user model.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		fmt.Printf("用户 '%s' 不存在\n", username)
		os.Exit(1)
	}

	// 设置为管理员
	database.DB.Model(&user).Update("role", "admin")
	fmt.Printf("已将用户 '%s' (ID:%d) 设为超级管理员\n", user.Username, user.ID)
}
