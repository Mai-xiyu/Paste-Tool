package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPasteDryRunArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"paste", "--source", "arg", "--text", "a\r\nb", "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "a\nb" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Paste Tool") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
