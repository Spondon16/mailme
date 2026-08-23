package api

import (
	"context"
	"fmt"
)

// Compile-time assertion that MailtmClient implements Provider.
var _ Provider = (*MailtmClient)(nil)

// Provider is the interface implemented by all email providers.
//
// Commands operate against Provider instead of a concrete client, so a new
// provider can be added by implementing this interface and registering it in
// NewProvider without touching any command logic.
type Provider interface {
	// Name returns the provider identifier (e.g. "mailtm").
	Name() string

	// GetDomains returns the domains available for account creation.
	GetDomains() ([]Domain, error)

	// CreateAccount creates a new account and returns it (including its JWT).
	CreateAccount(email, password string) (*Account, error)

	// GetMessages lists messages for the given page (1-based).
	GetMessages(page int) ([]Message, error)

	// GetMessage returns full details of a specific message.
	GetMessage(messageID string) (*MessageDetail, error)

	// SetSeen updates the seen/read status of a message.
	SetSeen(messageID string, seen bool) error

	// GetSource returns the raw RFC 822 source of a message.
	GetSource(messageID string) (string, error)

	// DownloadMessageEML downloads the full .eml raw data for a message.
	DownloadMessageEML(messageID string) ([]byte, error)

	// DeleteMessage deletes a specific message.
	DeleteMessage(messageID string) error

	// DeleteAccount deletes an account from the remote server.
	DeleteAccount(accountID string) error

	// DownloadAttachment downloads an attachment by ID.
	DownloadAttachment(messageID, attachmentID string) ([]byte, error)

	// ListenMessagesSSE connects to the real-time SSE stream for new messages.
	ListenMessagesSSE(ctx context.Context, accountID string, onMessage func(msg *Message)) error

	// Client auth / session state.
	SetToken(token string)
	SetCredentials(email, password string)
	OnTokenUpdate(fn func(token string))
}

// NewProvider returns a Provider for the named provider. An empty name
// resolves to the default provider (mailtm). An unknown name returns an error.
func NewProvider(name string) (Provider, error) {
	switch name {
	case "mailtm", "":
		return NewMailtmClient(), nil
	case "tempmail.plus", "tempmailc", "mailnesia":
		return NewThirdPartyClient(name)
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}
