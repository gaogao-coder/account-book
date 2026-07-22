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
	"douyincloud-gin-demo/internal/config"
	"douyincloud-gin-demo/internal/domain"
	"douyincloud-gin-demo/internal/repository"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultCode2SessionURL = "https://developer.toutiao.com/api/apps/v2/jscode2session"

var (
	ErrLoginConfigMissing = errors.New("login config missing")
	ErrLoginUserCheck     = errors.New("login user check failed")
	ErrLoginOpenIDExists  = errors.New("login openid already exists")
	ErrLoginStorage       = errors.New("login storage unavailable")
	ErrLoginToken         = errors.New("login token generation failed")
	ErrLoginUpstream      = errors.New("login upstream failed")
)

var code2SessionHTTPClient = &http.Client{Timeout: 5 * time.Second}

// LoginResp 表示抖音小程序登录成功后的业务结果。
type LoginResp struct {
	Token   string `json:"token"`
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid"`
}

// RefreshTokenResp 表示刷新服务端登录凭证后的业务结果。
type RefreshTokenResp struct {
	Token string `json:"token"`
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
func Login(ctx context.Context, code string) (*LoginResp, error) {
	session, err := code2Session(ctx, code)
	if err != nil {
		return nil, err
	}

	// 创建新用户前检查 OpenID 是否已存在，避免重复创建同一抖音用户。
	userExists, err := repository.GetDouyinUserByOpenID(ctx, session.Data.OpenID)
	if err != nil {
		return nil, fmt.Errorf("%w: check user existence failed: %v", ErrLoginUserCheck, err)
	}
	if userExists {
		return nil, fmt.Errorf("%w: user with openid %s already exists", ErrLoginOpenIDExists, session.Data.OpenID)
	}

	token, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLoginToken, err)
	}
	if err := repository.SaveDouyinUser(ctx, session.Data.OpenID, session.Data.UnionID, session.Data.SessionKey, token); err != nil {
		if errors.Is(err, repository.ErrDouyinOpenIDExists) {
			return nil, fmt.Errorf("%w: user with openid %s already exists", ErrLoginOpenIDExists, session.Data.OpenID)
		}
		return nil, fmt.Errorf("%w: %v", ErrLoginStorage, err)
	}

	return &LoginResp{
		Token:   token,
		OpenID:  session.Data.OpenID,
		UnionID: session.Data.UnionID,
	}, nil
}

// RefreshToken 使用当前未过期 token 换取新的服务端登录凭证。
func RefreshToken(ctx context.Context, token string) (*RefreshTokenResp, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, domain.ErrInvalidToken
	}

	nextToken, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLoginToken, err)
	}
	if err := repository.RefreshDouyinUserToken(ctx, token, nextToken); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrLoginStorage, err)
	}
	return &RefreshTokenResp{Token: nextToken}, nil
}

// code2Session 调用抖音开放平台接口，用 code 换取 open_id 和 session_key。
func code2Session(ctx context.Context, code string) (*code2SessionResp, error) {
	appID := config.FirstEnv("DOUYIN_APP_ID", "APP_ID")
	secret := config.FirstEnv("DOUYIN_APP_SECRET", "APP_SECRET")
	if appID == "" || secret == "" {
		return nil, fmt.Errorf("%w: DOUYIN_APP_ID and DOUYIN_APP_SECRET are required", ErrLoginConfigMissing)
	}

	body, err := json.Marshal(code2SessionReq{
		AppID:  appID,
		Secret: secret,
		Code:   code,
		ACode:  "",
	})
	if err != nil {
		return nil, fmt.Errorf("%w: build code2session request: %v", ErrLoginUpstream, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultCode2SessionURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: build code2session request: %v", ErrLoginUpstream, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := code2SessionHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: call code2session: %v", ErrLoginUpstream, err)
	}
	defer httpResp.Body.Close()

	var resp code2SessionResp
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("%w: decode code2session response: %v", ErrLoginUpstream, err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: code2session http status %d", ErrLoginUpstream, httpResp.StatusCode)
	}
	if resp.ErrNo != 0 {
		return nil, fmt.Errorf("%w: code2session failed: %s", ErrLoginUpstream, resp.ErrTips)
	}
	if resp.Data.OpenID == "" || resp.Data.SessionKey == "" {
		return nil, fmt.Errorf("%w: code2session response missing openid or session_key", ErrLoginUpstream)
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
