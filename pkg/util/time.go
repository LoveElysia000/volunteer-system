package util

import (
	"errors"
	"time"
)

// DateLayout 标准日期格式
const DateLayout = "2006-01-02"

// DateTimeLayout 标准日期时间格式
const DateTimeLayout = "2006-01-02 15:04:05"

// ParseDate 解析日期字符串，格式为 YYYY-MM-DD
func ParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("日期字符串不能为空")
	}
	return time.Parse(DateLayout, dateStr)
}

// ParseDateTime 解析日期时间字符串，格式为 YYYY-MM-DD HH:MM:SS
func ParseDateTime(dateTimeStr string) (time.Time, error) {
	if dateTimeStr == "" {
		return time.Time{}, errors.New("日期时间字符串不能为空")
	}
	return time.Parse(DateTimeLayout, dateTimeStr)
}

// ParsePastDate 解析日期字符串并校验是否为过去日期（不能是未来日期）
func ParsePastDate(dateStr string) (time.Time, error) {
	t, err := ParseDate(dateStr)
	if err != nil {
		return time.Time{}, err
	}
	if t.After(time.Now()) {
		return time.Time{}, errors.New("日期不能是未来日期")
	}
	return t, nil
}

// FormatDate 格式化时间为字符串 YYYY-MM-DD
func FormatDate(t time.Time) string {
	return t.Format(DateLayout)
}

// FormatDateTime 格式化时间为字符串 YYYY-MM-DD HH:MM:SS
func FormatDateTime(t time.Time) string {
	return t.Format(DateTimeLayout)
}

// FormatJoinDate 格式化加入日期，优先使用 joinedAt，否则使用 appliedAt
func FormatJoinDate(joinedAt *time.Time, appliedAt time.Time) string {
	if joinedAt != nil {
		return joinedAt.Format(DateTimeLayout)
	}
	return appliedAt.Format(DateTimeLayout)
}