package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

// VerificationService 验证码服务
type VerificationService struct {
	redis *redis.Client
}

// NewVerificationService 创建验证码服务
func NewVerificationService(redisClient *redis.Client) *VerificationService {
	return &VerificationService{
		redis: redisClient,
	}
}

// GenerateCode 生成6位数字验证码
func (s *VerificationService) GenerateCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)

	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("generate random number failed: %w", err)
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

// SaveCode 保存验证码到Redis，5分钟有效期
func (s *VerificationService) SaveCode(ctx context.Context, email, code string) error {
	key := fmt.Sprintf("verification:email:%s", email)
	return s.redis.Set(ctx, key, code, 5*time.Minute).Err()
}

// VerifyCode 验证验证码
func (s *VerificationService) VerifyCode(ctx context.Context, email, code string) (bool, error) {
	key := fmt.Sprintf("verification:email:%s", email)

	// 获取存储的验证码
	storedCode, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// 验证码不存在或已过期
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get verification code failed: %w", err)
	}

	// 验证成功后删除验证码
	if storedCode == code {
		s.redis.Del(ctx, key)
		return true, nil
	}

	return false, nil
}

// CheckRateLimit 检查发送频率限制（1分钟内只能发送1次）
func (s *VerificationService) CheckRateLimit(ctx context.Context, email string) (bool, error) {
	key := fmt.Sprintf("verification:ratelimit:%s", email)

	// 检查是否存在
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check rate limit failed: %w", err)
	}

	if exists > 0 {
		// 还在冷却期
		return false, nil
	}

	// 设置冷却期（1分钟）
	err = s.redis.Set(ctx, key, "1", 1*time.Minute).Err()
	if err != nil {
		return false, fmt.Errorf("set rate limit failed: %w", err)
	}

	return true, nil
}
