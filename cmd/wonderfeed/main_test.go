package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunBareInvoke(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	code := run([]string{"wonderfeed"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage: wonderfeed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	code := run([]string{"wonderfeed", "version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout.String(), "wonderfeed") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	code := run([]string{"wonderfeed", "help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout.String(), "provider up") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunUnknown(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	code := run([]string{"wonderfeed", "nope"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
}
