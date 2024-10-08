package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"

	"github.com/firefart/go-webserver-template/internal/config"

	gomail "github.com/wneessen/go-mail"
)

type Interface interface {
	SendHTMLEmail(ctx context.Context, subject, body string) error
	SendTXTEmail(ctx context.Context, subject, body string) error
	SendMultipartEmail(ctx context.Context, subject, textBody, htmlBody string) error
}

type Mail struct {
	client *gomail.Client
	config config.Configuration
	logger *slog.Logger
}

// compile time check that struct implements the interface
var _ Interface = (*Mail)(nil)

type NullMailer struct{}

func (NullMailer) SendHTMLEmail(_ context.Context, _, _ string) error {
	return nil
}

func (NullMailer) SendTXTEmail(_ context.Context, _, _ string) error {
	return nil
}

func (NullMailer) SendMultipartEmail(_ context.Context, _, _, _ string) error {
	return nil
}

// compile time check that struct implements the interface
var _ Interface = (*NullMailer)(nil)

func New(config config.Configuration, logger *slog.Logger) (*Mail, error) {
	var options []gomail.Option

	options = append(options, gomail.WithTimeout(config.Mail.Timeout))
	options = append(options, gomail.WithPort(config.Mail.Port))
	if config.Mail.User != "" && config.Mail.Password != "" {
		options = append(options, gomail.WithSMTPAuth(gomail.SMTPAuthPlain))
		options = append(options, gomail.WithUsername(config.Mail.User))
		options = append(options, gomail.WithPassword(config.Mail.Password))
	}
	if config.Mail.SkipTLS {
		options = append(options, gomail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}

	// use either tls, starttls, or starttls with fallback to plaintext
	if config.Mail.TLS {
		options = append(options, gomail.WithSSL())
	} else if config.Mail.StartTLS {
		options = append(options, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	} else {
		options = append(options, gomail.WithTLSPortPolicy(gomail.TLSOpportunistic))
	}

	mailer, err := gomail.NewClient(config.Mail.Server, options...)
	if err != nil {
		return nil, fmt.Errorf("could not create mail client: %w", err)
	}

	return &Mail{
		client: mailer,
		config: config,
		logger: logger,
	}, nil
}

func (m *Mail) SendHTMLEmail(ctx context.Context, subject, body string) error {
	for _, to := range m.config.Mail.To {
		if err := m.send(ctx, to, subject, "", body); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mail) SendTXTEmail(ctx context.Context, subject, body string) error {
	for _, to := range m.config.Mail.To {
		if err := m.send(ctx, to, subject, body, ""); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mail) SendMultipartEmail(ctx context.Context, subject, textBody, htmlBody string) error {
	for _, to := range m.config.Mail.To {
		if err := m.send(ctx, to, subject, textBody, htmlBody); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mail) send(ctx context.Context, to string, subject, textContent, htmlContent string) error {
	if textContent == "" && htmlContent == "" {
		return fmt.Errorf("need a content to send email")
	}

	m.logger.Debug("sending email", slog.String("subject", subject), slog.String("to", to), slog.String("content-text", textContent), slog.String("html-content", htmlContent))

	msg := gomail.NewMsg(gomail.WithNoDefaultUserAgent())
	if err := msg.FromFormat(m.config.Mail.From.Name, m.config.Mail.From.Mail); err != nil {
		return err
	}
	if err := msg.To(to); err != nil {
		return err
	}
	msg.Subject(subject)
	if textContent != "" {
		msg.SetBodyString(gomail.TypeTextPlain, textContent)
	}
	if htmlContent != "" {
		msg.SetBodyString(gomail.TypeTextHTML, htmlContent)
	}

	var err error
	for i := 1; i <= m.config.Mail.Retries; i++ {
		err = m.client.DialAndSendWithContext(ctx, msg)
		if err == nil {
			return nil
		}
		// bail out on cancel
		if errors.Is(err, context.Canceled) {
			return err
		}
		m.logger.Error("error on sending email", slog.String("subject", subject), slog.Int("try", i), slog.String("err", err.Error()))
	}
	return fmt.Errorf("could not send mail %q after %d retries. Last error: %w", subject, m.config.Mail.Retries, err)
}
