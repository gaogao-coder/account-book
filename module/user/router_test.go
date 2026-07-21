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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestGetUserInfoWithoutToken 校验缺少 token 时返回未授权错误。
func TestGetUserInfoWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/account/user", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/api/account/user", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"err_msg":"invalid request body"`) {
		t.Fatalf("expected invalid request body response, got %s", resp.Body.String())
	}
}
