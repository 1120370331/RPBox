package api

import "github.com/rpbox/server/internal/middleware"

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	v1 := s.router.Group("/api/v1")
	{
		// 公开接口
		v1.POST("/auth/register", s.register)
		v1.POST("/auth/login", s.login)

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
			auth.POST("/items/:id/like", s.likeItem)
			auth.DELETE("/items/:id/like", s.unlikeItem)
			auth.POST("/items/:id/favorite", s.favoriteItem)
			auth.DELETE("/items/:id/favorite", s.unfavoriteItem)

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
		}
	}
}
