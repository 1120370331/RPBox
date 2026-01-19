package main

import (
	"fmt"
	"log"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("开始清理重复的预设标签...")

	// 查询所有预设标签
	var allTags []model.Tag
	if err := database.DB.Where("type = ?", "preset").Order("id ASC").Find(&allTags).Error; err != nil {
		log.Fatalf("查询标签失败: %v", err)
	}

	fmt.Printf("找到 %d 个预设标签\n", len(allTags))

	// 使用 map 记录已见过的标签（key: name+category）
	seen := make(map[string]uint)
	duplicates := []uint{}

	for _, tag := range allTags {
		key := tag.Name + "|" + tag.Category
		if existingID, exists := seen[key]; exists {
			// 发现重复，保留ID较小的（较早创建的）
			duplicates = append(duplicates, tag.ID)
			fmt.Printf("发现重复标签: %s (category: %s), ID: %d (保留ID: %d)\n",
				tag.Name, tag.Category, tag.ID, existingID)
		} else {
			seen[key] = tag.ID
		}
	}

	if len(duplicates) == 0 {
		fmt.Println("没有发现重复的标签")
		return
	}

	fmt.Printf("\n共发现 %d 个重复标签，准备删除...\n", len(duplicates))

	// 删除重复的标签
	if err := database.DB.Where("id IN ?", duplicates).Delete(&model.Tag{}).Error; err != nil {
		log.Fatalf("删除重复标签失败: %v", err)
	}

	fmt.Printf("成功删除 %d 个重复标签\n", len(duplicates))
}
