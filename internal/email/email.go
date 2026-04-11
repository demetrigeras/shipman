package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// Enabled returns false when SMTP is not configured (dev with no email setup).
func (s *Service) Enabled() bool {
	return s.cfg.SMTPHost != "" && s.cfg.SMTPUser != ""
}

type InviteEmailData struct {
	RecipientEmail string
	RecipientRole  string // "shipowner", "charterer", "broker"
	DealTitle      string
	InviterName    string
	InviteLink     string // full URL, e.g. https://app.shipman.io/join?token=xxx
}

// SendInvite sends a deal invite email. Returns nil without sending when SMTP is not configured.
func (s *Service) SendInvite(data InviteEmailData) error {
	if !s.Enabled() {
		return nil
	}

	subject := fmt.Sprintf("You're invited to negotiate: %s", data.DealTitle)
	body := s.buildInviteHTML(data)

	return s.send(data.RecipientEmail, subject, body)
}

func (s *Service) buildInviteHTML(d InviteEmailData) string {
	roleLabel := map[string]string{
		"shipowner": "Ship Owner",
		"charterer": "Charterer",
		"broker":    "Broker",
	}[d.RecipientRole]
	if roleLabel == "" {
		roleLabel = d.RecipientRole
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;font-family:Inter,Arial,sans-serif;background:#f8fafc;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f8fafc;padding:40px 0;">
  <tr><td align="center">
    <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
      <!-- Header -->
      <tr>
        <td style="background:#0f172a;padding:28px 40px;">
          <h1 style="color:#ffffff;margin:0;font-size:22px;font-weight:700;">⚓ Shipman</h1>
          <p style="color:#94a3b8;margin:4px 0 0;font-size:13px;">Charter Party Negotiation Platform</p>
        </td>
      </tr>
      <!-- Body -->
      <tr>
        <td style="padding:36px 40px;">
          <h2 style="margin:0 0 8px;font-size:18px;color:#0f172a;">You've been invited to negotiate</h2>
          <p style="margin:0 0 24px;font-size:15px;color:#475569;">
            <strong>%s</strong> has invited you to join a charter party negotiation as <strong>%s</strong>.
          </p>
          <div style="background:#f1f5f9;border-radius:8px;padding:20px 24px;margin-bottom:28px;">
            <p style="margin:0 0 6px;font-size:13px;color:#64748b;text-transform:uppercase;letter-spacing:0.05em;">Deal</p>
            <p style="margin:0;font-size:17px;font-weight:600;color:#0f172a;">%s</p>
          </div>
          <p style="margin:0 0 24px;font-size:14px;color:#64748b;line-height:1.6;">
            Click the button below to join the deal room. You'll be asked to fill in your %s details
            so both parties can negotiate the charter party terms side by side.
          </p>
          <div style="text-align:center;margin:32px 0;">
            <a href="%s"
               style="background:#3b82f6;color:#ffffff;text-decoration:none;padding:14px 36px;border-radius:8px;font-size:15px;font-weight:600;display:inline-block;">
              Join Negotiation →
            </a>
          </div>
          <p style="margin:24px 0 0;font-size:13px;color:#94a3b8;text-align:center;">
            Or copy this link into your browser:<br>
            <span style="color:#64748b;word-break:break-all;">%s</span>
          </p>
        </td>
      </tr>
      <!-- Footer -->
      <tr>
        <td style="padding:20px 40px;border-top:1px solid #e2e8f0;">
          <p style="margin:0;font-size:12px;color:#94a3b8;text-align:center;">
            This invite expires in 7 days. If you weren't expecting this, you can safely ignore it.
          </p>
        </td>
      </tr>
    </table>
  </td></tr>
</table>
</body>
</html>`,
		d.InviterName, roleLabel,
		d.DealTitle,
		strings.ToLower(roleLabel),
		d.InviteLink, d.InviteLink,
	)
}

func (s *Service) send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	from := fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromAddress)

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	// Use STARTTLS (port 587)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.cfg.SMTPHost,
	}

	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err = conn.Mail(s.cfg.FromAddress); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	if err = conn.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	wc, err := conn.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer wc.Close()

	_, err = msg.WriteTo(wc)
	return err
}
