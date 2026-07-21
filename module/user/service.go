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
	"strings"
)

// QueryUserInfo 校验 token 并返回用户基础资料。
func QueryUserInfo(ctx context.Context, token string) (*component.UserInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, component.ErrInvalidToken
	}
	return component.GetUserInfoByToken(ctx, token)
}
