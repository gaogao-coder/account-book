/*
Copyright (year) Bytedance Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package handler

import (
	"context"
	"douyincloud-gin-demo/internal/domain"
	"douyincloud-gin-demo/internal/service"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

var queryUserInfo queryUserInfoFunc = service.QueryUserInfo

// GetUserInfo 返回当前 token 对应的用户基础信息。
func GetUserInfo(ctx *gin.Context) {
	var req getUserInfoReq
	if ctx.Request.Body != nil && ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			Failure(ctx, InvalidArgument("请求体不是合法 JSON"))
			return
		}
	}
	token := firstToken(req.Token, bearerToken(ctx.GetHeader("Authorization")))
	info, err := queryUserInfo(ctx.Request.Context(), token)
	if err != nil {
		writeUserInfoError(ctx, err)
		return
	}
	writeSuccess(ctx, info)
}

// firstToken 返回第一个非空 token。
func firstToken(tokens ...string) string {
	for _, token := range tokens {
		if strings.TrimSpace(token) != "" {
			return token
		}
	}
	return ""
}

// bearerToken 从 Authorization 请求头解析 Bearer token。
func bearerToken(header string) string {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// writeUserInfoError 将用户模块错误转换为 HTTP 响应。
func writeUserInfoError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		Failure(ctx, Unauthorized("登录态无效或已过期"))
	case errors.Is(err, domain.ErrUserNotFound):
		Failure(ctx, NotFound("用户不存在"))
	default:
		Failure(ctx, DependencyUnavailable("用户存储不可用", err))
	}
}

// writeSuccess 写入用户模块成功响应。
func writeSuccess(ctx *gin.Context, data interface{}) {
	Success(ctx, data)
}

type queryUserInfoFunc func(context.Context, string) (*domain.UserInfo, error)
