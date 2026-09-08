package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func shouldUseSMTPLoginAuth() bool {
	if SMTPForceAuthLogin {
		return true
	}
	return isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer)
}

func getSMTPAuth() smtp.Auth {
	return AutoSMTPAuth(SMTPAccount, SMTPToken)
}

func shouldAuthenticateSMTP() bool {
	return SMTPAccount != "" && SMTPToken != ""
}

func smtpTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         SMTPServer,
		InsecureSkipVerify: SMTPInsecureSkipVerify, // #nosec G402 -- admin-controlled SMTP compatibility option.
	}
}

func newSMTPClient(addr string) (*smtp.Client, error) {
	if SMTPSSLEnabled || (SMTPPort == 465 && !SMTPStartTLSEnabled) {
		conn, err := tls.Dial("tcp", addr, smtpTLSConfig())
		if err != nil {
			return nil, err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if SMTPStartTLSEnabled {
		startTLSSupported, _ := client.Extension("STARTTLS")
		if !startTLSSupported {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(smtpTLSConfig()); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

func SendEmail(subject string, receiver string, content string) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return err2
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	mail := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: %s\r\n"+ // 添加 Message-ID 头
		"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		receiver, SystemName, SMTPFrom, encodedSubject, time.Now().Format(time.RFC1123Z), id, content))
	return sendSMTPMessage(receiver, mail)
}

func SendEmailWithAttachments(subject, receiver, content string, attachments []EmailAttachment) error {
	if len(attachments) == 0 {
		return SendEmail(subject, receiver, content)
	}
	if SMTPFrom == "" {
		SMTPFrom = SMTPAccount
	}
	id, err := generateMessageID()
	if err != nil {
		return err
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fmt.Fprintf(&body, "To: %s\r\nFrom: %s <%s>\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", receiver, SystemName, SMTPFrom, encodedSubject, time.Now().Format(time.RFC1123Z), id, writer.Boundary())
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "base64")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return err
	}
	if err := writeBase64MIME(htmlPart, []byte(content)); err != nil {
		return err
	}
	for i := range attachments {
		filename := strings.NewReplacer("\r", "", "\n", "", "\"", "").Replace(filepath.Base(attachments[i].Filename))
		if filename == "" {
			filename = "invoice"
		}
		contentType := strings.TrimSpace(attachments[i].ContentType)
		if parsedType, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
			contentType = parsedType
		} else {
			contentType = "application/octet-stream"
		}
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", contentType)
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"invoice%s\"; filename*=UTF-8''%s", filepath.Ext(filename), url.PathEscape(filename)))
		header.Set("Content-Transfer-Encoding", "base64")
		part, createErr := writer.CreatePart(header)
		if createErr != nil {
			return createErr
		}
		if writeErr := writeBase64MIME(part, attachments[i].Data); writeErr != nil {
			return writeErr
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return sendSMTPMessage(receiver, body.Bytes())
}

func writeBase64MIME(writer io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := fmt.Fprintf(writer, "%s\r\n", encoded[:76]); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	_, err := fmt.Fprintf(writer, "%s\r\n", encoded)
	return err
}

func sendSMTPMessage(receiver string, mail []byte) error {
	auth := getSMTPAuth()
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	client, err := newSMTPClient(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if shouldAuthenticateSMTP() {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(SMTPFrom); err != nil {
		return err
	}
	for _, receiver := range to {
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(mail)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = client.Quit()
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}
