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
package user

import (
	"context"
	"douyincloud-gin-demo/component"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var queryUserInfo queryUserInfoFunc = QueryUserInfo

// RegisterRoutes 注册用户 Module 的 HTTP 路由。
func RegisterRoutes(router gin.IRoutes) {
	router.POST("/api/account/get_user_info", GetUserInfo)
}

// GetUserInfo 返回当前 token 对应的用户基础信息。
func GetUserInfo(ctx *gin.Context) {
	var req getUserInfoReq
	if ctx.Request.Body != nil && ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			writeFailure(ctx, http.StatusBadRequest, "invalid request body", nil)
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
	case errors.Is(err, component.ErrInvalidToken):
		writeFailure(ctx, http.StatusUnauthorized, "invalid token", nil)
	case errors.Is(err, component.ErrUserNotFound):
		writeFailure(ctx, http.StatusNotFound, "user not found", nil)
	default:
		writeFailure(ctx, http.StatusServiceUnavailable, "user store unavailable", err)
	}
}

// writeSuccess 写入用户模块成功响应。
func writeSuccess(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, userResp{
		ErrNo:  0,
		ErrMsg: "success",
		Data:   data,
	})
}

// writeFailure 写入用户模块失败响应。
func writeFailure(ctx *gin.Context, status int, message string, err error) {
	if err != nil && status >= http.StatusInternalServerError {
		log.Printf("user request failed: %v", err)
	}
	ctx.JSON(status, userResp{
		ErrNo:  -1,
		ErrMsg: message,
		Data:   nil,
	})
}

type queryUserInfoFunc func(context.Context, string) (*component.UserInfo, error)
