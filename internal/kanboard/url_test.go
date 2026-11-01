package kanboard

import (
	"strings"
	"testing"
)

func TestValidateBaseURL(t *testing.T) {
	ok, err := ValidateBaseURL("https://kanboard.example.com/")
	if err != nil || ok != "https://kanboard.example.com" {
		t.Fatalf("ok=%q err=%v", ok, err)
	}
	for _, raw := range []string{
		"ftp://example.com",
		"https://",
		"https:///path",
		"https://user:pass@host.example",
		"://no-scheme",
	} {
		if _, err := ValidateBaseURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestRPCURL(t *testing.T) {
	u, err := RPCURL("https://kb.example.com")
	if err != nil || u != "https://kb.example.com/jsonrpc.php" {
		t.Fatalf("got %q err=%v", u, err)
	}
	u2, err := RPCURL("https://kb.example.com/jsonrpc.php")
	if err != nil || u2 != "https://kb.example.com/jsonrpc.php" {
		t.Fatalf("got %q err=%v", u2, err)
	}
}

func TestStripURLUserinfo(t *testing.T) {
	in := "https://user:pass@host.example/jsonrpc.php"
	out := StripURLUserinfo(in)
	if strings.Contains(out, "user") || strings.Contains(out, "pass") {
		t.Fatalf("userinfo leaked: %q", out)
	}
}

func TestSanitizeUsers(t *testing.T) {
	raw := []byte(`[{"id":1,"username":"admin","name":"Admin","email":"secret@example.com"}]`)
	out, err := SanitizeUsers(raw)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "secret@") || strings.Contains(s, "email") {
		t.Fatalf("email leaked: %s", s)
	}
	if !strings.Contains(s, `"username":"admin"`) {
		t.Fatalf("expected username: %s", s)
	}
}
