package kanboard

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateBaseURL parses a Kanboard instance URL: http/https only, host required, no userinfo.
func ValidateBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("KANBOARD_URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid KANBOARD_URL: %w", err)
	}
	if u.User != nil {
		return "", fmt.Errorf("KANBOARD_URL must not contain userinfo (credentials embedded in the URL are not allowed)")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("KANBOARD_URL scheme must be http or https, not %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("KANBOARD_URL must include a host")
	}
	u.User = nil
	u.Fragment = ""
	u.RawQuery = ""
	out := strings.TrimRight(u.String(), "/")
	return out, nil
}

// RPCURL returns the JSON-RPC endpoint URL for a validated base URL.
func RPCURL(base string) (string, error) {
	normalized, err := ValidateBaseURL(base)
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(normalized)
	if strings.HasSuffix(lower, "/jsonrpc.php") {
		return normalized, nil
	}
	return normalized + "/jsonrpc.php", nil
}

// StripURLUserinfo removes embedded credentials from URLs in text.
func StripURLUserinfo(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return redactUserinfoSubstring(s)
	}
	if u.User != nil {
		withUser := u.String()
		u.User = nil
		without := strings.TrimRight(u.String(), "/")
		out := strings.ReplaceAll(s, withUser, without)
		return redactUserinfoSubstring(out)
	}
	return redactUserinfoSubstring(s)
}

func redactUserinfoSubstring(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		rest := s[i+3:]
		if at := strings.Index(rest, "@"); at > 0 && strings.Contains(rest[:at], ":") {
			return s[:i+3] + rest[at+1:]
		}
	}
	return s
}
