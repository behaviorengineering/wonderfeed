package ytzero

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// StatusReport is operator-facing provider status.
type StatusReport struct {
	Root           string
	ComposeFile    string
	DataDir        string
	ProviderDir    string
	ProviderHEAD   string
	ProviderDesc   string
	BaseURL        string
	ComposePS      string
	ComposeErr     string
	EnvFilePresent bool
}

// CollectStatus gathers pin and compose state for the operator.
func CollectStatus(ctx context.Context, paths Paths, baseURL string) (StatusReport, error) {
	const op = "ytzero.CollectStatus"
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	report := StatusReport{
		Root:           paths.Root,
		ComposeFile:    paths.ComposeFile,
		DataDir:        paths.DataDir,
		ProviderDir:    paths.ProviderDir,
		BaseURL:        baseURL,
		EnvFilePresent: fileExists(paths.EnvFile),
	}
	head, desc, err := providerPin(ctx, paths.ProviderDir)
	if err != nil {
		return report, apperr.Wrap(err, apperr.CodeFailed, op, "read provider pin")
	}
	report.ProviderHEAD = head
	report.ProviderDesc = desc

	ps, psErr := ComposePS(ctx, paths)
	if psErr != nil {
		report.ComposeErr = psErr.Error()
	} else {
		report.ComposePS = strings.TrimSpace(ps)
	}
	return report, nil
}

var statusTmpl = template.Must(template.New("status").Parse(`Provider: YT Zero
Root: {{.Root}}
Submodule: {{.ProviderDir}}
  HEAD: {{.ProviderHEAD}}
  Describe: {{.ProviderDesc}}
Compose: {{.ComposeFile}}
Data: {{.DataDir}}
Env file: {{if .EnvFilePresent}}present{{else}}missing (will copy from .env.example on up){{end}}
Base URL: {{.BaseURL}}
{{if .ComposeErr}}Compose ps: unavailable ({{.ComposeErr}})
{{else}}Compose ps:
{{.ComposePS}}
{{end}}`))

// FormatStatus renders StatusReport for the CLI.
func FormatStatus(report StatusReport) (string, error) {
	const op = "ytzero.FormatStatus"
	var buf bytes.Buffer
	if err := statusTmpl.Execute(&buf, report); err != nil {
		return "", apperr.Wrap(err, apperr.CodeFailed, op, "execute status template")
	}
	return buf.String(), nil
}

func providerPin(ctx context.Context, providerDir string) (head, desc string, err error) {
	if _, err := os.Stat(providerDir); err != nil {
		return "", "", fmt.Errorf("provider dir: %w", err)
	}
	headOut, err := gitOutput(ctx, providerDir, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	descOut, err := gitOutput(ctx, providerDir, "describe", "--tags", "--always")
	if err != nil {
		descOut = "unknown"
	}
	return strings.TrimSpace(headOut), strings.TrimSpace(descOut), nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %v: %w", args, err)
	}
	return string(out), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
