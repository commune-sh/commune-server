package app

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
)

func (c *App) SendVerificationCode(email string, code string) error {

	password := c.Config.SMTP.Password

	to := []string{email}

	var body bytes.Buffer

	type Values struct {
		Code string
	}

	v := Values{
		Code: code,
	}

	c.Templates.ExecuteTemplate(&body, "verification-code", v)

	message := []byte("From:" + c.Config.SMTP.Account + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + code + " is your code\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body.String() + "\r\n")

	auth := smtp.PlainAuth("", password, password, c.Config.SMTP.Server)

	ad := fmt.Sprintf(`%s:%d`, c.Config.SMTP.Server, c.Config.SMTP.Port)

	err := smtp.SendMail(ad, auth, c.Config.SMTP.Account, to, message)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
