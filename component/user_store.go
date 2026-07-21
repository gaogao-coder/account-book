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
	"errors"
	"fmt"
	"strconv"
	"sync"
)

const (
	usersTableName        = "users"
	userProfilesTableName = "user_profiles"
	userSettingsTableName = "user_settings"
	defaultAvatarURL      = "https://lf3-static.bytednsdoc.com/obj/eden-cn/default-avatar.png"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrUserNotFound = errors.New("user not found")

	userStoreMu    sync.Mutex
	userStoreReady bool
)

// UserInfo 表示用户模块对外返回的基础用户资料。
type UserInfo struct {
	UserID             uint64 `json:"user_id"`
	Phone              string `json:"phone"`
	DouyinOpenID       string `json:"douyin_open_id"`
	Username           string `json:"username"`
	AvatarURL          string `json:"avatar_url"`
	Gender             string `json:"gender"`
	Birthday           string `json:"birthday"`
	CurrentHouseholdID uint64 `json:"current_household_id"`
}

// GetUserInfoByToken 校验服务端 token，并在首次登录时初始化用户基础资料。
func GetUserInfoByToken(ctx context.Context, token string) (*UserInfo, error) {
	if err := ensureAuthStore(ctx); err != nil {
		return nil, err
	}
	if err := ensureUserStore(ctx); err != nil {
		return nil, err
	}
	mysqlComponent, err := getMysqlDB()
	if err != nil {
		return nil, err
	}

	tx, err := mysqlComponent.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var openID string
	err = tx.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT open_id FROM %s WHERE token_hash = ? AND token_expires_at > NOW() LIMIT 1`,
		douyinUserTableName,
	), hashSecret(token)).Scan(&openID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	if err := initAccountingUser(ctx, tx, openID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return queryUserInfoByOpenID(ctx, mysqlComponent, openID)
}

// queryUserInfoByOpenID 按抖音 open_id 查询用户基础资料。
func queryUserInfoByOpenID(ctx context.Context, mysqlComponent *mysqlComponent, openID string) (*UserInfo, error) {
	var row userInfoRow
	err := mysqlComponent.db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT u.id, u.phone, u.douyin_open_id,
COALESCE(p.nickname, ''), COALESCE(p.avatar_url, ''),
COALESCE(p.gender, ''), COALESCE(DATE_FORMAT(p.birthday, '%%Y-%%m-%%d'), ''),
s.current_household_id
FROM %s u
LEFT JOIN %s p ON p.user_id = u.id
LEFT JOIN %s s ON s.user_id = u.id
WHERE u.douyin_open_id = ?
LIMIT 1`,
		usersTableName,
		userProfilesTableName,
		userSettingsTableName,
	), openID).Scan(
		&row.UserID,
		&row.Phone,
		&row.DouyinOpenID,
		&row.Username,
		&row.AvatarURL,
		&row.Gender,
		&row.Birthday,
		&row.CurrentHouseholdID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if !row.UserID.Valid {
		return nil, ErrUserNotFound
	}

	return row.toUserInfo(), nil
}

type userInfoRow struct {
	UserID             sql.NullInt64
	Phone              sql.NullString
	DouyinOpenID       sql.NullString
	Username           string
	AvatarURL          string
	Gender             string
	Birthday           string
	CurrentHouseholdID sql.NullInt64
}

// toUserInfo 将数据库可空字段转换为接口响应结构。
func (r userInfoRow) toUserInfo() *UserInfo {
	info := &UserInfo{
		UserID:       uint64(r.UserID.Int64),
		Phone:        r.Phone.String,
		DouyinOpenID: r.DouyinOpenID.String,
		Username:     r.Username,
		AvatarURL:    r.AvatarURL,
		Gender:       r.Gender,
		Birthday:     r.Birthday,
	}
	if r.CurrentHouseholdID.Valid {
		info.CurrentHouseholdID = uint64(r.CurrentHouseholdID.Int64)
	}
	return info
}

// ensureUserStore 初始化用户模块 V1.0 所需的最小数据表。
func ensureUserStore(ctx context.Context) error {
	userStoreMu.Lock()
	defer userStoreMu.Unlock()
	if userStoreReady {
		return nil
	}
	mysqlComponent, err := getMysqlDB()
	if err != nil {
		return err
	}
	statements := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
phone VARCHAR(32) NOT NULL DEFAULT '',
douyin_open_id VARCHAR(128) NULL UNIQUE,
ali_user_id VARCHAR(128) NOT NULL DEFAULT '',
status VARCHAR(32) NOT NULL DEFAULT 'active',
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`, usersTableName),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
user_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
nickname VARCHAR(32) NOT NULL DEFAULT '',
avatar_asset_id BIGINT UNSIGNED NULL,
avatar_url VARCHAR(512) NOT NULL DEFAULT '',
gender VARCHAR(16) NULL,
birthday DATE NULL,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`, userProfilesTableName),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
user_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
current_household_id BIGINT UNSIGNED NULL,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`, userSettingsTableName),
	}
	for _, statement := range statements {
		if _, err := mysqlComponent.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := migrateUserStore(ctx, mysqlComponent); err != nil {
		return err
	}
	userStoreReady = true
	return nil
}

// migrateUserStore 补齐用户模块表结构的向后兼容字段。
func migrateUserStore(ctx context.Context, mysqlComponent *mysqlComponent) error {
	statements := []string{
		fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN nickname VARCHAR(32) NOT NULL DEFAULT ''", userProfilesTableName),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN avatar_url VARCHAR(512) NOT NULL DEFAULT ''", userProfilesTableName),
	}
	for _, statement := range statements {
		if _, err := mysqlComponent.db.ExecContext(ctx, statement); err != nil && !isIgnorableAlterError(err) {
			return err
		}
	}
	return nil
}

// defaultUsername 按用户 ID 生成系统默认用户名，避免静默登录用户重名。
func defaultUsername(userID uint64) string {
	return "用户" + strconv.FormatUint(userID, 36)
}
