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

			auth.GET("/items", s.listItems)
			auth.POST("/items", s.createItem)

			// 账号备份（以账号为单位）
			auth.GET("/account-backups", s.listAccountBackups)
			auth.GET("/account-backups/:account_id", s.getAccountBackup)
			auth.POST("/account-backups", s.upsertAccountBackup)
			auth.DELETE("/account-backups/:account_id", s.deleteAccountBackup)
			auth.GET("/account-backups/:account_id/versions", s.getAccountBackupVersions)
		}
	}
}
