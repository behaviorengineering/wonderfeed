package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/secret"
)

var envPlaceholderRE = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandConfigPlaceholders(cfg *FileConfig) error {
	if cfg == nil {
		return nil
	}
	expandAllStrings(reflect.ValueOf(cfg))
	if unresolved := collectUnresolvedEnvPlaceholders(reflect.ValueOf(cfg), "config"); len(unresolved) > 0 {
		return apperr.New(apperr.CodeInvalid, "config.expand", "unresolved env placeholders in config").
			With("fields", strings.Join(unresolved, ", "))
	}
	return nil
}

func expandPlaceholderString(s string) string {
	return envPlaceholderRE.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1]
		v, err := secret.Resolve(varName)
		if err != nil {
			if secret.IsNotFound(err) {
				return ""
			}
			return match
		}
		return strings.TrimSpace(v)
	})
}

func expandAllStrings(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		expandAllStrings(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			expandAllStrings(v.Field(i))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.TrimSpace(expandPlaceholderString(v.String())))
		}
	}
}

func collectUnresolvedEnvPlaceholders(v reflect.Value, path string) []string {
	if !v.IsValid() {
		return nil
	}
	var out []string
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return collectUnresolvedEnvPlaceholders(v.Elem(), path)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			fieldPath := path + "." + t.Field(i).Name
			out = append(out, collectUnresolvedEnvPlaceholders(v.Field(i), fieldPath)...)
		}
	case reflect.String:
		s := v.String()
		if strings.Contains(s, "${") {
			out = append(out, fmt.Sprintf("%s=%q", path, s))
		}
	}
	return out
}
