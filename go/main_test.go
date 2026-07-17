package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func envMap(values map[string]string) func(string) string {
	return func(k string) string {
		return values[k]
	}
}

func TestEncodeURIComponent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "unreserved", in: "abcXYZ-_.!~*'()", want: "abcXYZ-_.!~*'()"},
		{name: "space", in: "hello world", want: "hello%20world"},
		{name: "reserved", in: "f+g/h?", want: "f%2Bg%2Fh%3F"},
		{name: "utf8", in: "\u00e4", want: "%C3%A4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeURIComponent(tt.in); got != tt.want {
				t.Fatalf("encodeURIComponent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAppendSign(t *testing.T) {
	in := map[string]string{
		"a": "1",
		"b": "hello world",
	}

	out := appendSign(in)

	if out["a"] != "1" || out["b"] != "hello world" {
		t.Fatalf("appendSign did not keep original values: %#v", out)
	}
	if got, want := out["sign"], "63b637767acc479ec77537abe63e1597"; got != want {
		t.Fatalf("unexpected sign: got %q, want %q", got, want)
	}
	if len(out) != 3 {
		t.Fatalf("unexpected output size: got %d, want 3", len(out))
	}
	if _, ok := in["sign"]; ok {
		t.Fatalf("input map must not be modified")
	}
}

func TestDefaultFIDFromDotEnv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "plain", content: "DEFAULT_FID=123456\n", want: "123456"},
		{name: "export_and_spaces", content: "export DEFAULT_FID =  987654\n", want: "987654"},
		{name: "quoted", content: "DEFAULT_FID=\"  456789  \"\n", want: "456789"},
		{name: "single_quoted", content: "# comment\nFOO=bar\nDEFAULT_FID='abc123'\n", want: "abc123"},
		{name: "missing", content: "FOO=bar\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".env")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write .env: %v", err)
			}

			got, err := defaultFIDFromDotEnv(path)
			if err != nil {
				t.Fatalf("defaultFIDFromDotEnv returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("defaultFIDFromDotEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultFIDFromDotEnvMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	got, err := defaultFIDFromDotEnv(path)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty value for missing file, got %q", got)
	}
}

func TestNow(t *testing.T) {
	n := now()
	parsed, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		t.Fatalf("now() is not a base10 int: %q (%v)", n, err)
	}
	if parsed <= 0 {
		t.Fatalf("now() must be positive, got %d", parsed)
	}
}

func TestResolveFIDPrecedence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	dir := t.TempDir()
	dotEnvPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(dotEnvPath, []byte("DEFAULT_FID=111\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	tests := []struct {
		name string
		cli string
		env  map[string]string
		want string
	}{
		{name: "dotenv_used_when_no_env_or_cli", cli: "", env: map[string]string{}, want: "111"},
		{name: "env_overrides_dotenv", cli: "", env: map[string]string{"DEFAULT_FID": "222"}, want: "222"},
		{name: "cli_overrides_env_and_dotenv", cli: "333", env: map[string]string{"DEFAULT_FID": "222"}, want: "333"},
		{name: "empty_cli_does_not_override", cli: "", env: map[string]string{"DEFAULT_FID": "222"}, want: "222"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFID(tt.cli, envMap(tt.env), dotEnvPath)
			if err != nil {
				t.Fatalf("resolveFID returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveFID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveFIDDotEnvReadErrorFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	dir := t.TempDir()

	got, err := resolveFID("", envMap(map[string]string{"DEFAULT_FID": "222"}), dir)
	if err == nil {
		t.Fatalf("expected .env read error when path is a directory")
	}
	if got != "222" {
		t.Fatalf("resolveFID fallback with env failed: got %q, want %q", got, "222")
	}

	got, err = resolveFID("", envMap(map[string]string{}), dir)
	if err == nil {
		t.Fatalf("expected .env read error when path is a directory")
	}
	if got != defaultFID {
		t.Fatalf("resolveFID fallback default failed: got %q, want %q", got, defaultFID)
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantFID   string
		wantCode  string
		wantFile  string
		shouldErr bool
	}{
		{name: "code_only", args: []string{"HAPPY2026"}, wantFID: "", wantCode: "HAPPY2026", wantFile: ""},
		{name: "with_user", args: []string{"-u", "123456", "HAPPY2026"}, wantFID: "123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "trim_user", args: []string{"-u", " 123456 ", "HAPPY2026"}, wantFID: "123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "with_file", args: []string{"-u", "123456", "-f", "codes.json"}, wantFID: "123456", wantCode: "", wantFile: "codes.json"},
		{name: "file_and_code", args: []string{"-u", "123456", "-f", "codes.json", "HAPPY2026"}, shouldErr: true},
		{name: "no_code_or_file", args: []string{"-u", "123456"}, wantFID: "123456", wantCode: "", wantFile: ""},
		{name: "too_many_positional", args: []string{"CODE1", "CODE2"}, shouldErr: true},
		{name: "unknown_flag", args: []string{"-x", "123", "CODE"}, shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fid, code, file, err := parseArgs(tt.args)
			if tt.shouldErr {
				if err == nil {
					t.Fatalf("parseArgs expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseArgs returned error: %v", err)
			}
			if fid != tt.wantFID || code != tt.wantCode || file != tt.wantFile {
				t.Fatalf("parseArgs() = (%q, %q, %q), want (%q, %q, %q)", fid, code, file, tt.wantFID, tt.wantCode, tt.wantFile)
			}
		})
	}
}

