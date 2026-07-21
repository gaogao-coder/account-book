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
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"douyincloud-gin-demo/component"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultCode2SessionURL = "https://developer.toutiao.com/api/apps/v2/jscode2session"

var code2SessionHTTPClient = &http.Client{Timeout: 5 * time.Second}

type loginReq struct {
	Code string `json:"code"`
}

type loginResp struct {
	Token   string `json:"token"`
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid"`
}

type code2SessionReq struct {
	AppID  string `json:"appid"`
	Secret string `json:"secret"`
	Code   string `json:"code"`
	ACode  string `json:"anonymous_code"`
}

type code2SessionResp struct {
	ErrNo   int64  `json:"err_no"`
	ErrTips string `json:"err_tips"`
	LogID   string `json:"log_id"`
	Data    struct {
		SessionKey string `json:"session_key"`
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
	} `json:"data"`
}

// Login 使用 tt.login 返回的 code 完成静默登录或注册。
func Login(ctx *gin.Context) {
	var req loginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Failure(ctx, badRequest("invalid request body"))
		return
	}
	if req.Code == "" {
		Failure(ctx, badRequest("code is required"))
		return
	}

	session, err := code2Session(ctx.Request.Context(), req.Code)
	if err != nil {
		Failure(ctx, err)
		return
	}

	token, err := newToken()
	if err != nil {
		Failure(ctx, internalFailure("failed to create login token", err))
		return
	}
	if err := component.SaveDouyinUser(ctx.Request.Context(), session.Data.OpenID, session.Data.UnionID, session.Data.SessionKey, token); err != nil {
		Failure(ctx, dependencyUnavailable("login store unavailable", err))
		return
	}

	Success(ctx, loginResp{
		Token:   token,
		OpenID:  session.Data.OpenID,
		UnionID: session.Data.UnionID,
	})
}

// code2Session 调用抖音开放平台接口，用 code 换取 open_id 和 session_key。
func code2Session(ctx context.Context, code string) (*code2SessionResp, error) {
	appID := firstEnv("DOUYIN_APP_ID", "APP_ID")
	secret := firstEnv("DOUYIN_APP_SECRET", "APP_SECRET")
	if appID == "" || secret == "" {
		return nil, dependencyUnavailable("login service is not configured", fmt.Errorf("DOUYIN_APP_ID and DOUYIN_APP_SECRET are required"))
	}

	body, err := json.Marshal(code2SessionReq{
		AppID:  appID,
		Secret: secret,
		Code:   code,
		ACode:  "",
	})
	if err != nil {
		return nil, internalFailure("failed to build login request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultCode2SessionURL, bytes.NewReader(body))
	if err != nil {
		return nil, internalFailure("failed to build login request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := code2SessionHTTPClient.Do(httpReq)
	if err != nil {
		return nil, upstreamFailure("login upstream unavailable", err)
	}
	defer httpResp.Body.Close()

	var resp code2SessionResp
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, upstreamFailure("invalid login upstream response", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, upstreamFailure("login upstream returned an error", fmt.Errorf("code2session http status %d", httpResp.StatusCode))
	}
	if resp.ErrNo != 0 {
		return nil, upstreamFailure("login upstream rejected the request", fmt.Errorf("code2session failed: %s", resp.ErrTips))
	}
	if resp.Data.OpenID == "" || resp.Data.SessionKey == "" {
		return nil, upstreamFailure("invalid login upstream response", fmt.Errorf("code2session response missing openid or session_key"))
	}
	return &resp, nil
}

// newToken 生成开发者服务端登录凭证。
func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// firstEnv 返回第一个非空环境变量值。
func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
