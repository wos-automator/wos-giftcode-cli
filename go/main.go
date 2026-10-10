// WOS Gift-Code Redemption Client (reverse engineered) - Go implementation
// Usage: go run . [-u userid] [-k stateid] <code>
//
//	go run . [-u userid] [-k stateid] -f <codes.json>
package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	salt       = "tB87#kPtkxqOS2"
	baseURL    = "https://wos-giftcode-api.centurygame.com/api"
	defaultFID = "123456789" // the playerid
	defaultKID = "1234"      // the state
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/126.0 Safari/537.36"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

// StateMap stores the redeemed codes list for each player ID (fid).
type StateMap map[string][]string

// encodeURIComponent mirrors JavaScript's encodeURIComponent so that the
// signature matches the browser client byte-for-byte.
func encodeURIComponent(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"
	var b strings.Builder
	for _, r := range []byte(s) {
		if strings.IndexByte(unreserved, r) >= 0 {
			b.WriteByte(r)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", r))
		}
	}
	return b.String()
}

// appendSign replicates appendSign() from request.js:
// sort keys, URL-encode the values, join with '&', append the salt, MD5.
func appendSign(data map[string]string) map[string]string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+encodeURIComponent(data[k]))
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + salt))

	signed := map[string]string{"sign": fmt.Sprintf("%x", sum)}
	for k, v := range data {
		signed[k] = v
	}
	return signed
}

// api POSTs a signed, form-encoded request and decodes the JSON response.
func api(path string, data map[string]string) (map[string]any, error) {
	form := url.Values{}
	for k, v := range appendSign(data) {
		form.Set(k, v)
	}

	log.Printf("api request: %s %s", path, form.Encode())
	req, err := http.NewRequest(http.MethodPost, baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://wos-giftcode.centurygame.com")
	req.Header.Set("Referer", "https://wos-giftcode.centurygame.com/")
	req.Header.Set("User-Agent", userAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(body))
	}
	return out, nil
}

func now() string { return strconv.FormatInt(time.Now().Unix(), 10) }

func redeem(fid, cdk, kid string) (map[string]any, error) {
	return api("/gift_code", map[string]string{
		"fid": fid, "cdk": cdk, "kid": kid, "time": now(),
	})
}

// prompt reads a trimmed line from stdin.
func prompt(reader *bufio.Reader, question string) string {
	fmt.Print(question)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// prettyJSON pretty-prints a decoded JSON map.
func prettyJSON(v map[string]any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// defaultsFromDotEnv reads
// DEFAULT_FID, DEFAULT_KID from a local .env file.
func defaultsFromDotEnv(path string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	fid := ""
	kid := ""

	for _, rawLine := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		k, v, ok := strings.Cut(line, "=")
		if !ok || !strings.HasPrefix(strings.TrimSpace(k), "DEFAULT_") {
			continue
		}

		if strings.TrimSpace(k) == "DEFAULT_FID" {
			fid = strings.Trim(v, `'" \t`)
		}
		if strings.TrimSpace(k) == "DEFAULT_KID" {
			kid = strings.Trim(v, `'" \t`)
		}

	}
	return fid, kid, nil
}

// resolveFID applies precedence: CLI arg > env var > .env > built-in default.
func resolveIDs(cliFID string, cliKID string, getenv func(string) string, dotEnvPath string) (string, string, error) {
	fid := defaultFID
	kid := defaultKID
	var readErr error

	// 1. Check global ~/.wos-cli-env config (lowest priority config)
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".wos-cli-env")
		if globalFID, globalKID, err := defaultsFromDotEnv(globalPath); err == nil {
			fid = globalFID
			kid = globalKID
		}
	}

	// 2. Check local .env (higher priority than global)
	dotEnvFID, dotEnvKID, err := defaultsFromDotEnv(dotEnvPath)
	if err != nil {
		if !os.IsNotExist(err) {
			readErr = err
		}
	} else {
		if dotEnvFID != "" {
			fid = dotEnvFID
		}
		if dotEnvKID != "" {
			kid = dotEnvKID
		}
	}

	// 3. Check environment variable (higher priority than local .env)
	if envFID := strings.TrimSpace(getenv("DEFAULT_FID")); envFID != "" {
		fid = envFID
	}
	if envKID := strings.TrimSpace(getenv("DEFAULT_KID")); envKID != "" {
		kid = envKID
	}

	// 4. Check CLI flag (highest priority)
	if cliFID = strings.TrimSpace(cliFID); cliFID != "" {
		fid = cliFID
	}
	if cliKID = strings.TrimSpace(cliKID); cliKID != "" {
		kid = cliKID
	}

	return fid, kid, readErr
}

// parseArgs parses the flags -u and -f, and returns positional args if any.
func parseArgs(args []string) (string, string, string, string, error) {
	fs := flag.NewFlagSet("wos-giftcode-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	userID := fs.String("u", "", "user id")
	stateID := fs.String("k", "", "state id")
	codesFile := fs.String("f", "", "path to JSON codes file (use '-' for stdin)")

	if err := fs.Parse(args); err != nil {
		return "", "", "", "", fmt.Errorf("usage:\n  wos-giftcode-cli [-u userid] [-k state] <code>\n  wos-giftcode-cli [-u userid] -f <codes.json>")
	}

	rest := fs.Args()
	if *codesFile != "" {
		if len(rest) > 0 {
			return "", "", "", "", fmt.Errorf("cannot specify both -f and a direct giftcode")
		}
		return strings.TrimSpace(*userID), strings.TrimSpace(*stateID), "", *codesFile, nil
	}

	code := ""
	if len(rest) == 1 {
		code = strings.TrimSpace(rest[0])
	} else if len(rest) > 1 {
		return "", "", "", "", fmt.Errorf("too many arguments\nusage:\n  wos-giftcode-cli [-u userid] [-k stateid] <code>\n  wos-giftcode-cli [-u userid] [-k stateid] -f <codes.json>")
	}

	return strings.TrimSpace(*userID), strings.TrimSpace(*stateID), code, "", nil
}

// --- State Cache Management ---
func stateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wos-redeemed-codes.json"), nil
}

func loadState(path string) StateMap {
	b, err := os.ReadFile(path)
	if err != nil {
		return make(StateMap)
	}
	var m StateMap
	if err := json.Unmarshal(b, &m); err != nil {
		return make(StateMap)
	}
	return m
}

func saveState(path string, m StateMap) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o644)
}

func isRedeemed(m StateMap, fid, code string) bool {
	codes, ok := m[fid]
	if !ok {
		return false
	}
	for _, c := range codes {
		if strings.EqualFold(c, code) {
			return true
		}
	}
	return false
}

func markRedeemed(m StateMap, fid, code string) {
	if isRedeemed(m, fid, code) {
		return
	}
	m[fid] = append(m[fid], code)
}

// parseCodesJson parses codes from various JSON formats.
func parseCodesJson(data []byte) ([]string, error) {
	type CodeObj struct {
		Code string `json:"code"`
	}

	// Array of objects
	var items []CodeObj
	if err := json.Unmarshal(data, &items); err == nil && len(items) > 0 && items[0].Code != "" {
		var codes []string
		for _, item := range items {
			if item.Code != "" {
				codes = append(codes, item.Code)
			}
		}
		return codes, nil
	}

	// Array of strings
	var stringsArr []string
	if err := json.Unmarshal(data, &stringsArr); err == nil {
		return stringsArr, nil
	}

	// Single object
	var single CodeObj
	if err := json.Unmarshal(data, &single); err == nil && single.Code != "" {
		return []string{single.Code}, nil
	}

	return nil, fmt.Errorf("invalid JSON format: must be array of objects, array of strings, or single object with 'code' key")
}

// redeemAll redeems every code in order and caches each terminal result in the
// state file. A failed request is skipped without being cached, so the code is
// retried on the next run.
func redeemAll(codes []string, fid, kid string, state StateMap, statePath string,
	redeemFn func(fid, cdk, kid string) (map[string]any, error), delay time.Duration) {
	for idx, currentCdk := range codes {
		fmt.Println("\n========================================")
		fmt.Printf("Processing gift code: %s\n", currentCdk)

		if isRedeemed(state, fid, currentCdk) {
			fmt.Printf("Code already redeemed or marked as processed for player %s (cached). Skipping.\n", fid)
			continue
		}

		if idx > 0 {
			fmt.Printf("Waiting %s to avoid API rate limits...\n", delay)
			time.Sleep(delay)
		}

		fmt.Println("Redeeming gift code...")
		result, err := redeemFn(fid, currentCdk, kid)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error redeeming:", err)
			continue
		}
		fmt.Println(prettyJSON(result))

		cVal := 0.0
		if cv, ok := result["code"].(float64); ok {
			cVal = cv
		} else if cv, ok := result["code"].(int); ok {
			cVal = float64(cv)
		}

		msgVal, _ := result["msg"].(string)

		isSuccess := cVal == 0

		if isSuccess {
			markRedeemed(state, fid, currentCdk)
			saveState(statePath, state)
			continue
		} else {
			// Other terminal error (e.g. CDK code error, claimed, expired) - stop retrying and cache code
			fmt.Printf("Terminal error received: %s. Caching code to prevent retries.\n", msgVal)
			markRedeemed(state, fid, currentCdk)
			saveState(statePath, state)
			continue
		}
	}
}

func main() {
	cliFID, cliKID, cdk, codesFile, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	reader := bufio.NewReader(os.Stdin)

	// Check if stdin is a pipe
	fi, err := os.Stdin.Stat()
	isPipe := err == nil && (fi.Mode()&os.ModeCharDevice) == 0

	// If no positional CDK and no codes file flag, but stdin is piped, default to reading from stdin
	if cdk == "" && codesFile == "" && isPipe {
		codesFile = "-"
	}

	fid, kid, err := resolveIDs(cliFID, cliKID, os.Getenv, ".env")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not read .env:", err)
	}

	interactive := (codesFile == "") && !isPipe

	var codesList []string
	if codesFile != "" {
		var data []byte
		if codesFile == "-" {
			fmt.Println("Reading codes list from stdin...")
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Reading codes list from %s...\n", codesFile)
			data, err = os.ReadFile(codesFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", codesFile, err)
				os.Exit(1)
			}
		}

		codesList, err = parseCodesJson(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing codes list JSON:", err)
			os.Exit(1)
		}
	} else if cdk != "" {
		codesList = []string{cdk}
	} else {
		if interactive {
			prompted := prompt(reader, "\nEnter gift code: ")
			if prompted != "" {
				codesList = []string{prompted}
			}
		} else {
			fmt.Fprintln(os.Stderr, "Error: No gift code specified and not running in interactive mode.")
			os.Exit(1)
		}
	}

	if len(codesList) == 0 {
		fmt.Println("No codes to process.")
		os.Exit(0)
	}

	statePath, err := stateFilePath()
	var state StateMap
	if err == nil {
		state = loadState(statePath)
	} else {
		state = make(StateMap)
	}

	redeemAll(codesList, fid, kid, state, statePath, redeem, 10*time.Second)
	fmt.Println("\nAll done!")
}
