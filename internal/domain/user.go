package domain

import "errors"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrUserNotFound = errors.New("user not found")
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
