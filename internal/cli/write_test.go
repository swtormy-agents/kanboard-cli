package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestWriteConfirmationSuccess(t *testing.T) {
	old := isTerminal
	isTerminal = func(ioReader) bool { return true }
	defer func() { isTerminal = old }()

	srv := newRPCServer(t, map[string]func(map[string]any) any{
		"closeTask": func(p map[string]any) any { return true },
	})
	defer srv.Close()
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute(
		[]string{"task", "close", "--write", "--id", "5"},
		strings.NewReader("write\n"),
		&out,
		&errOut,
		testEnv(srv.URL),
	)
	if code != 0 {
		t.Fatalf("exit %d out=%s err=%s", code, out.String(), errOut.String())
	}
}

func TestWriteConfirmationDenied(t *testing.T) {
	var errOut bytes.Buffer
	code := Execute(
		[]string{"task", "close", "--write", "--id", "5"},
		strings.NewReader("no\n"),
		io.Discard,
		&errOut,
		testEnv("http://example"),
	)
	if code != 3 {
		t.Fatalf("want 3 got %d: %s", code, errOut.String())
	}
}
