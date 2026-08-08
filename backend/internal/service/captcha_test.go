package service

import (
	"strings"
	"testing"

	"github.com/mojocn/base64Captcha"
)

// TestCaptchaGenerateAndVerify 生成验证码后，正确校验通过且一次性消费，重复校验失败
func TestCaptchaGenerateAndVerify(t *testing.T) {
	svc := NewCaptchaService()
	res, err := svc.Generate()
	if err != nil {
		t.Fatalf("生成验证码失败: %v", err)
	}
	if res.CaptchaID == "" {
		t.Error("captcha_id 不应为空")
	}
	if !strings.HasPrefix(res.CaptchaImage, "data:image/png;base64,") {
		t.Error("captcha_image 应为 base64 PNG 数据")
	}
	// 从全局内存 store 读取答案（不消费），验证校验路径
	answer := base64Captcha.DefaultMemStore.Get(res.CaptchaID, false)
	if answer == "" {
		t.Fatal("内存 store 应能读到验证码答案")
	}
	if !svc.Verify(res.CaptchaID, answer) {
		t.Error("正确验证码应校验通过")
	}
	if svc.Verify(res.CaptchaID, answer) {
		t.Error("验证码消费后再次校验应失败")
	}
}

// TestCaptchaVerifyWrongCode 错误验证码应校验失败
func TestCaptchaVerifyWrongCode(t *testing.T) {
	svc := NewCaptchaService()
	if svc.Verify("non-exist-id", "0000") {
		t.Error("不存在的验证码应校验失败")
	}
}
