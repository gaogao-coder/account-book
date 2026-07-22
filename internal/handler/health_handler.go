package handler

import (
	"douyincloud-gin-demo/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Hello 处理 MySQL 健康检查请求。
func Hello(ctx *gin.Context) {
	resp, err := service.CheckMysqlHealth(ctx.Request.Context())
	if err != nil {
		log.Printf("mysql health check failed: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, Response{
			Error:   http.StatusServiceUnavailable,
			Message: "MySQL 连接不可用",
			Data:    resp,
		})
		return
	}
	Success(ctx, resp)
}
