package validator

import "strings"

// Required 校验字符串去除空白后是否非空。
func Required(value string) bool {
	return strings.TrimSpace(value) != ""
}
