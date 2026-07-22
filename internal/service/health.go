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
	"context"
	"douyincloud-gin-demo/internal/repository"
	"time"
)

const mysqlHealthTimeout = 5 * time.Second

// MysqlHealthResp 表示 MySQL 健康检查结果。
type MysqlHealthResp struct {
	Healthy        bool  `json:"healthy"`
	ResponseTimeMS int64 `json:"response_time_ms"`
}

// CheckMysqlHealth 检查登录链路依赖的 MySQL 连接状态。
func CheckMysqlHealth(ctx context.Context) (MysqlHealthResp, error) {
	start := time.Now()
	healthCtx, cancel := context.WithTimeout(ctx, mysqlHealthTimeout)
	defer cancel()
	err := repository.CheckMysqlHealth(healthCtx)
	responseTimeMS := time.Since(start).Milliseconds()
	if err != nil {
		return MysqlHealthResp{
			Healthy:        false,
			ResponseTimeMS: responseTimeMS,
		}, err
	}
	return MysqlHealthResp{
		Healthy:        true,
		ResponseTimeMS: responseTimeMS,
	}, nil
}
