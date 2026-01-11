package api

import "github.com/rpbox/server/internal/middleware"

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	v1 := s.router.Group("/api/v1")
	{
		// 公开接口
		v1.POST("/auth/register", s.register)
		v1.POST("/auth/login", s.login)

		// 需要认证的接口
		auth := v1.Group("")
		auth.Use(middleware.JWTAuth())
		{
			auth.GET("/profiles", s.listProfiles)
			auth.POST("/profiles", s.createProfile)
			auth.GET("/profiles/:id", s.getProfile)

			auth.GET("/stories", s.listStories)
			auth.POST("/stories", s.createStory)

			auth.GET("/items", s.listItems)
			auth.POST("/items", s.createItem)
		}
	}
}
