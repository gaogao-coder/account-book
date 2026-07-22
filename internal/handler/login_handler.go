package handler

import (
	"douyincloud-gin-demo/internal/domain"
	"douyincloud-gin-demo/internal/service"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Code string `json:"code"`
}

type refreshTokenReq struct {
	Token string `json:"token"`
}

// Login 处理抖音小程序静默登录请求，并在登录成功后触发默认用户资料初始化。
func Login(ctx *gin.Context) {
	var req loginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Failure(ctx, InvalidArgument("请求体不是合法 JSON"))
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		Failure(ctx, InvalidArgument("code 不能为空"))
		return
	}

	resp, err := service.Login(ctx.Request.Context(), req.Code)
	if err != nil {
		writeLoginError(ctx, err)
		return
	}
	Success(ctx, resp)
}

// RefreshToken 使用当前未过期 token 换取新的服务端登录凭证。
func RefreshToken(ctx *gin.Context) {
	var req refreshTokenReq
	if ctx.Request.Body != nil && ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			Failure(ctx, InvalidArgument("请求体不是合法 JSON"))
			return
		}
	}
	token := firstToken(req.Token, bearerToken(ctx.GetHeader("Authorization")))
	resp, err := service.RefreshToken(ctx.Request.Context(), token)
	if err != nil {
		writeRefreshTokenError(ctx, err)
		return
	}
	Success(ctx, resp)
}

// writeLoginError 将登录服务错误转换为 HTTP 响应。
func writeLoginError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLoginOpenIDExists):
		Failure(ctx, Conflict("OpenID 已存在，不能重复创建用户"))
	case errors.Is(err, service.ErrLoginConfigMissing), errors.Is(err, service.ErrLoginUserCheck), errors.Is(err, service.ErrLoginStorage):
		Failure(ctx, DependencyUnavailable("登录服务不可用", err))
	case errors.Is(err, service.ErrLoginUpstream):
		Failure(ctx, UpstreamFailure("抖音登录上游不可用", err))
	default:
		Failure(ctx, InternalFailure("登录凭证生成失败", err))
	}
}

// writeRefreshTokenError 将 token 刷新错误转换为 HTTP 响应。
func writeRefreshTokenError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		Failure(ctx, Unauthorized("登录态无效或已过期"))
	case errors.Is(err, service.ErrLoginStorage):
		Failure(ctx, DependencyUnavailable("登录服务不可用", err))
	default:
		Failure(ctx, InternalFailure("登录凭证刷新失败", err))
	}
}
