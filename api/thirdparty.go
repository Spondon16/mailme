package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	tempmail "github.com/josskixg/TempMail-UnofficialAPI/go"
)

// thirdPartyStatelessProviders lists the TempMail-UnofficialAPI provider
// names that are safe to drive from mailme: their inbox/message lookups are
// pure calls keyed by the email address itself, with no server session kept
// only in memory.
//
// mailme is a stateless CLI — every command (`generate`, then later `inbox`,
// `read`, `watch`, ...) is a fresh process. Providers like guerrillamail,
// tempmail.lol, and dropmail require a session/auth token that the
// TempMail-UnofficialAPI library only ever stores in an unexported struct
// field with no getter, so it can't be extracted after GenerateEmail() and
// persisted to accounts.json the way mail.tm's JWT is. Wiring one of those
// in would work for `generate` and then silently break on the very next
// command. They're deliberately excluded until upstream exposes a way to
// save/restore that state. See README's "Additional Providers" section.
var thirdPartyStatelessProviders = map[string]bool{
	"tempmail.plus": true,
	"tempmailc":     true,
	"mailnesia":     true,
}

// ThirdPartyClient adapts one of the address-stateless TempMail-UnofficialAPI
// providers to mailme's Provider interface.
type ThirdPartyClient struct {
	name  string
	email string
}

// NewThirdPartyClient returns a client for a supported stateless provider.
func NewThirdPartyClient(name string) (*ThirdPartyClient, error) {
	if !thirdPartyStatelessProviders[name] {
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
	return &ThirdPartyClient{name: name}, nil
}

func (c *ThirdPartyClient) lib() (tempmail.TempMailProvider, error) {
	p, err := tempmail.NewProvider(c.name, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	return p, nil
}

func (c *ThirdPartyClient) Name() string { return c.name }

// GetDomains is not supported: these services assign the full address
// (username + domain) themselves, with no way to request a specific one.
func (c *ThirdPartyClient) GetDomains() ([]Domain, error) {
	return nil, fmt.Errorf("%s: domain selection is not supported; the service assigns a random address automatically", c.name)
}

// CreateAccount ignores the requested email/password (unsupported by these
// providers) and generates a random address instead.
func (c *ThirdPartyClient) CreateAccount(email, password string) (*Account, error) {
	p, err := c.lib()
	if err != nil {
		return nil, err
	}
	addr, err := p.GenerateEmail()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	c.email = addr
	return &Account{
		ID:       addr,
		Email:    addr,
		Password: password,
		Provider: c.name,
	}, nil
}

func (c *ThirdPartyClient) GetMessages(page int) ([]Message, error) {
	if page > 1 {
		// No pagination support; only the first page is ever available.
		return nil, nil
	}
	p, err := c.lib()
	if err != nil {
		return nil, err
	}
	msgs, err := p.GetInbox(c.email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		out[i] = Message{
			ID:        m.ID,
			Subject:   m.Subject,
			FromAddr:  m.Sender,
			Intro:     m.Preview,
			CreatedAt: timeOrNil(m.Date),
		}
	}
	return out, nil
}

// GetMessage fetches one message's full body.
//
// It deliberately does NOT go through the TempMail-UnofficialAPI library's
// ReadMessage(): that method refuses to run unless GenerateEmail() was
// already called on that exact in-memory instance (it caches the mailbox
// address in an unexported field). Since ThirdPartyClient builds a fresh
// library instance per call — and mailme itself is a fresh process per
// command — that check always fails with "no email — call GenerateEmail()
// first", even though the underlying HTTP endpoints are plain lookups by
// address and don't actually require it. mailme calls the same endpoints
// directly instead, passing c.email explicitly.
func (c *ThirdPartyClient) GetMessage(messageID string) (*MessageDetail, error) {
	if c.email == "" {
		return nil, fmt.Errorf("%s: no active address", c.name)
	}
	switch c.name {
	case "tempmail.plus":
		return c.getMessageTempmailPlus(messageID)
	case "tempmailc":
		return c.getMessageTempmailc(messageID)
	case "mailnesia":
		return c.getMessageMailnesia(messageID)
	default:
		return nil, fmt.Errorf("%s: reading a message is not supported by this provider", c.name)
	}
}

var thirdPartyHTTP = &http.Client{Timeout: 20 * time.Second}

func thirdPartyGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mailme-cli/1.0")
	req.Header.Set("Accept", "application/json, text/html")
	resp, err := thirdPartyHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func (c *ThirdPartyClient) getMessageTempmailPlus(messageID string) (*MessageDetail, error) {
	endpoint := fmt.Sprintf("https://tempmail.plus/api/mails/%s?email=%s",
		url.QueryEscape(messageID), url.QueryEscape(c.email))
	body, err := thirdPartyGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("tempmail.plus: %w", err)
	}
	var m struct {
		MailID      json.RawMessage `json:"mail_id"`
		FromMail    string          `json:"from_mail"`
		From        string          `json:"from"`
		Subject     string          `json:"subject"`
		Text        string          `json:"text"`
		HTML        string          `json:"html"`
		Result      *bool           `json:"result"`
		Error       string          `json:"error"`
		Attachments []struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("tempmail.plus: parse error: %w", err)
	}
	if m.Error != "" {
		return nil, fmt.Errorf("tempmail.plus: %s", m.Error)
	}
	if m.Result != nil && !*m.Result {
		return nil, fmt.Errorf("tempmail.plus: message not found")
	}
	from := m.FromMail
	if from == "" {
		from = m.From
	}
	if from == "" && m.Subject == "" && m.Text == "" && m.HTML == "" {
		return nil, fmt.Errorf("tempmail.plus: message not found")
	}
	var attachments []Attachment
	for _, a := range m.Attachments {
		attachments = append(attachments, Attachment{Filename: a.Filename, ContentType: a.ContentType})
	}
	return &MessageDetail{
		ID:          messageID,
		Subject:     m.Subject,
		FromAddr:    from,
		To:          c.email,
		Text:        m.Text,
		HTML:        m.HTML,
		Attachments: attachments,
	}, nil
}

func (c *ThirdPartyClient) getMessageTempmailc(messageID string) (*MessageDetail, error) {
	endpoint := fmt.Sprintf("https://tempmailc.com/api/v1/message?msg_id=%s&email=%s",
		url.QueryEscape(messageID), url.QueryEscape(c.email))
	body, err := thirdPartyGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("tempmailc: %w", err)
	}
	var m struct {
		From     string `json:"from"`
		FromMail string `json:"from_mail"`
		Subject  string `json:"subject"`
		Text     string `json:"text"`
		BodyText string `json:"body_text"`
		HTML     string `json:"html"`
		BodyHTML string `json:"body_html"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("tempmailc: parse error: %w", err)
	}
	if m.Error != "" {
		return nil, fmt.Errorf("tempmailc: %s", m.Error)
	}
	from := m.From
	if from == "" {
		from = m.FromMail
	}
	text := m.Text
	if text == "" {
		text = m.BodyText
	}
	html := m.HTML
	if html == "" {
		html = m.BodyHTML
	}
	if from == "" && m.Subject == "" && text == "" && html == "" {
		return nil, fmt.Errorf("tempmailc: message not found")
	}
	return &MessageDetail{
		ID:       messageID,
		Subject:  m.Subject,
		FromAddr: from,
		To:       c.email,
		Text:     text,
		HTML:     html,
	}, nil
}

var (
	thirdPartyTagRe   = regexp.MustCompile(`(?s)<[^>]+>`)
	thirdPartyBlankRe = regexp.MustCompile(`\n{3,}`)
)

func stripTags(html string) string {
	s := thirdPartyTagRe.ReplaceAllString(html, "\n")
	s = thirdPartyBlankRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func (c *ThirdPartyClient) getMessageMailnesia(messageID string) (*MessageDetail, error) {
	username := c.email
	if i := strings.Index(c.email, "@"); i > 0 {
		username = c.email[:i]
	}
	endpoint := fmt.Sprintf("https://mailnesia.com/mailbox/%s/%s", url.PathEscape(username), url.PathEscape(messageID))
	body, err := thirdPartyGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("mailnesia: %w", err)
	}
	html := string(body)
	return &MessageDetail{
		ID:   messageID,
		To:   c.email,
		Text: stripTags(html),
		HTML: html,
	}, nil
}

// SetSeen is a no-op: these providers don't track read/unread state.
func (c *ThirdPartyClient) SetSeen(messageID string, seen bool) error { return nil }

func (c *ThirdPartyClient) GetSource(messageID string) (string, error) {
	return "", fmt.Errorf("%s: raw message source is not supported by this provider", c.name)
}

func (c *ThirdPartyClient) DownloadMessageEML(messageID string) ([]byte, error) {
	return nil, fmt.Errorf("%s: .eml download is not supported by this provider", c.name)
}

func (c *ThirdPartyClient) DeleteMessage(messageID string) error {
	return fmt.Errorf("%s: deleting a single message is not supported by this provider", c.name)
}

// DeleteAccount deletes the whole inbox (there's no separate account ID for
// these providers; the address itself is the identity).
func (c *ThirdPartyClient) DeleteAccount(accountID string) error {
	p, err := c.lib()
	if err != nil {
		return err
	}
	email := c.email
	if email == "" {
		email = accountID
	}
	ok, err := p.DeleteEmail(email)
	if err != nil {
		return fmt.Errorf("%s: %w", c.name, err)
	}
	if !ok {
		return fmt.Errorf("%s: delete failed", c.name)
	}
	return nil
}

func (c *ThirdPartyClient) DownloadAttachment(messageID, attachmentID string) ([]byte, error) {
	return nil, fmt.Errorf("%s: attachment download is not supported by this provider", c.name)
}

// ListenMessagesSSE always returns an error immediately: none of these
// providers offer real-time push. Callers (mailme's `watch` command) are
// expected to fall back to polling on error, which is exactly what happens.
func (c *ThirdPartyClient) ListenMessagesSSE(ctx context.Context, accountID string, onMessage func(msg *Message)) error {
	return fmt.Errorf("%s: real-time updates are not supported; use polling", c.name)
}

func (c *ThirdPartyClient) SetToken(token string) {}

// SetCredentials captures the account's email address. It's called by
// mailme's command layer before every operation, so it doubles as how this
// stateless client learns which mailbox to look up.
func (c *ThirdPartyClient) SetCredentials(email, password string) { c.email = email }

func (c *ThirdPartyClient) OnTokenUpdate(fn func(token string)) {}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
