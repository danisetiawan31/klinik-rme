package mailer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

func TestMockMailer_SuccessAndFailure(t *testing.T) {
	ctx := context.Background()

	// 1. Success case
	mockSuccess := mailer.NewMockMailer(false)
	err := mockSuccess.SendInviteEmail(ctx, "user@test.com", "http://localhost:4200/set-password?token=123")
	require.NoError(t, err)
	assert.Len(t, mockSuccess.SentInvites, 1)
	assert.Equal(t, "user@test.com|http://localhost:4200/set-password?token=123", mockSuccess.SentInvites[0])

	err = mockSuccess.SendResetEmail(ctx, "user@test.com", "http://localhost:4200/set-password?token=456")
	require.NoError(t, err)
	assert.Len(t, mockSuccess.SentResets, 1)

	// 2. Simulated failure case
	mockFail := mailer.NewMockMailer(true)
	err = mockFail.SendInviteEmail(ctx, "user@test.com", "http://localhost:4200/set-password?token=123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated mailer network failure")

	err = mockFail.SendResetEmail(ctx, "user@test.com", "http://localhost:4200/set-password?token=456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated mailer network failure")
}
