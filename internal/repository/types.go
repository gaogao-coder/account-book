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
package repository

import (
	"sync"
)

var (
	mysqlHelloWorld *mysqlComponent

	componentMu sync.Mutex
)

// getMysqlComponent 返回登录链路依赖的 MySQL 组件单例。
func getMysqlComponent() (*mysqlComponent, error) {
	componentMu.Lock()
	defer componentMu.Unlock()
	if mysqlHelloWorld != nil {
		return mysqlHelloWorld, nil
	}
	component, err := NewMysqlComponent()
	if err != nil {
		return nil, err
	}
	mysqlHelloWorld = component
	return mysqlHelloWorld, nil
}

// getMysqlDB 返回底层 MySQL 连接池。
func getMysqlDB() (*mysqlComponent, error) {
	return getMysqlComponent()
}
