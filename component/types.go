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
package component

import (
	"context"
	"fmt"
)

type HelloWorldComponent interface {
	GetName(ctx context.Context, key string) (name string, err error)
	SetName(ctx context.Context, key string, name string) error
}

const Mongo = "mongodb"
const Redis = "redis"
const Mysql = "mysql"

var (
	mongoHelloWorld *mongoComponent
	redisHelloWorld *redisComponent
	mysqlHelloWorld *mysqlComponent
)

// GetComponent 通过传入的component的名称返回实现了HelloWorldComponent接口的component。
func GetComponent(component string) (HelloWorldComponent, error) {
	switch component {
	case Mongo:
		return mongoHelloWorld, nil
	case Redis:
		return redisHelloWorld, nil
	case Mysql:
		return mysqlHelloWorld, nil
	default:
		return nil, fmt.Errorf("invalid component")
	}
}

// InitComponents 初始化项目依赖的组件。
func InitComponents() {
	mongoHelloWorld = NewMongoComponent()
	redisHelloWorld = NewRedisComponent()
	mysqlHelloWorld = NewMysqlComponent()
	ctx := context.TODO()
	err := mongoHelloWorld.SetName(ctx, "name", "mongodb")
	if err != nil {
		panic(err)
	}
	err = redisHelloWorld.SetName(ctx, "name", "redis")
	if err != nil {
		panic(err)
	}
	err = mysqlHelloWorld.SetName(ctx, "name", "mysql")
	if err != nil {
		panic(err)
	}
}
