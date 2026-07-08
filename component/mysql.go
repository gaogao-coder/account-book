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

const mysqlTableName = "demo"

type mysqlComponent struct {
	db *sql.DB
}

// GetName 从MySQL中获取名称。
func (m *mysqlComponent) GetName(ctx context.Context, key string) (name string, err error) {
	err = m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s WHERE `key` = ?", mysqlTableName), key).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// SetName 将名称写入MySQL。
func (m *mysqlComponent) SetName(ctx context.Context, key string, name string) error {
	_, err := m.db.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s (`key`, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)", mysqlTableName),
		key,
		name,
	)
	return err
}

// NewMysqlComponent 初始化一个实现了HelloWorldComponent接口的MysqlComponent。
func NewMysqlComponent() *mysqlComponent {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysqlUserName, mysqlPassword, mysqlAddr, mysqlDataBase)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("mysqlClient init error. err %s\n", err)
		panic("mysql open error")
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.PingContext(context.TODO()); err != nil {
		fmt.Printf("mysqlClient init error. err %s\n", err)
		panic(fmt.Sprintf("mysql init failed. err %s\n", err))
	}

	component := &mysqlComponent{db: db}
	if err := component.initTable(context.TODO()); err != nil {
		fmt.Printf("mysqlClient init table error. err %s\n", err)
		panic("mysql init table error")
	}
	return component
}

// initTable 初始化MySQL示例表。
func (m *mysqlComponent) initTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (`key` VARCHAR(128) NOT NULL PRIMARY KEY, value VARCHAR(255) NOT NULL)",
		mysqlTableName,
	))
	return err
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
