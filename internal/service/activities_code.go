package service

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	attendanceCodeLength  = 6
	attendanceCodeDigits  = "23456789"
	attendanceCodeLetters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	attendanceCodeCharset = attendanceCodeDigits + attendanceCodeLetters
)

// validateAttendanceCodeForActivity 根据码类型校验活动签到码或签退码是否匹配且未过期。
func validateAttendanceCodeForActivity(activity *model.Activity, inputCode string, codeType int32) error {
	if activity == nil {
		return errors.New("活动不存在")
	}

	switch codeType {
	case model.AttendanceCodeTypeCheckIn:
		return validateAttendanceCodeValue(
			inputCode,
			activity.CheckInCode,
			activity.CheckInCodeHash,
			activity.CheckInCodeExpireAt,
			"签到码错误或已过期",
		)
	case model.AttendanceCodeTypeCheckOut:
		return validateAttendanceCodeValue(
			inputCode,
			activity.CheckOutCode,
			activity.CheckOutCodeHash,
			activity.CheckOutCodeExpireAt,
			"签退码错误或已过期",
		)
	default:
		return errors.New("码类型不合法")
	}
}

// validateAttendanceCodeValue 统一处理码存在性、过期性与值匹配校验。
func validateAttendanceCodeValue(inputCode, expectedCode, expectedCodeHash string, expireAt *time.Time, errMsg string) error {
	normalizedInputCode := strings.TrimSpace(inputCode)
	normalizedExpectedCode := strings.TrimSpace(expectedCode)
	normalizedExpectedCodeHash := strings.TrimSpace(expectedCodeHash)
	if expireAt != nil && time.Now().After(*expireAt) {
		return errors.New(errMsg)
	}

	// 优先使用哈希字段校验；保留明文字段回退以兼容历史数据。
	if normalizedExpectedCodeHash != "" {
		// Prefer hashed comparison when hash exists; plaintext fallback is for legacy rows only.
		inputCodeHash, err := util.HashSensitiveField(normalizedInputCode)
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(inputCodeHash), []byte(normalizedExpectedCodeHash)) != 1 {
			return errors.New(errMsg)
		}
		return nil
	}

	if normalizedExpectedCode == "" {
		return errors.New(errMsg)
	}
	if subtle.ConstantTimeCompare([]byte(normalizedInputCode), []byte(normalizedExpectedCode)) != 1 {
		return errors.New(errMsg)
	}
	return nil
}

// generateAttendanceCodeForWrite 生成随机码、哈希值和过期时间，用于写入活动码字段。
func generateAttendanceCodeForWrite(now time.Time, validMinutes int32) (string, string, *time.Time, error) {
	code, err := generateRandomAttendanceCode(attendanceCodeLength)
	if err != nil {
		return "", "", nil, err
	}
	codeHash, err := util.HashSensitiveField(strings.TrimSpace(code))
	if err != nil {
		return "", "", nil, err
	}
	return code, codeHash, buildAttendanceCodeExpireAt(now, validMinutes), nil
}

// isRequestFieldProvided checks whether a field is explicitly provided in query or JSON body.
func isRequestFieldProvided(c *app.RequestContext, field string) bool {
	if c == nil || strings.TrimSpace(field) == "" {
		return false
	}
	if _, ok := c.GetQuery(field); ok {
		return true
	}

	raw := c.GetRawData()
	if len(raw) == 0 {
		return false
	}
	body := make(map[string]json.RawMessage)
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	_, ok := body[field]
	return ok
}

// buildAttendanceCodeExpireAt 根据有效分钟数计算过期时间；<=0 表示不过期。
func buildAttendanceCodeExpireAt(now time.Time, validMinutes int32) *time.Time {
	if validMinutes <= 0 {
		return nil
	}
	expireAt := now.Add(time.Duration(validMinutes) * time.Minute)
	return &expireAt
}

// generateRandomAttendanceCode 生成固定长度码，且至少包含 1 位数字与 1 位字母。
func generateRandomAttendanceCode(length int) (string, error) {
	if length < 2 {
		return "", errors.New("无效的签到签退码长度")
	}

	digitPos, err := util.RandomIndex(length)
	if err != nil {
		return "", err
	}
	letterPos, err := util.RandomIndex(length)
	if err != nil {
		return "", err
	}
	for letterPos == digitPos {
		letterPos, err = util.RandomIndex(length)
		if err != nil {
			return "", err
		}
	}

	chars := []byte(attendanceCodeCharset)
	digits := []byte(attendanceCodeDigits)
	letters := []byte(attendanceCodeLetters)

	result := make([]byte, length)
	// 先固定一个数字位和一个字母位，确保复杂度下限。
	digitIdx, err := util.RandomIndex(len(digits))
	if err != nil {
		return "", err
	}
	letterIdx, err := util.RandomIndex(len(letters))
	if err != nil {
		return "", err
	}
	result[digitPos] = digits[digitIdx]
	result[letterPos] = letters[letterIdx]

	for i := range result {
		if i == digitPos || i == letterPos {
			continue
		}
		charIdx, idxErr := util.RandomIndex(len(chars))
		if idxErr != nil {
			return "", idxErr
		}
		result[i] = chars[charIdx]
	}
	return string(result), nil
}
