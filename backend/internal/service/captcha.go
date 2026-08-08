package service

import (
	"github.com/mojocn/base64Captcha"
)

// CaptchaService 图形验证码服务：生成与校验。
// 验证码存储使用 base64Captcha 自带的内存 store（DefaultMemStore），单实例场景足够。
type CaptchaService struct {
	driver base64Captcha.Driver
}

// NewCaptchaService 创建验证码服务（4 位数字+字母，80 高 × 240 宽）
func NewCaptchaService() *CaptchaService {
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
	return &CaptchaService{driver: driver}
}

// CaptchaResult 验证码生成结果
type CaptchaResult struct {
	CaptchaID    string `json:"captcha_id"`    // 验证码唯一标识，登录时回传
	CaptchaImage string `json:"captcha_image"` // base64 编码的 PNG 图片
}

// Generate 生成一张验证码，返回验证码 ID 与 base64 图片
func (s *CaptchaService) Generate() (*CaptchaResult, error) {
	captcha := base64Captcha.NewCaptcha(s.driver, base64Captcha.DefaultMemStore)
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		return nil, err
	}
	return &CaptchaResult{CaptchaID: id, CaptchaImage: b64s}, nil
}

// Verify 校验验证码（校验一次即消费作废，clear=true 由 DefaultMemStore 在比对后删除）
func (s *CaptchaService) Verify(id, code string) bool {
	return base64Captcha.DefaultMemStore.Verify(id, code, true)
}
