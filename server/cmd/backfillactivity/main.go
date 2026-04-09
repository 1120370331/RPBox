package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
)

func main() {
	var (
		username string
		before   string
		apply    bool
	)

	flag.StringVar(&username, "username", "", "仅回补指定用户名")
	flag.StringVar(&before, "before", "", "仅统计该时间之前的历史行为（支持 2006-01-02 或 RFC3339）")
	flag.BoolVar(&apply, "apply", false, "实际写入数据库；默认仅 dry-run")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := database.Init(&cfg.Database); err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	opts := service.ActivityBackfillOptions{
		DryRun: !apply,
	}

	if before != "" {
		parsed, err := parseBeforeTime(before)
		if err != nil {
			fmt.Printf("解析 before 失败: %v\n", err)
			os.Exit(1)
		}
		opts.Before = &parsed
	}

	if username != "" {
		var user model.User
		if err := database.DB.Select("id, username").Where("username = ?", username).First(&user).Error; err != nil {
			fmt.Printf("用户 '%s' 不存在\n", username)
			os.Exit(1)
		}
		opts.UserID = user.ID
	}

	summary, err := service.BackfillLegacyCommunityActivity(database.DB, opts)
	if err != nil {
		fmt.Printf("回补失败: %v\n", err)
		os.Exit(1)
	}

	mode := "DRY-RUN"
	if apply {
		mode = "APPLY"
	}

	fmt.Printf("[%s] 社区经验回补完成\n", mode)
	if opts.Before != nil {
		fmt.Printf("截止时间: %s\n", opts.Before.Format(time.RFC3339))
	}
	if username != "" {
		fmt.Printf("目标用户: %s\n", username)
	}
	fmt.Printf("覆盖用户数: %d\n", summary.SelectedUsers)
	fmt.Printf("受影响用户数: %d\n", summary.AffectedUsers)
	fmt.Printf("补发奖励条数: %d\n", summary.AppliedRewards)
	fmt.Printf("补发积分: %d\n", summary.TotalPoints)
	fmt.Printf("补发经验: %d\n", summary.TotalExperience)

	if len(summary.Users) == 0 {
		fmt.Println("无可补发数据。")
		return
	}

	fmt.Println()
	fmt.Println("用户明细:")
	for _, user := range summary.Users {
		name := user.Username
		if name == "" {
			name = fmt.Sprintf("user-%d", user.UserID)
		}
		fmt.Printf("- %s (ID:%d): 奖励 %d 条, 积分 %+d, 经验 %+d\n",
			name, user.UserID, user.RewardCount, user.PointsDelta, user.ExperienceDelta)
	}

	if !apply {
		fmt.Println()
		fmt.Println("未写入数据库。确认结果后，追加 -apply 执行正式回补。")
	}
}

func parseBeforeTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		var (
			t   time.Time
			err error
		)
		if layout == "2006-01-02" || layout == "2006-01-02 15:04:05" || layout == "2006-01-02 15:04" {
			t, err = time.ParseInLocation(layout, raw, time.Local)
		} else {
			t, err = time.Parse(layout, raw)
		}
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("不支持的时间格式: %s", raw)
}
