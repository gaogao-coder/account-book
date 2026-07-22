package handler

import (
	"github.com/gin-gonic/gin"
)

// NewRouter 创建 API 服务的 Gin 路由。
func NewRouter() *gin.Engine {
	router := gin.Default()

	RegisterRoutes(router)

	return router
}

// RegisterRoutes 注册所有业务路由。
func RegisterRoutes(router gin.IRoutes) {
	router.GET("/api/health/mysql", Hello)
	router.POST("/api/auth/douyin/login", Login)
	router.POST("/api/auth/token/refresh", RefreshToken)
	router.POST("/api/account/user_info", GetUserInfo)
}
