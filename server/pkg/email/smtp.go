package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPClient SMTP客户端
type SMTPClient struct {
	config *SMTPConfig
}

// NewSMTPClient 创建SMTP客户端
func NewSMTPClient(config *SMTPConfig) *SMTPClient {
	return &SMTPClient{config: config}
}

// SendMail 发送邮件
func (c *SMTPClient) SendMail(to, subject, body string) error {
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	// 邮件头
	header := make(map[string]string)
	header["From"] = c.config.From
	header["To"] = to
	header["Subject"] = subject
	header["Content-Type"] = "text/html; charset=UTF-8"

	// 组装邮件内容
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// 126邮箱需要TLS
	if c.config.Port == 465 {
		return c.sendMailTLS(addr, auth, to, []byte(message))
	}

	return smtp.SendMail(addr, auth, c.config.From, []string{to}, []byte(message))
}

// sendMailTLS 使用TLS发送邮件
func (c *SMTPClient) sendMailTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	// TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         c.config.Host,
	}

	// 连接到服务器
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("new client failed: %w", err)
	}
	defer client.Close()

	// 认证
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}

	// 设置发件人
	if err = client.Mail(c.config.From); err != nil {
		return fmt.Errorf("set mail from failed: %w", err)
	}

	// 设置收件人
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("set rcpt to failed: %w", err)
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("get data writer failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("write message failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("close writer failed: %w", err)
	}

	return client.Quit()
}

// SendVerificationCode 发送验证码邮件
func (c *SMTPClient) SendVerificationCode(to, code string) error {
	subject := "RPBox 注册验证码"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4a5568; border-bottom: 2px solid #4299e1; padding-bottom: 10px;">
            RPBox 邮箱验证
        </h2>
        <p>您好，</p>
        <p>感谢您注册 RPBox！您的验证码是：</p>
        <div style="background-color: #f7fafc; border: 2px solid #4299e1; border-radius: 5px; padding: 20px; margin: 20px 0; text-align: center;">
            <span style="font-size: 32px; font-weight: bold; color: #2d3748; letter-spacing: 5px;">%s</span>
        </div>
        <p style="color: #718096;">验证码有效期为 <strong>5分钟</strong>，请尽快完成验证。</p>
        <p style="color: #718096;">如果这不是您的操作，请忽略此邮件。</p>
        <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 30px 0;">
        <p style="font-size: 12px; color: #a0aec0; text-align: center;">
            此邮件由 RPBox 系统自动发送，请勿回复。
        </p>
    </div>
</body>
</html>
`, code)

	return c.SendMail(to, subject, body)
}

// ValidateEmail 简单的邮箱格式验证
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) < 5 {
		return false
	}
	if !strings.Contains(email, "@") {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}
