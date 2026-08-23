package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.mail.tm"

// MailtmClient is the mail.tm API client.
type MailtmClient struct {
	httpClient    *http.Client
	token         string
	email         string
	password      string
	onTokenUpdate func(token string) // callback to persist new token
}

// NewMailtmClient creates a new mail.tm API client.
func NewMailtmClient() *MailtmClient {
	return &MailtmClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken sets the JWT auth token.
func (c *MailtmClient) SetToken(token string) {
	c.token = token
}

// SetCredentials stores email/password for automatic token refresh.
func (c *MailtmClient) SetCredentials(email, password string) {
	c.email = email
	c.password = password
}

// OnTokenUpdate registers a callback for when the token is refreshed.
func (c *MailtmClient) OnTokenUpdate(fn func(token string)) {
	c.onTokenUpdate = fn
}

// Name returns the provider name.
func (c *MailtmClient) Name() string {
	return "mailtm"
}

func (c *MailtmClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	return c.doRequestWithRetry(method, path, body, true)
}

func (c *MailtmClient) doRequestWithRetry(method, path string, body interface{}, canRetry bool) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/ld+json, application/json")
	req.Header.Set("User-Agent", "mailme-cli/1.0")

	// Only set Content-Type for requests with a body
	if body != nil {
		if method == "PATCH" {
			req.Header.Set("Content-Type", "application/merge-patch+json")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Auto-refresh token on 401 if we have credentials
	if resp.StatusCode == 401 && canRetry && c.email != "" && c.password != "" {
		resp.Body.Close()
		if err := c.refreshToken(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		return c.doRequestWithRetry(method, path, body, false)
	}

	return resp, nil
}

func (c *MailtmClient) refreshToken() error {
	tok, err := c.getToken(c.email, c.password)
	if err != nil {
		return err
	}
	c.token = tok
	if c.onTokenUpdate != nil {
		c.onTokenUpdate(tok)
	}
	return nil
}

func (c *MailtmClient) getToken(email, password string) (string, error) {
	resp, err := c.doRequestWithRetry("POST", "/token", tokenRequest{
		Address:  email,
		Password: password,
	}, false)
	if err != nil {
		return "", err
	}
	var tok tokenResponse
	if err := parseJSON(resp, &tok); err != nil {
		return "", err
	}
	return tok.Token, nil
}

// --- Domains ---

type domainsResponse struct {
	Member []domainJSON `json:"hydra:member"`
}

type domainJSON struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	IsActive bool   `json:"isActive"`
}

// GetDomains returns available email domains.
func (c *MailtmClient) GetDomains() ([]Domain, error) {
	resp, err := c.doRequest("GET", "/domains?page=1", nil)
	if err != nil {
		return nil, err
	}

	var dr domainsResponse
	if err := parseJSON(resp, &dr); err != nil {
		return nil, err
	}

	var domains []Domain
	for _, d := range dr.Member {
		if d.IsActive {
			domains = append(domains, Domain{
				ID:     d.ID,
				Name:   d.Domain,
				Active: d.IsActive,
			})
		}
	}
	return domains, nil
}

// --- Account ---

type accountJSON struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	CreatedAt string `json:"createdAt"`
}

type createAccountRequest struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type tokenRequest struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// CreateAccount creates a new mail.tm account and returns it with a JWT token.
func (c *MailtmClient) CreateAccount(email, password string) (*Account, error) {
	// Create account
	resp, err := c.doRequest("POST", "/accounts", createAccountRequest{
		Address:  email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	var acc accountJSON
	if err := parseJSON(resp, &acc); err != nil {
		return nil, fmt.Errorf("parse account: %w", err)
	}

	// Get token
	resp, err = c.doRequest("POST", "/token", tokenRequest{
		Address:  email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}
	var tok tokenResponse
	if err := parseJSON(resp, &tok); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	c.token = tok.Token

	return &Account{
		ID:        acc.ID,
		Email:     email,
		Password:  password,
		Token:     tok.Token,
		CreatedAt: ParseTime(acc.CreatedAt),
	}, nil
}

// Authenticate authenticates with existing credentials.
func (c *MailtmClient) Authenticate(email, password string) (*Account, error) {
	resp, err := c.doRequest("POST", "/token", tokenRequest{
		Address:  email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	var tok tokenResponse
	if err := parseJSON(resp, &tok); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	c.token = tok.Token

	// Get account details
	resp, err = c.doRequest("GET", "/me", nil)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	var acc accountJSON
	if err := parseJSON(resp, &acc); err != nil {
		return nil, fmt.Errorf("parse account: %w", err)
	}

	return &Account{
		ID:        acc.ID,
		Email:     email,
		Password:  password,
		Token:     tok.Token,
		CreatedAt: ParseTime(acc.CreatedAt),
	}, nil
}

// --- Messages ---

type messagesResponse struct {
	Member []messageJSON `json:"hydra:member"`
}

type messageJSON struct {
	ID        string   `json:"id"`
	Subject   string   `json:"subject"`
	From      fromJSON `json:"from"`
	Intro     string   `json:"intro"`
	Seen      bool     `json:"seen"`
	CreatedAt string   `json:"createdAt"`
}

type fromJSON struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type messageDetailJSON struct {
	ID          string           `json:"id"`
	Subject     string           `json:"subject"`
	From        fromJSON         `json:"from"`
	To          []toJSON         `json:"to"`
	Cc          []toJSON         `json:"cc"`
	Bcc         []toJSON         `json:"bcc"`
	Text        string           `json:"text"`
	HTML        interface{}      `json:"html"`
	Intro       string           `json:"intro"`
	Seen        bool             `json:"seen"`
	CreatedAt   string           `json:"createdAt"`
	Attachments []attachmentJSON `json:"attachments"`
}

type toJSON struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type attachmentJSON struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// GetMessages returns messages for the given page (1-based).
func (c *MailtmClient) GetMessages(page int) ([]Message, error) {
	if page < 1 {
		page = 1
	}
	resp, err := c.doRequest("GET", fmt.Sprintf("/messages?page=%d", page), nil)
	if err != nil {
		return nil, err
	}

	var mr messagesResponse
	if err := parseJSON(resp, &mr); err != nil {
		return nil, err
	}

	var messages []Message
	for _, m := range mr.Member {
		messages = append(messages, Message{
			ID:        m.ID,
			Subject:   m.Subject,
			FromAddr:  m.From.Address,
			Intro:     m.Intro,
			Seen:      m.Seen,
			CreatedAt: ParseTime(m.CreatedAt),
		})
	}
	return messages, nil
}

// GetMessage returns full details of a specific message.
func (c *MailtmClient) GetMessage(messageID string) (*MessageDetail, error) {
	resp, err := c.doRequest("GET", "/messages/"+messageID, nil)
	if err != nil {
		return nil, err
	}

	var md messageDetailJSON
	if err := parseJSON(resp, &md); err != nil {
		return nil, err
	}

	var toAddrs []string
	for _, t := range md.To {
		toAddrs = append(toAddrs, t.Address)
	}
	var ccAddrs []string
	for _, cc := range md.Cc {
		ccAddrs = append(ccAddrs, cc.Address)
	}
	var bccAddrs []string
	for _, bcc := range md.Bcc {
		bccAddrs = append(bccAddrs, bcc.Address)
	}

	var attachments []Attachment
	for _, a := range md.Attachments {
		attachments = append(attachments, Attachment{
			ID:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
		})
	}

	return &MessageDetail{
		ID:          md.ID,
		Subject:     md.Subject,
		FromAddr:    md.From.Address,
		To:          joinStrings(toAddrs),
		Cc:          joinStrings(ccAddrs),
		Bcc:         joinStrings(bccAddrs),
		Text:        md.Text,
		HTML:        htmlToString(md.HTML),
		Intro:       md.Intro,
		Seen:        md.Seen,
		CreatedAt:   ParseTime(md.CreatedAt),
		Attachments: attachments,
	}, nil
}

type updateMessageRequest struct {
	Seen bool `json:"seen"`
}

// SetSeen updates the seen status of a message.
func (c *MailtmClient) SetSeen(messageID string, seen bool) error {
	resp, err := c.doRequest("PATCH", "/messages/"+messageID, updateMessageRequest{
		Seen: seen,
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("set seen failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

type sourceJSON struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

// GetSource returns the raw RFC 822 source text of a message.
func (c *MailtmClient) GetSource(messageID string) (string, error) {
	resp, err := c.doRequest("GET", "/sources/"+messageID, nil)
	if err != nil {
		return "", err
	}
	var src sourceJSON
	if err := parseJSON(resp, &src); err != nil {
		return "", err
	}
	return src.Data, nil
}

// DownloadMessageEML downloads the full .eml raw data for a message.
func (c *MailtmClient) DownloadMessageEML(messageID string) ([]byte, error) {
	// Try /messages/{id}/download first
	resp, err := c.doRequest("GET", fmt.Sprintf("/messages/%s/download", messageID), nil)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Fallback to /sources/{id}
	src, err := c.GetSource(messageID)
	if err == nil && src != "" {
		return []byte(src), nil
	}
	return nil, fmt.Errorf("could not download message EML")
}

// DeleteMessage deletes a specific message.
func (c *MailtmClient) DeleteMessage(messageID string) error {
	resp, err := c.doRequest("DELETE", "/messages/"+messageID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("delete failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// DeleteAccount deletes the current account.
func (c *MailtmClient) DeleteAccount(accountID string) error {
	resp, err := c.doRequest("DELETE", "/accounts/"+accountID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("delete failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// DownloadAttachment downloads an attachment by ID.
func (c *MailtmClient) DownloadAttachment(messageID, attachmentID string) ([]byte, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/messages/%s/attachment/%s", messageID, attachmentID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ListenMessagesSSE connects to the Mercure SSE stream for real-time inbox events.
func (c *MailtmClient) ListenMessagesSSE(ctx context.Context, accountID string, onMessage func(msg *Message)) error {
	mercureURL := fmt.Sprintf("https://mercure.mail.tm/.well-known/mercure?topic=/accounts/%s", accountID)
	req, err := http.NewRequestWithContext(ctx, "GET", mercureURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mercure SSE connection failed: HTTP %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			var m messageDetailJSON
			if err := json.Unmarshal([]byte(data), &m); err == nil && m.ID != "" {
				msg := &Message{
					ID:        m.ID,
					Subject:   m.Subject,
					FromAddr:  m.From.Address,
					Intro:     m.Intro,
					Seen:      m.Seen,
					CreatedAt: ParseTime(m.CreatedAt),
				}
				onMessage(msg)
			}
		}
	}
}
