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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const douyinUserTableName = "douyin_users"
const tokenTTL = 24 * time.Hour

var (
	authStoreMu    sync.Mutex
	authStoreReady bool
)

// SaveDouyinUser 保存或更新抖音小程序用户的开发者服务端登录态。
func SaveDouyinUser(ctx context.Context, openID string, unionID string, sessionKey string, token string) error {
	if err := ensureAuthStore(ctx); err != nil {
		return err
	}
	if err := ensureUserStore(ctx); err != nil {
		return err
	}
	mysqlComponent, err := getMysqlDB()
	if err != nil {
		return err
	}

	tx, err := mysqlComponent.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(
		ctx,
		fmt.Sprintf(
			`INSERT INTO %s (open_id, union_id, session_key_hash, token_hash, token_expires_at)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE union_id = VALUES(union_id), session_key_hash = VALUES(session_key_hash), token_hash = VALUES(token_hash), token_expires_at = VALUES(token_expires_at)`,
			douyinUserTableName,
		),
		openID,
		unionID,
		hashSecret(sessionKey),
		hashSecret(token),
		time.Now().Add(tokenTTL),
	); err != nil {
		return err
	}
	if err := initAccountingUser(ctx, tx, openID); err != nil {
		return err
	}
	return tx.Commit()
}

// ensureAuthStore 初始化抖音用户登录态表。
func ensureAuthStore(ctx context.Context) error {
	authStoreMu.Lock()
	defer authStoreMu.Unlock()
	if authStoreReady {
		return nil
	}
	mysqlComponent, err := getMysqlDB()
	if err != nil {
		return err
	}
	_, err = mysqlComponent.db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
open_id VARCHAR(128) NOT NULL UNIQUE,
union_id VARCHAR(128) NOT NULL DEFAULT '',
session_key_hash CHAR(64) NOT NULL,
token_hash CHAR(64) NOT NULL UNIQUE,
token_expires_at TIMESTAMP NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`,
		douyinUserTableName,
	))
	if err != nil {
		return err
	}
	if err := migrateAuthStore(ctx, mysqlComponent); err != nil {
		return err
	}
	authStoreReady = true
	return nil
}

func migrateAuthStore(ctx context.Context, mysqlComponent *mysqlComponent) error {
	statements := []string{
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN session_key_hash CHAR(64) NOT NULL DEFAULT ''", douyinUserTableName),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN token_hash CHAR(64) NOT NULL DEFAULT ''", douyinUserTableName),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN token_expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP", douyinUserTableName),
		fmt.Sprintf("ALTER TABLE %s ADD UNIQUE KEY token_hash_unique (token_hash)", douyinUserTableName),
	}
	for _, statement := range statements {
		if _, err := mysqlComponent.db.ExecContext(ctx, statement); err != nil && !isIgnorableAlterError(err) {
			return err
		}
	}
	return nil
}

// initAccountingUser 初始化家庭记账用户模块所需的基础用户资料。
func initAccountingUser(ctx context.Context, tx *sql.Tx, openID string) error {
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (douyin_open_id) VALUES (?)
ON DUPLICATE KEY UPDATE douyin_open_id = VALUES(douyin_open_id)`,
		usersTableName,
	), openID); err != nil {
		return err
	}

	var userID uint64
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT id FROM %s WHERE douyin_open_id = ? LIMIT 1`,
		usersTableName,
	), openID).Scan(&userID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT IGNORE INTO %s (user_id, nickname) VALUES (?, ?)`,
		userProfilesTableName,
	), userID, "抖音用户"); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, fmt.Sprintf(
		`INSERT IGNORE INTO %s (user_id) VALUES (?)`,
		userSettingsTableName,
	), userID)
	return err
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func isIgnorableAlterError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Duplicate key name") || strings.Contains(err.Error(), "Duplicate column name")
}
