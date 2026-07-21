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
	"douyincloud-gin-demo/component"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const mysqlHealthTimeout = 5 * time.Second

type mysqlHealthResp struct {
	Healthy        bool  `json:"healthy"`
	ResponseTimeMS int64 `json:"response_time_ms"`
}

// Hello 检查登录链路依赖的 MySQL 连接状态。
func Hello(ctx *gin.Context) {
	start := time.Now()
	healthCtx, cancel := context.WithTimeout(ctx.Request.Context(), mysqlHealthTimeout)
	defer cancel()
	err := component.CheckMysqlHealth(healthCtx)
	responseTimeMS := time.Since(start).Milliseconds()
	if err != nil {
		log.Printf("mysql health check failed: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, &Resp{
			ErrNo:  -1,
			ErrMsg: "mysql connection unavailable",
			Data: mysqlHealthResp{
				Healthy:        false,
				ResponseTimeMS: responseTimeMS,
			},
		})
		return
	}
	Success(ctx, mysqlHealthResp{
		Healthy:        true,
		ResponseTimeMS: responseTimeMS,
	})
}

func Failure(ctx *gin.Context, err error) {
	status := http.StatusInternalServerError
	errMsg := "internal server error"
	internalErr := err
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		status = apiErr.StatusCode
		errMsg = apiErr.Message
		internalErr = apiErr.Internal
	}
	if internalErr != nil && status >= http.StatusInternalServerError {
		log.Printf("request failed: %v", internalErr)
	}
	resp := &Resp{
		ErrNo:  -1,
		ErrMsg: errMsg,
	}
	ctx.JSON(status, resp)
}

func Success(ctx *gin.Context, data interface{}) {
	resp := &Resp{
		ErrNo:  0,
		ErrMsg: "success",
		Data:   data,
	}
	ctx.JSON(200, resp)
}

type Resp struct {
	ErrNo  int         `json:"err_no"`
	ErrMsg string      `json:"err_msg"`
	Data   interface{} `json:"data"`
}

type apiError struct {
	StatusCode int
	Message    string
	Internal   error
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	if e.Internal != nil {
		return e.Internal.Error()
	}
	return e.Message
}

func badRequest(message string) error {
	return &apiError{
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func dependencyUnavailable(message string, err error) error {
	return &apiError{
		StatusCode: http.StatusServiceUnavailable,
		Message:    message,
		Internal:   err,
	}
}

func upstreamFailure(message string, err error) error {
	return &apiError{
		StatusCode: http.StatusBadGateway,
		Message:    message,
		Internal:   err,
	}
}

func internalFailure(message string, err error) error {
	return &apiError{
		StatusCode: http.StatusInternalServerError,
		Message:    message,
		Internal:   err,
	}
}
