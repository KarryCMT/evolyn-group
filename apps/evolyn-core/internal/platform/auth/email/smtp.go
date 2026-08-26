package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

// SMTPSender 使用 SMTP STARTTLS/PLAIN 通道发送纯文本验证码邮件。账号密码
// 仅由部署配置注入；该结构不保存或输出邮件正文以外的敏感信息。
type SMTPSender struct {
	host        string
	port        int
	username    string
	password    string
	from        string
	implicitTLS bool
}

func NewSMTPSender(host string, port int, username, password, from string, implicitTLS bool) (*SMTPSender, error) {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	from = strings.TrimSpace(from)
	if host == "" || port <= 0 || from == "" {
		return nil, fmt.Errorf("smtp host/port/from must be configured")
	}
	if username != "" && password == "" {
		return nil, fmt.Errorf("smtp password is required when username is configured")
	}
	return &SMTPSender{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		implicitTLS: implicitTLS || port == 465,
	}, nil
}

func (s *SMTPSender) Send(ctx context.Context, to, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := s.dial()
	if err != nil {
		return err
	}
	defer client.Quit()

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	message := strings.Join([]string{
		"To: " + to,
		"From: " + s.from,
		"Subject: =?UTF-8?B?54G16KGN5LqR6YKu566x6aqM6K+B56CB?=",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("你的灵衍云邮箱验证码是：%s。验证码 5 分钟内有效，请勿向他人泄露。", code),
	}, "\r\n")
	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("smtp write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close message: %w", err)
	}
	return nil
}

// dial 同时支持 465 隐式 TLS 与 587 STARTTLS。smtp.SendMail 只能处理后者，
// 若直接向 465 发送明文 SMTP 命令，服务端通常会主动断开并返回 EOF。
func (s *SMTPSender) dial() (*smtp.Client, error) {
	address := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	if s.implicitTLS {
		connection, err := tls.Dial("tcp", address, &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, fmt.Errorf("smtp implicit tls dial: %w", err)
		}
		client, err := smtp.NewClient(connection, s.host)
		if err != nil {
			connection.Close()
			return nil, fmt.Errorf("smtp client: %w", err)
		}
		return client, nil
	}

	client, err := smtp.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			client.Close()
			return nil, fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return client, nil
}
