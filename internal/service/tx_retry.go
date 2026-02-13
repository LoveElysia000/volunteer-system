package service

import (
	"errors"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	// MySQL: deadlock found when trying to get lock.
	deadlockErrCode        uint16 = 1213
	// MySQL: lock wait timeout exceeded.
	lockWaitTimeoutErrCode uint16 = 1205
	// 最多重试次数（包含首次执行）。
	maxTxRetryTimes               = 3
	// 线性退避基准时长，避免重试瞬时碰撞。
	txRetryBackoffBaseMs          = 20
)

// withTransaction 使用原生 DB.Transaction，并在死锁/锁等待超时时重试。
func (s *Service) withTransaction(fn func(tx *gorm.DB) error) error {
	var err error
	for attempt := 0; attempt < maxTxRetryTimes; attempt++ {
		err = s.repo.DB.WithContext(s.ctx).Transaction(fn)
		if err == nil {
			return nil
		}
		if !isRetryableTxError(err) || attempt == maxTxRetryTimes-1 {
			return err
		}
		// 线性退避，降低多事务同时重试再次冲突的概率。
		time.Sleep(time.Duration(attempt+1) * txRetryBackoffBaseMs * time.Millisecond)
	}
	return err
}

// isRetryableTxError 判断是否属于可重试的事务冲突错误。
func isRetryableTxError(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == deadlockErrCode || mysqlErr.Number == lockWaitTimeoutErrCode
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock found") || strings.Contains(msg, "lock wait timeout exceeded")
}
