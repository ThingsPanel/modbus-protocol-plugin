package services

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// AuthLimiter 认证限流器
type AuthLimiter struct {
	failureCount map[string]int        // IP -> 连续失败次数
	blockUntil   map[string]time.Time  // IP -> 解封时间
	lastLogTime  map[string]time.Time  // IP -> 上次日志输出时间
	mutex        sync.RWMutex
	enabled      bool
	threshold    int           // 失败阈值
	blockTime    time.Duration // 封禁时长
}

// NewAuthLimiter 创建认证限流器
func NewAuthLimiter() *AuthLimiter {
	enabled := viper.GetBool("auth_limiter.enabled")
	if !enabled {
		enabled = true // 默认启用
	}

	threshold := viper.GetInt("auth_limiter.failure_threshold")
	if threshold <= 0 {
		threshold = 3 // 默认3次
	}

	blockDuration := viper.GetDuration("auth_limiter.block_duration")
	if blockDuration <= 0 {
		blockDuration = 3 * time.Minute // 默认3分钟
	}

	logrus.Infof("认证限流器初始化: enabled=%v, threshold=%d, blockTime=%v", enabled, threshold, blockDuration)

	return &AuthLimiter{
		failureCount: make(map[string]int),
		blockUntil:   make(map[string]time.Time),
		lastLogTime:  make(map[string]time.Time),
		enabled:      enabled,
		threshold:    threshold,
		blockTime:    blockDuration,
	}
}

// IsBlocked 检查IP是否被封禁
func (al *AuthLimiter) IsBlocked(ip string) bool {
	if !al.enabled {
		return false
	}

	al.mutex.RLock()
	defer al.mutex.RUnlock()

	// 懒清理：检查时清理过期记录
	if blockTime, exists := al.blockUntil[ip]; exists {
		if time.Now().Before(blockTime) {
			return true
		}
		// 过期了，清理记录
		delete(al.blockUntil, ip)
		delete(al.failureCount, ip)
		delete(al.lastLogTime, ip)
	}
	return false
}

// RecordFailure 记录认证失败
func (al *AuthLimiter) RecordFailure(ip string) {
	if !al.enabled {
		return
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	al.failureCount[ip]++
	count := al.failureCount[ip]

	if count >= al.threshold {
		al.blockUntil[ip] = time.Now().Add(al.blockTime)
		logrus.Warnf("IP认证限流触发: IP=%s, 失败次数=%d, 封禁时间=%v", ip, count, al.blockTime)
	}
}

// RecordSuccess 记录认证成功
func (al *AuthLimiter) RecordSuccess(ip string) {
	if !al.enabled {
		return
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	// 清零失败计数
	delete(al.failureCount, ip)
	delete(al.blockUntil, ip)
	delete(al.lastLogTime, ip)
}

// ShouldLogBlock 检查是否应该输出封禁日志（限制日志频率：每分钟最多1条）
func (al *AuthLimiter) ShouldLogBlock(ip string) bool {
	if !al.enabled {
		return false
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	now := time.Now()
	lastLog, exists := al.lastLogTime[ip]

	// 如果没有记录或者距离上次日志超过1分钟，则可以输出日志
	if !exists || now.Sub(lastLog) >= time.Minute {
		al.lastLogTime[ip] = now
		return true
	}

	return false
}