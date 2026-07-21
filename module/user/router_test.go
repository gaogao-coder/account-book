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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type testResp struct {
	ErrNo  int                `json:"err_no"`
	ErrMsg string             `json:"err_msg"`
	Data   component.UserInfo `json:"data"`
}

// TestGetUserInfoWithoutToken 校验缺少 token 时返回未授权错误。
func TestGetUserInfoWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/account/get_user_info", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"err_msg":"invalid token"`) {
		t.Fatalf("expected invalid token response, got %s", resp.Body.String())
	}
}

// TestGetUserInfoWithInvalidJSON 校验非法 JSON 请求体返回参数错误。
func TestGetUserInfoWithInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/account/get_user_info", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"err_msg":"invalid request body"`) {
		t.Fatalf("expected invalid request body response, got %s", resp.Body.String())
	}
}

// TestGetUserInfoForNewLogin 校验新登录用户会返回默认用户资料。
func TestGetUserInfoForNewLogin(t *testing.T) {
	withQueryUserInfoStub(t, newFakeUserQuery())

	info := requestUserInfo(t, "token-a")
	if info.UserID == 0 {
		t.Fatalf("expected user id, got 0")
	}
	if info.Username == "" {
		t.Fatalf("expected default username")
	}
	if info.AvatarURL == "" {
		t.Fatalf("expected default avatar url")
	}
}

// TestGetUserInfoForRepeatedLogin 校验重复登录返回已保存的同一份用户资料。
func TestGetUserInfoForRepeatedLogin(t *testing.T) {
	withQueryUserInfoStub(t, newFakeUserQuery())

	first := requestUserInfo(t, "token-a")
	second := requestUserInfo(t, "token-a")

	if first.UserID != second.UserID || first.Username != second.Username || first.AvatarURL != second.AvatarURL {
		t.Fatalf("expected same user info, got %#v and %#v", first, second)
	}
}

// TestGetUserInfoDefaultUsernameUnique 校验不同新用户的默认用户名不重复。
func TestGetUserInfoDefaultUsernameUnique(t *testing.T) {
	withQueryUserInfoStub(t, newFakeUserQuery())

	first := requestUserInfo(t, "token-a")
	second := requestUserInfo(t, "token-b")

	if first.Username == second.Username {
		t.Fatalf("expected unique usernames, got %q", first.Username)
	}
}

// withQueryUserInfoStub 临时替换用户查询函数，避免单元测试依赖真实 MySQL。
func withQueryUserInfoStub(t *testing.T, fn queryUserInfoFunc) {
	t.Helper()
	old := queryUserInfo
	queryUserInfo = fn
	t.Cleanup(func() {
		queryUserInfo = old
	})
}

// newFakeUserQuery 返回一个内存版登录会话查询，用于覆盖首次和重复登录。
func newFakeUserQuery() queryUserInfoFunc {
	users := map[string]*component.UserInfo{}
	return func(_ context.Context, token string) (*component.UserInfo, error) {
		if token == "" {
			return nil, component.ErrInvalidToken
		}
		if info, ok := users[token]; ok {
			return info, nil
		}
		id := uint64(len(users) + 1)
		info := &component.UserInfo{
			UserID:    id,
			Username:  "用户" + strconv.FormatUint(id, 36),
			AvatarURL: "https://example.com/default-avatar.png",
		}
		users[token] = info
		return info, nil
	}
}

// requestUserInfo 请求用户信息接口并解析响应数据。
func requestUserInfo(t *testing.T, token string) component.UserInfo {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/account/get_user_info", strings.NewReader(`{"token":"`+token+`"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var parsed testResp
	if err := json.Unmarshal(resp.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return parsed.Data
}
