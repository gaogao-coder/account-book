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
	"sync"
)

const (
	usersTableName        = "users"
	userProfilesTableName = "user_profiles"
	userSettingsTableName = "user_settings"
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
	Nickname           string `json:"nickname"`
	AvatarAssetID      string `json:"avatar_asset_id"`
	Gender             string `json:"gender"`
	Birthday           string `json:"birthday"`
	CurrentHouseholdID uint64 `json:"current_household_id"`
}

// GetUserInfoByToken 校验服务端 token 并查询对应用户基础资料。
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

	var row userInfoRow
	err = mysqlComponent.db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT u.id, u.phone, u.douyin_open_id,
COALESCE(p.nickname, ''), COALESCE(CAST(p.avatar_asset_id AS CHAR), ''),
COALESCE(p.gender, ''), COALESCE(DATE_FORMAT(p.birthday, '%%Y-%%m-%%d'), ''),
s.current_household_id
FROM %s d
LEFT JOIN %s u ON u.douyin_open_id = d.open_id
LEFT JOIN %s p ON p.user_id = u.id
LEFT JOIN %s s ON s.user_id = u.id
WHERE d.token_hash = ? AND d.token_expires_at > NOW()
LIMIT 1`,
		douyinUserTableName,
		usersTableName,
		userProfilesTableName,
		userSettingsTableName,
	), hashSecret(token)).Scan(
		&row.UserID,
		&row.Phone,
		&row.DouyinOpenID,
		&row.Nickname,
		&row.AvatarAssetID,
		&row.Gender,
		&row.Birthday,
		&row.CurrentHouseholdID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidToken
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
	Nickname           string
	AvatarAssetID      string
	Gender             string
	Birthday           string
	CurrentHouseholdID sql.NullInt64
}

// toUserInfo 将数据库可空字段转换为接口响应结构。
func (r userInfoRow) toUserInfo() *UserInfo {
	info := &UserInfo{
		UserID:        uint64(r.UserID.Int64),
		Phone:         r.Phone.String,
		DouyinOpenID:  r.DouyinOpenID.String,
		Nickname:      r.Nickname,
		AvatarAssetID: r.AvatarAssetID,
		Gender:        r.Gender,
		Birthday:      r.Birthday,
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
nickname VARCHAR(12) NOT NULL DEFAULT '',
avatar_asset_id BIGINT UNSIGNED NULL,
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
	userStoreReady = true
	return nil
}
