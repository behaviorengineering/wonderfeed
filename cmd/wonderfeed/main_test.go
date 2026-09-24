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
	if !strings.Contains(stdout.String(), "backup create") {
		t.Fatalf("stdout missing backup: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "control serve") {
		t.Fatalf("stdout missing control: %q", stdout.String())
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

func TestRunControlHelp(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	code := run([]string{"wonderfeed", "control", "help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout.String(), "serve") || !strings.Contains(stdout.String(), "migrate up") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunControlServeHelp(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	code := run([]string{"wonderfeed", "control", "serve", "--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout.String(), "WONDERFEED_CONTROL_BIND") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
