package app

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
)

func (c *App) SendSignupCode(email string, code string) error {

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

func (c *App) SendSignupVerificationEmail(email string) error {
	token := RandomString(73)

	link := fmt.Sprintf(`%s/signup?code=%s`, c.Config.SMTP.Domain, token)

	password := c.Config.SMTP.Password

	to := []string{email}

	body := fmt.Sprintf(`
		We're sending you this email to make sure that this address actually belongs to you. Follow the link below to create your new Shpong account.<br><br>
		%s
	`, link)

	message := []byte("From:" + c.Config.SMTP.Account + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: Create Your Shpong Account\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", password, password, c.Config.SMTP.Server)

	ad := fmt.Sprintf(`%s:%d`, c.Config.SMTP.Server, c.Config.SMTP.Port)

	err := smtp.SendMail(ad, auth, c.Config.SMTP.Account, to, message)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

/*

func (c *App) SendEmailUpdateVerificationEmail(email string) error {

	token := RandomString(73)

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	err := c.AddEmailVerification(ctx, email, token)
	if err != nil {
		log.Println(err)
	}

	link := fmt.Sprintf(`%s/verify?code=%s`, c.Config.App.Domain, token)

	from := fmt.Sprintf(`support@%s`, c.Config.App.Domain)
	password := c.Config.SMTP.Password

	to := []string{email}

	body := fmt.Sprintf(`
		You've added this email address to your shpong account. Follow the link below to verify this email.<br><br>
		%s
	`, link)

	log.Println("waht is body", body)

	message := []byte("From:" + "support@shpong.chat" + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: Verify Email Address Update\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", password, password, c.Config.SMTP.Server)

	ad := fmt.Sprintf(`%s:%s`, c.Config.SMTP.Server, c.Config.SMTP.Port)

	err = smtp.SendMail(ad, auth, from, to, message)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (c *App) SendPasswordResetEmail(email string) error {

	token := RandomString(73)

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	err := c.AddPasswordReset(ctx, email, token)
	if err != nil {
		log.Println(err)
	}

	link := fmt.Sprintf(`%s/password/reset?code=%s`, c.Config.App.Domain, token)

	from := fmt.Sprintf(`support@%s`, c.Config.App.Domain)
	password := c.Config.SMTP.Password

	to := []string{email}

	body := fmt.Sprintf(`
		You requested a password reset. Follow the link below to reset your password.<br><br>
		%s<br><br>
		If you did not request a password reset, you can safely ignore this email.
	`, link)

	log.Println("waht is body", body)

	message := []byte("From:" + "support@shpong.chat" + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: Reset Your shpong Password\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", password, password, c.Config.SMTP.Server)

	ad := fmt.Sprintf(`%s:%s`, c.Config.SMTP.Server, c.Config.SMTP.Port)

	err = smtp.SendMail(ad, auth, from, to, message)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (c *App) SendInvitationEmail(email string, user *User) error {

	token := RandomString(73)

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	err := c.AddEmailVerification(ctx, email, token)
	if err != nil {
		log.Println(err)
	}
	err = c.AddInvite(ctx, user.UserID, email, token)
	if err != nil {
		log.Println(err)
	}

	link := fmt.Sprintf(`%s/signup?code=%s`, c.Config.App.Domain, token)

	from := fmt.Sprintf(`support@%s`, c.Config.App.Domain)
	password := c.Config.SMTP.Password

	to := []string{email}

	username := user.UserID

	if !user.Federated {
		username = GetLocalPart(user.UserID)
		username = "@" + username
	}

	body := fmt.Sprintf(`
		Your friend <a href="https://%s/%s">%s</a> sent you an invite to join shpong. Follow the link below to create your shpong account.<br><br>
		%s
	`, c.Config.App.Domain, username, username, link)

	log.Println("waht is body", body)

	message := []byte("From:" + "support@shpong.chat" + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: You've been invited to join shpong!\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", password, password, c.Config.SMTP.Server)

	ad := fmt.Sprintf(`%s:%s`, c.Config.SMTP.Server, c.Config.SMTP.Port)

	err = smtp.SendMail(ad, auth, from, to, message)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (c *App) VerifyEmailUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		us := LoggedInUser(r)

		query := r.URL.Query()
		code := query.Get("code")

		if len(code) == 0 {
			c.NotFound(w, r)
			return
		}

		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, 7*time.Second)
		email, valid, err := c.GetEmailVerificationToken(ctx, code)
		if err != nil || !valid {
			log.Println(err)
			c.VerificationCodeInvalid(w, r)
			return
		}

		err = c.VerifyEmail(ctx, email)
		if err != nil {
			log.Println(err)
		} else {
			username := GetLocalPart(us.UserID)
			err = c.UnsafeUpdateEmail(ctx, username, email)
			if err != nil {
				log.Println(err)
			}
		}

		type page struct {
			BasePage
		}

		nonce := secure.CSPNonce(r.Context())

		t := &page{}

		t.Nonce = nonce
		t.LoggedInUser = us

		c.Templates.ExecuteTemplate(w, "email-update-success", t)
	}
}

func (c *App) EmailVerification(email, token string) error {

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	err := c.AddEmailVerification(ctx, email, token)
	if err != nil {
		log.Println(err)
	}

	client := postmark.NewApp(c.Config.SMTP.Server, c.Config.SMTP.Account)

	x := postmark.TemplatedEmail{
		TemplateId: c.Config.SMTP.Templates.VerifyEmail,
		TemplateModel: map[string]interface{}{
			"email":             email,
			"verification_code": token,
		},
		InlineCss: true,
		From:      "support@shpong.chat",
		To:        email,
		Tag:       "Veritificatoin Code",
		ReplyTo:   "support@shpong.chat",
	}
	log.Println(x)

	resp, err := client.SendTemplatedEmail(x)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(resp)

	return nil
}

func (c *App) EmailPasswordVerification(email, token string) error {

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 7*time.Second)
	err := c.AddPasswordReset(ctx, email, token)
	if err != nil {
		log.Println(err)
	}

	client := postmark.NewApp(c.Config.SMTP.Server, c.Config.SMTP.Account)

	x := postmark.TemplatedEmail{
		TemplateId: c.Config.SMTP.Templates.PasswordReset,
		TemplateModel: map[string]interface{}{
			"email":             email,
			"verification_code": token,
		},
		InlineCss: true,
		From:      "support@shpong.chat",
		To:        email,
		Tag:       "Veritificatoin Code",
		ReplyTo:   "support@shpong.chat",
	}
	log.Println(x)

	resp, err := client.SendTemplatedEmail(x)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(resp)

	return nil
}
*/
