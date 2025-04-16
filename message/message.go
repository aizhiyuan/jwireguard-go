package message

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type EmailSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Name     string
}

// SendMail 发送邮件（支持 HTML）
func (s *EmailSender) SendMail(to []string, subject string, body string, isHTML bool) error {
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	fromHeader := fmt.Sprintf("%s <%s>", s.Name, s.From)

	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = to[0]
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	// 构造消息体
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 连接 SMTP server
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.Host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return err
	}
	defer c.Quit()

	if err = c.Auth(auth); err != nil {
		return err
	}

	if err = c.Mail(s.From); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	return w.Close()
}
