package validator

import (
	"errors"
	"strings"
	"testing"

	v10 "github.com/go-playground/validator/v10"
)

type validateSample struct {
	Username string `validate:"required"`
	Email    string `validate:"email"`
	Password string `validate:"min=6"`
}

func TestTranslateError(t *testing.T) {
	validator := v10.New()
	err := validator.Struct(validateSample{})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	message := TranslateError(err)
	if !strings.Contains(message, "用户名不能为空") {
		t.Fatalf("expected username message, got %s", message)
	}
	if !strings.Contains(message, "邮箱格式不正确") {
		t.Fatalf("expected email message, got %s", message)
	}
	if !strings.Contains(message, "密码长度不能少于6个字符") {
		t.Fatalf("expected password message, got %s", message)
	}
}

func TestTranslateErrorPassthrough(t *testing.T) {
	err := errors.New("boom")
	if TranslateError(err) != "boom" {
		t.Fatalf("expected passthrough error")
	}
}
