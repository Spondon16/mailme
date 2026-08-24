package api

import (
	"context"
	"strings"
	"testing"
)

func TestNewThirdPartyClientSupportedProvider(t *testing.T) {
	for _, name := range []string{"tempmail.plus", "tempmailc", "mailnesia"} {
		c, err := NewThirdPartyClient(name)
		if err != nil {
			t.Fatalf("NewThirdPartyClient(%q) error = %v", name, err)
		}
		if c.Name() != name {
			t.Fatalf("Name() = %q, want %q", c.Name(), name)
		}
	}
}

func TestNewThirdPartyClientUnsupportedProvider(t *testing.T) {
	if _, err := NewThirdPartyClient("guerrillamail"); err == nil {
		t.Fatal("NewThirdPartyClient(\"guerrillamail\") error = nil, want error")
	}
}

func TestThirdPartyClientGetDomainsUnsupported(t *testing.T) {
	c, _ := NewThirdPartyClient("tempmailc")
	if _, err := c.GetDomains(); err == nil {
		t.Fatal("GetDomains() error = nil, want error")
	}
}

func TestThirdPartyClientSetCredentialsSetsEmail(t *testing.T) {
	c, _ := NewThirdPartyClient("tempmailc")
	c.SetCredentials("user@example.com", "ignored")
	if c.email != "user@example.com" {
		t.Fatalf("email = %q, want %q", c.email, "user@example.com")
	}
}

func TestThirdPartyClientGetMessageNoActiveAddress(t *testing.T) {
	c, _ := NewThirdPartyClient("tempmailc")
	if _, err := c.GetMessage("some-id"); err == nil {
		t.Fatal("GetMessage() error = nil, want error when no address is set")
	}
}

func TestThirdPartyClientUnsupportedOperations(t *testing.T) {
	c, _ := NewThirdPartyClient("tempmailc")

	if err := c.SetSeen("id", true); err != nil {
		t.Fatalf("SetSeen() error = %v, want nil (no-op)", err)
	}
	if _, err := c.GetSource("id"); err == nil {
		t.Fatal("GetSource() error = nil, want error")
	}
	if _, err := c.DownloadMessageEML("id"); err == nil {
		t.Fatal("DownloadMessageEML() error = nil, want error")
	}
	if err := c.DeleteMessage("id"); err == nil {
		t.Fatal("DeleteMessage() error = nil, want error")
	}
	if _, err := c.DownloadAttachment("id", "att"); err == nil {
		t.Fatal("DownloadAttachment() error = nil, want error")
	}
	if err := c.ListenMessagesSSE(context.Background(), "acc", nil); err == nil {
		t.Fatal("ListenMessagesSSE() error = nil, want error")
	}
}

func TestStripTags(t *testing.T) {
	html := "<html><body><p>Hello <b>world</b></p></body></html>"
	got := stripTags(html)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Fatalf("stripTags() = %q, still contains tags", got)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "world") {
		t.Fatalf("stripTags() = %q, missing expected text", got)
	}
}

func TestStripTagsCollapsesBlankLines(t *testing.T) {
	html := "one\n\n\n\n\ntwo"
	got := stripTags(html)
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("stripTags() = %q, want blank lines collapsed", got)
	}
}
