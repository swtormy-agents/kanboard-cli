package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const testToken = "kanboard-test-tk16" // 16 chars // gitleaks:allow

func testEnv(url string) func(string) string {
	return func(k string) string {
		switch k {
		case "KANBOARD_URL":
			return url
		case "KANBOARD_API_TOKEN":
			return testToken
		}
		return ""
	}
}

func testEnvUser(url string) func(string) string {
	return func(k string) string {
		switch k {
		case "KANBOARD_URL":
			return url
		case "KANBOARD_API_TOKEN":
			return testToken
		case "KANBOARD_USERNAME":
			return "agent"
		}
		return ""
	}
}

func checkBasicAuth(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Basic ") {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(h, "Basic "))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	return ok && pass == testToken && (user == "jsonrpc" || user == "agent")
}

type rpcReq struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

func newRPCServer(t *testing.T, handlers map[string]func(map[string]any) any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jsonrpc.php" {
			http.NotFound(w, r)
			return
		}
		if !checkBasicAuth(r) {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		var params map[string]any
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}
		h, ok := handlers[req.Method]
		if !ok {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"error":   map[string]any{"code": -32601, "message": "unknown method"},
			})
			return
		}
		result := h(params)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	}))
}

func runCLI(t *testing.T, getenv func(string) string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Execute(args, strings.NewReader(""), &out, &errOut, getenv)
	return code, out.String(), errOut.String()
}

func TestHealthAndVersion(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getVersion": func(map[string]any) any { return "1.2.40" },
	})
	defer srv.Close()
	code, out, _ := runCLI(t, testEnv(srv.URL), "health")
	if code != 0 || !strings.Contains(out, `"ok":true`) {
		t.Fatalf("health code=%d out=%s", code, out)
	}
	code, out, _ = runCLI(t, testEnv(srv.URL), "version")
	if code != 0 || !strings.Contains(out, "1.2.40") {
		t.Fatalf("version code=%d out=%s", code, out)
	}
}

func TestProjectsList(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getAllProjects": func(map[string]any) any {
			return []map[string]any{{"id": 1, "name": "demo"}}
		},
	})
	defer srv.Close()
	code, out, _ := runCLI(t, testEnv(srv.URL), "projects", "list")
	if code != 0 || !strings.Contains(out, "demo") {
		t.Fatalf("projects code=%d out=%s", code, out)
	}
}

func TestTasksListStatus(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getAllTasks": func(p map[string]any) any {
			if int(p["status_id"].(float64)) != 0 {
				t.Fatalf("status_id=%v", p["status_id"])
			}
			return []any{}
		},
	})
	defer srv.Close()
	code, _, _ := runCLI(t, testEnv(srv.URL), "tasks", "list", "--project", "1", "--status", "inactive")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestUsersListStripsEmail(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getAllUsers": func(map[string]any) any {
			return []map[string]any{{"id": 1, "username": "u", "name": "U", "email": "x@y.z"}}
		},
	})
	defer srv.Close()
	code, out, _ := runCLI(t, testEnv(srv.URL), "users", "list")
	if code != 0 || strings.Contains(out, "x@y.z") || strings.Contains(out, "email") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestSearch(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"searchTasks": func(p map[string]any) any {
			if p["query"] != "assignee:nobody" {
				t.Fatalf("query=%v", p["query"])
			}
			return []any{}
		},
	})
	defer srv.Close()
	code, _, _ := runCLI(t, testEnv(srv.URL), "search", "--project", "2", "--query", "assignee:nobody")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestMeRequiresUserAuth(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{})
	defer srv.Close()
	code, _, errOut := runCLI(t, testEnv(srv.URL), "me")
	if code != 2 {
		t.Fatalf("want exit 2 got %d: %s", code, errOut)
	}
}

func TestMeWithUserAuth(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getMe": func(map[string]any) any { return map[string]any{"id": 5, "username": "agent"} },
	})
	defer srv.Close()
	code, out, _ := runCLI(t, testEnvUser(srv.URL), "me")
	if code != 0 || !strings.Contains(out, "agent") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	code, _, errOut := runCLI(t, testEnv(srv.URL), "version")
	if code != 3 {
		t.Fatalf("want exit 3 got %d stderr=%s", code, errOut)
	}
	if strings.Contains(errOut, testToken) {
		t.Fatalf("token leaked: %s", errOut)
	}
}

func TestDryRunRedactsCredentials(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	code, out, _ := runCLI(t, testEnv(srv.URL), "health", "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out, testToken) {
		t.Fatalf("token leaked: %s", out)
	}
	if !strings.Contains(out, "<credentials>") {
		t.Fatalf("expected placeholder: %s", out)
	}
}

func TestRPCReadOnly(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getAllProjects": func(map[string]any) any { return []any{} },
	})
	defer srv.Close()
	code, _, _ := runCLI(t, testEnv(srv.URL), "rpc", "getAllProjects")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestRPCWriteRequiresWriteFlag(t *testing.T) {
	var errOut bytes.Buffer
	code := Execute([]string{"rpc", "createTask", "--params", `{"title":"x","project_id":1}`},
		strings.NewReader(""), io.Discard, &errOut, testEnv("http://example"))
	if code != 3 {
		t.Fatalf("want 3 got %d: %s", code, errOut.String())
	}
}

func TestRPCBatchParamsRejected(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{})
	defer srv.Close()
	code, _, errOut := runCLI(t, testEnv(srv.URL), "rpc", "getVersion", "--params", `[{"a":1}]`)
	if code != 2 {
		t.Fatalf("want 2 got %d: %s", code, errOut)
	}
}

func TestTaskCreateRequiresWrite(t *testing.T) {
	var errOut bytes.Buffer
	code := Execute([]string{"task", "create", "--project", "1", "--title", "x"},
		strings.NewReader(""), io.Discard, &errOut, testEnv("http://example"))
	if code != 3 {
		t.Fatalf("want 3 got %d: %s", code, errOut.String())
	}
}

func TestInvalidBaseURLs(t *testing.T) {
	cases := []struct {
		url  string
		code int
	}{
		{"ftp://example.com", 2},
		{"https://", 2},
		{"https://user:pass@host.example", 2},
	}
	for _, tc := range cases {
		getenv := func(k string) string {
			switch k {
			case "KANBOARD_URL":
				return tc.url
			case "KANBOARD_API_TOKEN":
				return testToken
			}
			return ""
		}
		var errOut bytes.Buffer
		code := Execute([]string{"version"}, strings.NewReader(""), io.Discard, &errOut, getenv)
		if code != tc.code {
			t.Fatalf("url %q want exit %d got %d: %s", tc.url, tc.code, code, errOut.String())
		}
	}
}

func TestInvalidIDFlags(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{})
	defer srv.Close()
	cases := [][]string{
		{"tasks", "list", "--project", "0"},
		{"tasks", "list", "--project", "-1"},
		{"tasks", "list", "--project", "1a"},
		{"tasks", "list", "--project", "1/2"},
		{"tasks", "list", "--project", "%2e"},
		{"tasks", "list", "--project", "18446744073709551616"},
		{"task", "get", "--id", "0"},
	}
	for _, args := range cases {
		code, _, errOut := runCLI(t, testEnv(srv.URL), args...)
		if code != 2 {
			t.Fatalf("args %v want exit 2 got %d stderr=%s", args, code, errOut)
		}
	}
}

func TestSkillPrint(t *testing.T) {
	code, out, _ := runCLI(t, testEnv("http://127.0.0.1:1"), "skill", "print")
	if code != 0 || !strings.Contains(out, "name: kanboard") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestSkillMarkdownIdentical(t *testing.T) {
	a, err := os.ReadFile("SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../skills/kanboard/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("SKILL.md files differ; run scripts/sync-skill-md.sh")
	}
	if string(a) != skillMD {
		t.Fatalf("embedded SKILL.md out of sync")
	}
}

func TestJSONRPCErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBasicAuth(r) {
			http.Error(w, "nope", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"error":   map[string]any{"code": -32603, "message": "boom with " + testToken},
		})
	}))
	defer srv.Close()
	code, _, errOut := runCLI(t, testEnv(srv.URL), "version")
	if code != 1 {
		t.Fatalf("want 1 got %d", code)
	}
	if strings.Contains(errOut, testToken) {
		t.Fatalf("token in rpc error: %s", errOut)
	}
	if !strings.Contains(errOut, "boom") {
		t.Fatalf("missing message: %s", errOut)
	}
}

func TestReportErrorRedactsToken(t *testing.T) {
	var errOut bytes.Buffer
	a := &app{token: testToken, errOut: &errOut}
	code := a.reportError(runtimeErr(fmt.Errorf("failure mentioning %s", testToken)))
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(errOut.String(), testToken) {
		t.Fatalf("token leaked")
	}
}

func TestShortTokenRejected(t *testing.T) {
	short := "short-token-15c" // 15 chars // gitleaks:allow
	getenv := func(k string) string {
		switch k {
		case "KANBOARD_URL":
			return "http://127.0.0.1:1"
		case "KANBOARD_API_TOKEN":
			return short
		}
		return ""
	}
	var errOut bytes.Buffer
	code := Execute([]string{"version"}, strings.NewReader(""), io.Discard, &errOut, getenv)
	if code != 2 {
		t.Fatalf("want exit 2 got %d: %s", code, errOut.String())
	}
}

func TestAppUsesJsonrpcUser(t *testing.T) {
	var seenUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(r.Header.Get("Authorization"), "Basic "))
		seenUser, _, _ = strings.Cut(string(raw), ":")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "ok"})
	}))
	defer srv.Close()
	runCLI(t, testEnv(srv.URL), "version")
	if seenUser != "jsonrpc" {
		t.Fatalf("user=%q want jsonrpc", seenUser)
	}
}

func TestCommentsList(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getAllComments": func(p map[string]any) any {
			if int(p["task_id"].(float64)) != 9 {
				t.Fatalf("task_id=%v", p["task_id"])
			}
			return []any{}
		},
	})
	defer srv.Close()
	code, _, _ := runCLI(t, testEnv(srv.URL), "comments", "list", "--task", "9")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestProjectGet(t *testing.T) {
	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"getProjectById": func(p map[string]any) any {
			return map[string]any{"id": p["project_id"], "name": "P"}
		},
	})
	defer srv.Close()
	code, out, _ := runCLI(t, testEnv(srv.URL), "project", "get", "--id", "3")
	if code != 0 || !strings.Contains(out, `"name":"P"`) {
		t.Fatalf("code=%d out=%s", code, out)
	}
}
