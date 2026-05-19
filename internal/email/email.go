package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Config struct {
	SendGridAPIKey string
	TemplateID     string
	FromAddress    string
	FromName       string
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// Enabled returns false when SendGrid is not configured.
func (s *Service) Enabled() bool {
	return s.cfg.SendGridAPIKey != ""
}

type InviteEmailData struct {
	RecipientEmail string
	RecipientRole  string // "shipowner", "charterer", "broker"
	DealTitle      string
	InviterName    string
	InviteLink     string
}

// SendInvite sends a deal invite email via SendGrid Dynamic Template.
// Returns nil without sending when SendGrid is not configured.
func (s *Service) SendInvite(data InviteEmailData) error {
	if !s.Enabled() {
		return nil
	}

	roleLabel := map[string]string{
		"shipowner": "Ship Owner",
		"charterer": "Charterer",
		"broker":    "Broker",
	}[data.RecipientRole]
	if roleLabel == "" {
		roleLabel = data.RecipientRole
	}

	// Keys must match the Handlebars variables in your SendGrid template exactly.
	// The template uses {{INVITER_NAME}}, {{DEAL_TITLE}}, {{ROLE_LABEL}}, {{INVITE_LINK}}
	templateData := map[string]string{
		"INVITER_NAME": data.InviterName,
		"DEAL_TITLE":   data.DealTitle,
		"ROLE_LABEL":   roleLabel,
		"INVITE_LINK":  data.InviteLink,
	}

	return s.sendViaSendGrid(data.RecipientEmail, templateData)
}

// sendGridPayload matches SendGrid v3 mail/send API shape for dynamic templates.
type sendGridPayload struct {
	Personalizations []personalization `json:"personalizations"`
	From             address           `json:"from"`
	TemplateID       string            `json:"template_id"`
}

type personalization struct {
	To                  []address         `json:"to"`
	DynamicTemplateData map[string]string `json:"dynamic_template_data"`
}

type address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

func (s *Service) sendViaSendGrid(to string, data map[string]string) error {
	payload := sendGridPayload{
		Personalizations: []personalization{{
			To:                  []address{{Email: to}},
			DynamicTemplateData: data,
		}},
		From:       address{Email: s.cfg.FromAddress, Name: s.cfg.FromName},
		TemplateID: s.cfg.TemplateID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email marshal: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.SendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("email send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned %d", resp.StatusCode)
	}
	return nil
}

