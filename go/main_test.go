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
	tests := []struct {
		name    string
		in      map[string]string
		want    string
		wantLen int
	}{
		{name: "check signing result", in: map[string]string{"a": "1", "b": "hello world"}, want: "63b637767acc479ec77537abe63e1597", wantLen: 3},
		// sign=ef2fdbd577d7da48847f69b02e95affb&fid=123456789&cdk=GuDokYTKOR&kid=2402&time=1789316476
		{name: "check signing result", in: map[string]string{"time": "1789316476", "kid": "2402", "fid": "123456789", "cdk": "GuDokYTKOR"}, want: "ef2fdbd577d7da48847f69b02e95affb", wantLen: 5},
	}

	for _, tt := range tests {
		out := appendSign(tt.in)
		for k, v := range tt.in {
			if out[k] != v {
				t.Fatalf("appendSign did not keep original value for %q: got %q, want %q", k, out[k], v)
			}
		}
		if got, want := out["sign"], tt.want; got != want {
			t.Fatalf("unexpected sign: got %q, want %q", got, want)
		}
		if len(out) != tt.wantLen {
			t.Fatalf("unexpected output size: got %d, want 3", len(out))
		}
	}

}

func TestDefaultFIDFromDotEnv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantKid string
	}{
		{name: "plain", content: "DEFAULT_FID=123456\n", want: "123456"},
		{name: "export_and_spaces", content: "export DEFAULT_FID =  987654\n", want: "987654"},
		{name: "quoted", content: "DEFAULT_FID=\"  456789  \"\n", want: "456789"},
		{name: "single_quoted", content: "# comment\nFOO=bar\nDEFAULT_FID='abc123'\n", want: "abc123"},
		{name: "missing", content: "FOO=bar\n", want: ""},
		{name: "kid_plain", content: "DEFAULT_KID=123456\n", wantKid: "123456"},
		{name: "kid_export_and_spaces", content: "export DEFAULT_KID =  987654\n", wantKid: "987654"},
		{name: "kid_quoted", content: "DEFAULT_KID=\"  456789  \"\n", wantKid: "456789"},
		{name: "kid_single_quoted", content: "# comment\nFOO=bar\nDEFAULT_KID='abc123'\n", wantKid: "abc123"},
		{name: "kid_missing", content: "FOO=bar\n", want: ""},
		{name: "all_single_quoted", content: "# comment\nFOO=bar\nDEFAULT_FID='abc123'\nDEFAULT_KID='kidabc123'\n", want: "abc123", wantKid: "kidabc123"},
		{name: "all_missing", content: "FOO=bar\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".env")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write .env: %v", err)
			}

			got, gotKid, err := defaultsFromDotEnv(path)
			if err != nil {
				t.Fatalf("defaultsFromDotEnv returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("defaultsFromDotEnv() = %q, want %q", got, tt.want)
			}
			if gotKid != tt.wantKid {
				t.Fatalf("defaultsFromDotEnv() = %q, want %q", got, tt.wantKid)
			}
		})
	}
}

func TestDefaultFIDFromDotEnvMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	got, gotKid, err := defaultsFromDotEnv(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("expected no exist error for missing file, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty value for missing file, got %q", got)
	}
	if gotKid != "" {
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
	if err := os.WriteFile(dotEnvPath, []byte("DEFAULT_FID=111\nDEFAULT_KID=kid111\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	tests := []struct {
		name    string
		cli     string
		cliKid  string
		env     map[string]string
		want    string
		wantKid string
	}{
		{name: "dotenv_used_when_no_env_or_cli", cli: "", cliKid: "", env: map[string]string{}, want: "111", wantKid: "kid111"},
		{name: "env_overrides_dotenv", cli: "", cliKid: "", env: map[string]string{"DEFAULT_FID": "222", "DEFAULT_KID": "kid222"}, want: "222", wantKid: "kid222"},
		{name: "cli_overrides_env_and_dotenv", cli: "333", cliKid: "kid333", env: map[string]string{"DEFAULT_FID": "222", "DEFAULT_KID": "kid222"}, want: "333", wantKid: "kid333"},
		{name: "empty_cli_does_not_override", cli: "", cliKid: "", env: map[string]string{"DEFAULT_FID": "222", "DEFAULT_KID": "kid222"}, want: "222", wantKid: "kid222"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotKid, err := resolveIDs(tt.cli, tt.cliKid, envMap(tt.env), dotEnvPath)
			if err != nil {
				t.Fatalf("resolveFID returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveIDs() = %q, want %q", got, tt.want)
			}
			if gotKid != tt.wantKid {
				t.Fatalf("resolveIDs() = %q, wantKid %q", got, tt.wantKid)
			}
		})
	}
}

func TestResolveFIDDotEnvReadErrorFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	dir := t.TempDir()

	got, gotKid, err := resolveIDs("", "", envMap(map[string]string{"DEFAULT_FID": "222", "DEFAULT_KID": "kid222"}), dir)
	if err == nil {
		t.Fatalf("expected .env read error when path is a directory")
	}
	if got != "222" {
		t.Fatalf("resolveFID fallback with env failed: got %q, want %q", got, "222")
	}
	if gotKid != "kid222" {
		t.Fatalf("resolveKID fallback with env failed: got %q, want %q", got, "kid222")
	}

	got, gotKid, err = resolveIDs("", "", envMap(map[string]string{}), dir)
	if err == nil {
		t.Fatalf("expected .env read error when path is a directory")
	}
	if got != defaultFID {
		t.Fatalf("resolveFID fallback default failed: got %q, want %q", got, defaultFID)
	}
	if gotKid != defaultKID {
		t.Fatalf("resolveFID fallback default failed: got %q, want %q", got, defaultFID)
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantFID   string
		wantKID   string
		wantCode  string
		wantFile  string
		shouldErr bool
	}{
		{name: "code_only", args: []string{"HAPPY2026"}, wantFID: "", wantKID: "", wantCode: "HAPPY2026", wantFile: ""},
		{name: "with_user", args: []string{"-u", "123456", "HAPPY2026"}, wantFID: "123456", wantKID: "", wantCode: "HAPPY2026", wantFile: ""},
		{name: "trim_user", args: []string{"-u", " 123456 ", "HAPPY2026"}, wantFID: "123456", wantKID: "", wantCode: "HAPPY2026", wantFile: ""},
		{name: "with_file", args: []string{"-u", "123456", "-f", "codes.json"}, wantFID: "123456", wantKID: "", wantCode: "", wantFile: "codes.json"},
		{name: "file_and_code", args: []string{"-u", "123456", "-f", "codes.json", "HAPPY2026"}, shouldErr: true},
		{name: "no_code_or_file", args: []string{"-u", "123456"}, wantFID: "123456", wantKID: "", wantCode: "", wantFile: ""},
		{name: "too_many_positional", args: []string{"CODE1", "CODE2"}, shouldErr: true},
		{name: "unknown_flag", args: []string{"-x", "123", "CODE"}, shouldErr: true},
		{name: "kid_with_user", args: []string{"-k", "123456", "HAPPY2026"}, wantFID: "", wantKID: "123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "kid_trim_user", args: []string{"-k", " 123456 ", "HAPPY2026"}, wantFID: "", wantKID: "123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "kid_with_file", args: []string{"-k", "123456", "-f", "codes.json"}, wantFID: "", wantKID: "123456", wantCode: "", wantFile: "codes.json"},
		{name: "kid_no_code_or_file", args: []string{"-k", "123456"}, wantFID: "", wantKID: "123456", wantCode: "", wantFile: ""},
		{name: "uid_kid_with_user", args: []string{"-u", "123456", "-k", "kid123456", "HAPPY2026"}, wantFID: "123456", wantKID: "kid123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "uid_kid_trim_user", args: []string{"-u", "123456", "-k", " kid123456 ", "HAPPY2026"}, wantFID: "123456", wantKID: "kid123456", wantCode: "HAPPY2026", wantFile: ""},
		{name: "uid_kid_with_file", args: []string{"-u", "123456", "-k", "kid123456", "-f", "codes.json"}, wantFID: "123456", wantKID: "kid123456", wantCode: "", wantFile: "codes.json"},
		{name: "uid_kid_no_code_or_file", args: []string{"-u", "123456", "-k", "kid123456"}, wantFID: "123456", wantKID: "kid123456", wantCode: "", wantFile: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fid, kid, code, file, err := parseArgs(tt.args)
			if tt.shouldErr {
				if err == nil {
					t.Fatalf("parseArgs expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseArgs returned error: %v", err)
			}
			if fid != tt.wantFID || kid != tt.wantKID || code != tt.wantCode || file != tt.wantFile {
				t.Fatalf("parseArgs() = (%q, %q, %q, %q), want (%q, %q, %q, %q)", fid, kid, code, file, tt.wantFID, tt.wantKID, tt.wantCode, tt.wantFile)
			}
		})
	}
}
