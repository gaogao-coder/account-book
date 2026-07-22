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
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	// mysql地址
	mysqlAddr = ""
	// mysql用户名
	mysqlUserName = ""
	// mysql密码
	mysqlPassword = ""
	// mysql数据库名
	mysqlDataBase = "demo"
)

type mysqlComponent struct {
	db *sql.DB
}

// NewMysqlComponent 初始化登录链路使用的 MySQL 组件。
func NewMysqlComponent() (*mysqlComponent, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysqlUserName, mysqlPassword, mysqlAddr, mysqlDataBase)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return &mysqlComponent{db: db}, nil
}

// CheckMysqlHealth 测试登录链路依赖的 MySQL 连接是否可用。
func CheckMysqlHealth(ctx context.Context) error {
	mysqlComponent, err := getMysqlDB()
	if err != nil {
		return err
	}
	return mysqlComponent.db.PingContext(ctx)
}

// init 项目启动时，会从环境变量中获取mysql的地址、用户名、密码和数据库名。
func init() {
	mysqlAddr = os.Getenv("MYSQL_ADDRESS")
	mysqlUserName = os.Getenv("MYSQL_USERNAME")
	mysqlPassword = os.Getenv("MYSQL_PASSWORD")
	if dataBase := os.Getenv("MYSQL_DATABASE"); dataBase != "" {
		mysqlDataBase = dataBase
	}
}
