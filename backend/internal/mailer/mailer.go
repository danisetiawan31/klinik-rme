package mailer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/resend/resend-go/v3"
)

// EmailSender defines the interface for sending transactional emails.
type EmailSender interface {
	SendInviteEmail(ctx context.Context, toEmail, inviteLink string) error
	SendResetEmail(ctx context.Context, toEmail, resetLink string) error
}

// ResendMailer implements EmailSender using the official Resend Go SDK.
type ResendMailer struct {
	client    *resend.Client
	fromEmail string
}

func NewResendMailer(apiKey, fromEmail string) *ResendMailer {
	return &ResendMailer{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
	}
}

func (r *ResendMailer) SendInviteEmail(ctx context.Context, toEmail, inviteLink string) error {
	// Wrap HTTP call with a strict 10-second timeout to prevent hanging connections
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	params := &resend.SendEmailRequest{
		From:    r.fromEmail,
		To:      []string{toEmail},
		Subject: "Undangan Akun Modul RME & Antrian Klinik",
		Html:    fmt.Sprintf("<p>Halo,</p><p>Anda telah diundang ke sistem Klinik RME. Silakan atur password Anda melalui link berikut:</p><p><a href=\"%s\">%s</a></p><p>Link ini berlaku selama 7 hari.</p>", inviteLink, inviteLink),
	}

	_, err := r.client.Emails.SendWithContext(timeoutCtx, params)
	if err != nil {
		return fmt.Errorf("failed to send invite email via Resend: %w", err)
	}
	return nil
}

func (r *ResendMailer) SendResetEmail(ctx context.Context, toEmail, resetLink string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	params := &resend.SendEmailRequest{
		From:    r.fromEmail,
		To:      []string{toEmail},
		Subject: "Instruksi Reset Password Akun Klinik RME",
		Html:    fmt.Sprintf("<p>Halo,</p><p>Permintaan reset password telah diterima. Silakan atur ulang password Anda melalui link berikut:</p><p><a href=\"%s\">%s</a></p><p>Link ini berlaku selama 1 jam.</p>", resetLink, resetLink),
	}

	_, err := r.client.Emails.SendWithContext(timeoutCtx, params)
	if err != nil {
		return fmt.Errorf("failed to send reset email via Resend: %w", err)
	}
	return nil
}

// MockMailer is a thread-safe mock implementation for unit testing.
type MockMailer struct {
	mu          sync.Mutex
	ShouldFail  bool
	SentInvites []string
	SentResets  []string
}

func NewMockMailer(shouldFail bool) *MockMailer {
	return &MockMailer{
		ShouldFail: shouldFail,
	}
}

func (m *MockMailer) SendInviteEmail(ctx context.Context, toEmail, inviteLink string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		return fmt.Errorf("simulated mailer network failure")
	}
	m.SentInvites = append(m.SentInvites, fmt.Sprintf("%s|%s", toEmail, inviteLink))
	return nil
}

func (m *MockMailer) SendResetEmail(ctx context.Context, toEmail, resetLink string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		return fmt.Errorf("simulated mailer network failure")
	}
	m.SentResets = append(m.SentResets, fmt.Sprintf("%s|%s", toEmail, resetLink))
	return nil
}
