// Package kanboard is a minimal Kanboard JSON-RPC API client (community-built; not affiliated with Kanboard).
package kanboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
)

const MinTokenLength = 16

const AppAPIUsername = "jsonrpc"

type Client struct {
	RPCURL   string
	Username string
	Token    string
	HTTP     *http.Client
}

type APIError struct {
	Status  int
	Message string
	Body    json.RawMessage
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("kanboard HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("kanboard HTTP %d", e.Status)
}

func (e *APIError) IsAuth() bool { return e.Status == 401 || e.Status == 403 }

type RPCError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *RPCError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("JSON-RPC error %d", e.Code)
}

var rpcID atomic.Uint64

func nextRPCID() int64 {
	return int64(rand.Int31()) + int64(rpcID.Add(1))
}

func (c *Client) http() *http.Client {
	if c.HTTP == nil {
		return http.DefaultClient
	}
	return c.HTTP
}

func (c *Client) Redact(s string) string {
	s = StripURLUserinfo(s)
	if c.Token != "" && len(c.Token) >= MinTokenLength {
		s = strings.ReplaceAll(s, c.Token, "<API_TOKEN>")
	}
	if c.Username != "" && c.Username != AppAPIUsername && len(c.Username) >= 3 {
		s = strings.ReplaceAll(s, c.Username, "<USERNAME>")
	}
	// Redact Basic auth payload if present in text.
	if c.Token != "" && len(c.Token) >= MinTokenLength {
		user := c.Username
		if user == "" {
			user = AppAPIUsername
		}
		raw := user + ":" + c.Token
		enc := base64.StdEncoding.EncodeToString([]byte(raw))
		s = strings.ReplaceAll(s, enc, "<BASIC_AUTH>")
	}
	return s
}

func (c *Client) basicAuth() string {
	user := c.Username
	if user == "" {
		user = AppAPIUsername
	}
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + c.Token))
}

func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      nextRPCID(),
	}
	if params != nil {
		reqBody["params"] = params
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.RPCURL, bytes.NewReader(b))
	if err != nil {
		return nil, errors.New(c.Redact(err.Error()))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.basicAuth())

	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("JSON-RPC %s failed: %s", method, c.Redact(err.Error()))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %s", c.Redact(err.Error()))
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, &APIError{
			Status:  resp.StatusCode,
			Message: "authentication failed (check KANBOARD_API_TOKEN and KANBOARD_USERNAME)",
		}
	}
	if resp.StatusCode >= 400 {
		msg := c.Redact(parseHTTPMessage(data))
		return nil, &APIError{Status: resp.StatusCode, Message: msg, Body: data}
	}
	return parseRPCResult(data)
}

func parseHTTPMessage(data []byte) string {
	var m struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &m) == nil {
		if m.Message != "" {
			return m.Message
		}
		if m.Error != "" {
			return m.Error
		}
	}
	return strings.TrimSpace(string(data))
}

func parseRPCResult(data []byte) (json.RawMessage, error) {
	if len(data) == 0 {
		return json.RawMessage("null"), nil
	}
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '[' {
		return nil, errors.New("batch JSON-RPC responses are not supported")
	}
	var wrap struct {
		Result json.RawMessage
		Error  *struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC response: %w", err)
	}
	if wrap.Error != nil {
		return nil, &RPCError{
			Code:    wrap.Error.Code,
			Message: wrap.Error.Message,
			Data:    wrap.Error.Data,
		}
	}
	if len(wrap.Result) == 0 {
		return json.RawMessage("null"), nil
	}
	return wrap.Result, nil
}

func DryRun(method, rpcURL string, params any) map[string]any {
	safeURL := StripURLUserinfo(rpcURL)
	out := map[string]any{
		"method": method,
		"url":    safeURL,
		"header": map[string]string{"Authorization": "Basic <credentials>"},
	}
	if params != nil {
		out["params"] = params
	}
	return out
}

// SanitizeUsers returns id/username/name only — never email addresses.
func SanitizeUsers(raw json.RawMessage) (json.RawMessage, error) {
	var users []map[string]any
	if err := json.Unmarshal(raw, &users); err != nil {
		return raw, nil
	}
	out := make([]map[string]any, 0, len(users))
	allowed := map[string]bool{"id": true, "username": true, "name": true}
	for _, u := range users {
		safe := make(map[string]any, 3)
		for key, val := range u {
			if allowed[key] {
				safe[key] = val
			}
		}
		out = append(out, safe)
	}
	return json.Marshal(out)
}
