package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// TranslateError 将验证错误转换为中文提示
func TranslateError(err error) string {
	if err == nil {
		return ""
	}

	// 尝试解包为 validator.ValidationErrors
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		var messages []string
		for _, e := range validationErrors {
			msg := translateFieldError(e)
			messages = append(messages, msg)
		}
		return strings.Join(messages, "; ")
	}

	// 如果不是验证错误，返回原始错误信息
	return err.Error()
}

func translateFieldError(e validator.FieldError) string {
	field := translateField(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s不能为空", field)
	case "min":
		if e.Type().String() == "string" {
			return fmt.Sprintf("%s长度不能少于%s个字符", field, e.Param())
		}
		return fmt.Sprintf("%s不能小于%s", field, e.Param())
	case "max":
		if e.Type().String() == "string" {
			return fmt.Sprintf("%s长度不能超过%s个字符", field, e.Param())
		}
		return fmt.Sprintf("%s不能大于%s", field, e.Param())
	case "email":
		return "邮箱格式不正确"
	case "len":
		return fmt.Sprintf("%s长度必须为%s", field, e.Param())
	case "eq":
		return fmt.Sprintf("%s必须等于%s", field, e.Param())
	case "ne":
		return fmt.Sprintf("%s不能等于%s", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s必须大于%s", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s必须大于或等于%s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s必须小于%s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s必须小于或等于%s", field, e.Param())
	default:
		return fmt.Sprintf("%s验证失败", field)
	}
}

func translateField(field string) string {
	fieldMap := map[string]string{
		"Username":         "用户名",
		"Email":            "邮箱",
		"Password":         "密码",
		"VerificationCode": "验证码",
		"Title":            "标题",
		"Content":          "内容",
		"Name":             "名称",
		"Description":      "描述",
		"Message":          "消息",
		"Bio":              "个人简介",
		"Location":         "地区",
		"Website":          "网站",
	}

	if translated, ok := fieldMap[field]; ok {
		return translated
	}
	return field
}
