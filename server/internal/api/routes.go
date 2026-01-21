package api

import (
	"path/filepath"

	"github.com/rpbox/server/internal/middleware"
)

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	// 静态文件服务 - 更新包下载
	s.router.Static("/releases", "./releases")
	// 静态文件服务 - 图片上传
	s.router.Static("/uploads", filepath.Join(s.cfg.Storage.Path, "uploads"))

	v1 := s.router.Group("/api/v1")
	{
		// 公开接口
		v1.POST("/auth/send-code", s.sendVerificationCode)
		v1.POST("/auth/register", s.register)
		v1.POST("/auth/login", s.login)
		v1.POST("/auth/forgot-password", s.forgotPassword) // 发送重置密码验证码
		v1.POST("/auth/reset-password", s.resetPassword)   // 重置密码

		// 插件版本管理（公开）
		v1.GET("/addon/manifest", s.getAddonManifest)
		v1.GET("/addon/latest", s.getAddonLatest)
		v1.GET("/addon/download/:version", s.downloadAddon)

		// 公开剧情（无需登录）
		v1.GET("/public/stories/:code", s.getPublicStory)

		// 图标服务（公开）
		v1.GET("/icons/:name", s.getIcon)

		// 预设标签（公开）
		v1.GET("/tags/preset", s.getPresetTags)

		// 客户端更新检查（公开）
		v1.GET("/updater/:target/:arch/:current_version", s.checkUpdate)

		// 公开公会列表（社区广场）
		v1.GET("/public/guilds", s.listPublicGuilds)

		// 测试端点（仅用于开发）
		v1.POST("/test/send-notification", s.testSendNotification)

		// 画作图片公开访问
		v1.GET("/items/:id/images/:imageId", s.getItemImage)
		v1.GET("/items/:id/images/:imageId/download", s.downloadItemImage)

		// 通用图片服务（支持缩略图）
		v1.GET("/images/:type/:id", s.getImage)

		// 需要认证的接口
		auth := v1.Group("")
		auth.Use(middleware.JWTAuth())
		{
			auth.GET("/profiles", s.listProfiles)
			auth.POST("/profiles", s.createProfile)
			auth.GET("/profiles/:id", s.getProfile)
			auth.PUT("/profiles/:id", s.updateProfile)
			auth.DELETE("/profiles/:id", s.deleteProfile)
			auth.GET("/profiles/:id/versions", s.getProfileVersions)
			auth.POST("/profiles/:id/rollback", s.rollbackProfile)

			auth.GET("/stories", s.listStories)
			auth.POST("/stories", s.createStory)
			auth.GET("/stories/:id", s.getStory)
			auth.PUT("/stories/:id", s.updateStory)
			auth.DELETE("/stories/:id", s.deleteStory)
			auth.POST("/stories/:id/entries", s.addStoryEntries)
			auth.PUT("/stories/:id/entries/:entryId", s.updateStoryEntry)
			auth.DELETE("/stories/:id/entries/:entryId", s.deleteStoryEntry)
			auth.POST("/stories/:id/publish", s.publishStory)

			// 角色管理
			auth.GET("/characters", s.listCharacters)
			auth.POST("/characters", s.createOrUpdateCharacter)
			auth.GET("/characters/:id", s.getCharacter)
			auth.PUT("/characters/:id", s.updateCharacter)
			auth.DELETE("/characters/:id", s.deleteCharacter)

			// 道具市场
			auth.GET("/items", s.listItems)
			auth.POST("/items", s.createItem)
			auth.GET("/items/:id", s.getItem)
			auth.PUT("/items/:id", s.updateItem)
			auth.DELETE("/items/:id", s.deleteItem)
			auth.POST("/items/:id/download", s.downloadItem)
			auth.POST("/items/:id/rate", s.rateItem)
			auth.GET("/items/:id/comments", s.getItemComments)
			auth.POST("/items/:id/comments", s.addItemComment)
			auth.GET("/items/:id/tags", s.getItemTags)
			auth.POST("/items/:id/tags", s.addItemTag)
			auth.DELETE("/items/:id/tags/:tagId", s.removeItemTag)
			auth.GET("/items/favorites", s.listMyItemFavorites)
			auth.GET("/items/likes", s.listMyItemLikes)
			auth.GET("/items/views", s.listMyItemViews)
			auth.POST("/items/:id/like", s.likeItem)
			auth.DELETE("/items/:id/like", s.unlikeItem)
			auth.POST("/items/:id/favorite", s.favoriteItem)
			auth.DELETE("/items/:id/favorite", s.unfavoriteItem)

			// 画作图片管理
			auth.POST("/items/:id/images", s.uploadItemImages)
			auth.GET("/items/:id/images", s.listItemImages)
			auth.DELETE("/items/:id/images/:imageId", s.deleteItemImage)

			// 账号备份（以账号为单位）
			auth.GET("/account-backups", s.listAccountBackups)
			auth.GET("/account-backups/:account_id", s.getAccountBackup)
			auth.POST("/account-backups", s.upsertAccountBackup)
			auth.DELETE("/account-backups/:account_id", s.deleteAccountBackup)
			auth.GET("/account-backups/:account_id/versions", s.getAccountBackupVersions)

			// 标签管理
			auth.GET("/tags", s.listTags)
			auth.POST("/tags", s.createTag)
			auth.PUT("/tags/:id", s.updateTag)
			auth.DELETE("/tags/:id", s.deleteTag)

			// 剧情标签管理
			auth.GET("/stories/:id/tags", s.getStoryTags)
			auth.POST("/stories/:id/tags", s.addStoryTag)
			auth.DELETE("/stories/:id/tags/:tagId", s.removeStoryTag)

			// 剧情归档的公会
			auth.GET("/stories/:id/guilds", s.getStoryGuilds)

			// 公会管理
			auth.GET("/guilds", s.listGuilds)
			auth.POST("/guilds", s.createGuild)
			auth.GET("/guilds/:id", s.getGuild)
			auth.PUT("/guilds/:id", s.updateGuild)
			auth.DELETE("/guilds/:id", s.deleteGuild)
			auth.POST("/guilds/join", s.joinGuild)
			auth.POST("/guilds/:id/leave", s.leaveGuild)
			auth.GET("/guilds/:id/members", s.listGuildMembers)
			auth.PUT("/guilds/:id/members/:uid", s.updateMemberRole)
			auth.DELETE("/guilds/:id/members/:uid", s.removeMember)
			auth.POST("/guilds/:id/banner", s.uploadGuildBanner)

			// 公会申请系统
			auth.POST("/guilds/:id/apply", s.applyGuild)
			auth.GET("/guilds/:id/applications", s.listGuildApplications)
			auth.POST("/guilds/:id/applications/:appId/review", s.reviewGuildApplication)
			auth.DELETE("/guilds/:id/applications/:appId", s.cancelApplication)
			auth.GET("/user/guild-applications", s.listMyApplications)

			// 公会标签
			auth.GET("/guilds/:id/tags", s.listGuildTags)
			auth.POST("/guilds/:id/tags", s.createGuildTag)
			auth.DELETE("/guilds/:id/tags/:tagId", s.deleteGuildTag)

			// 公会剧情归档
			auth.GET("/guilds/:id/stories", s.listGuildStories)
			auth.POST("/guilds/:id/stories/:storyId", s.archiveStoryToGuild)
			auth.DELETE("/guilds/:id/stories/:storyId", s.removeStoryFromGuild)

			// 社区帖子
			auth.GET("/posts", s.listPosts)
			auth.POST("/posts", s.createPost)
			auth.GET("/posts/favorites", s.listMyFavorites)
			auth.GET("/posts/likes", s.listMyPostLikes)
			auth.GET("/posts/views", s.listMyPostViews)
			auth.GET("/posts/events", s.listEvents) // 活动日历
			auth.GET("/posts/:id", s.getPost)
			auth.PUT("/posts/:id", s.updatePost)
			auth.DELETE("/posts/:id", s.deletePost)
			auth.POST("/posts/:id/like", s.likePost)
			auth.DELETE("/posts/:id/like", s.unlikePost)
			auth.POST("/posts/:id/favorite", s.favoritePost)
			auth.DELETE("/posts/:id/favorite", s.unfavoritePost)
			auth.GET("/posts/:id/tags", s.getPostTags)
			auth.POST("/posts/:id/tags", s.addPostTag)
			auth.DELETE("/posts/:id/tags/:tagId", s.removePostTag)

			// 帖子评论
			auth.GET("/posts/:id/comments", s.listComments)
			auth.POST("/posts/:id/comments", s.createComment)
			auth.DELETE("/posts/:id/comments/:commentId", s.deleteComment)
			auth.POST("/comments/:commentId/like", s.likeComment)
			auth.DELETE("/comments/:commentId/like", s.unlikeComment)

			// 用户管理
			auth.GET("/user/info", s.getUserInfo)
			auth.PUT("/user/info", s.updateUserInfo)
			auth.POST("/user/avatar", s.updateAvatar)
			auth.POST("/user/bind-email", s.bindEmail)
			auth.GET("/users/:id", s.getUserProfile)
			auth.GET("/users/:id/guilds", s.getUserGuilds)

			// 通知中心
			auth.GET("/notifications", s.listNotifications)
			auth.PUT("/notifications/:id/read", s.markNotificationAsRead)
			auth.PUT("/notifications/read-all", s.markAllNotificationsAsRead)
			auth.GET("/notifications/unread-count", s.getUnreadCount)
			auth.DELETE("/notifications/:id", s.deleteNotification)
			auth.DELETE("/notifications/all", s.deleteAllNotifications)

			// WebSocket 实时通知
			auth.GET("/ws/notifications", s.handleWebSocket)

			// 通用图片上传
			auth.POST("/upload/image", s.uploadImage)

			// 版主中心（需要版主权限）
			mod := auth.Group("/moderator")
			mod.Use(middleware.ModeratorAuth())
			{
				// 统计数据
				mod.GET("/stats", s.getModeratorStats)

				// 审核中心 - 帖子
				mod.GET("/review/posts", s.listPendingPosts)
				mod.POST("/review/posts/:id", s.reviewPost)

				// 审核中心 - 帖子编辑
				mod.GET("/review/post-edits", s.listPendingEdits)
				mod.POST("/review/post-edits/:id", s.reviewPostEdit)

				// 审核中心 - 道具
				mod.GET("/review/items", s.listPendingItems)
				mod.POST("/review/items/:id", s.reviewItem)

				// 审核中心 - 道具编辑
				mod.GET("/review/item-edits", s.listPendingItemEdits)
				mod.POST("/review/item-edits/:id", s.reviewItemEdit)

				// 管理中心 - 帖子
				mod.GET("/manage/posts", s.listAllPosts)
				mod.DELETE("/manage/posts/:id", s.deletePostByMod)
				mod.POST("/manage/posts/:id/hide", s.hidePostByMod)
				mod.POST("/manage/posts/:id/pin", s.pinPost)
				mod.POST("/manage/posts/:id/feature", s.featurePost)

				// 管理中心 - 道具
				mod.GET("/manage/items", s.listAllItems)
				mod.DELETE("/manage/items/:id", s.deleteItemByMod)
				mod.POST("/manage/items/:id/hide", s.hideItemByMod)

				// 公会管理
				mod.GET("/review/guilds", s.listPendingGuilds)
				mod.POST("/review/guilds/:id", s.reviewGuild)
				mod.GET("/manage/guilds", s.listAllGuilds)
				mod.PUT("/manage/guilds/:id/owner", s.changeGuildOwner)
				mod.DELETE("/manage/guilds/:id", s.deleteGuildByMod)

				// 用户管理
				mod.GET("/users", s.listUsers)
				mod.POST("/users/:id/mute", s.muteUser)
				mod.DELETE("/users/:id/mute", s.unmuteUser)
				mod.POST("/users/:id/ban", s.banUser)
				mod.DELETE("/users/:id/ban", s.unbanUser)
				mod.POST("/users/:id/posts/disable", s.disableUserPosts)
				mod.DELETE("/users/:id/posts", s.deleteUserPosts)

				// 操作日志
				mod.GET("/action-logs", s.listActionLogs)

				// 数据统计
				mod.GET("/metrics/history", middleware.AdminAuth(), s.getMetricsHistory)
				mod.GET("/metrics/summary", middleware.AdminAuth(), s.getMetricsSummary)
				mod.GET("/metrics/basic", middleware.AdminAuth(), s.getMetricsBasic)
			}

			// 管理员中心（需要管理员权限）
			admin := auth.Group("/admin")
			admin.Use(middleware.AdminAuth())
			{
				admin.GET("/users", s.listUsers)
				admin.PUT("/users/:id/role", s.setUserRole)
			}
		}
	}
}
