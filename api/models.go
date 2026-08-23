package api

import "time"

// Account represents a temporary email account.
type Account struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Password  string     `json:"password"`
	Token     string     `json:"token,omitempty"`
	Provider  string     `json:"provider,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Domain represents an available email domain.
type Domain struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Message represents an email message summary.
type Message struct {
	ID        string     `json:"id"`
	Subject   string     `json:"subject"`
	FromAddr  string     `json:"from_addr"`
	Intro     string     `json:"intro"`
	Seen      bool       `json:"seen"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// MessageDetail represents a full email message.
type MessageDetail struct {
	ID          string       `json:"id"`
	Subject     string       `json:"subject"`
	FromAddr    string       `json:"from_addr"`
	To          string       `json:"to"`
	Cc          string       `json:"cc,omitempty"`
	Bcc         string       `json:"bcc,omitempty"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Intro       string       `json:"intro"`
	Seen        bool         `json:"seen"`
	CreatedAt   *time.Time   `json:"created_at,omitempty"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment represents an email attachment.
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
