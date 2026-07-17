// WOS Gift-Code Redemption Client (reverse engineered) - Go implementation
// Usage: go run . [-u userid] <code>
//        go run . [-u userid] -f <codes.json>
package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	salt       = "tB87#kPtkxqOS2"
	baseURL    = "https://wos-giftcode-api.centurygame.com/api"
	defaultFID = "123456789"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/126.0 Safari/537.36"
)

var dataURIRe = regexp.MustCompile(`^data:image/(\w+);base64,`)

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

func now() string { return strconv.FormatInt(time.Now().UnixMilli(), 10) }

// --- Endpoints ---
func getPlayer(fid string) (map[string]any, error) {
	return api("/player", map[string]string{"fid": fid, "time": now()})
}

func getCaptcha(fid string) (map[string]any, error) {
	return api("/captcha", map[string]string{"fid": fid, "time": now(), "init": "0"})
}

func redeem(fid, cdk, captchaCode string) (map[string]any, error) {
	return api("/gift_code", map[string]string{
		"fid": fid, "cdk": cdk, "captcha_code": captchaCode, "time": now(),
	})
}

// openInBrowser opens a file with the OS default application.
func openInBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Could not open file automatically:", err)
	}
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

// defaultFIDFromDotEnv reads DEFAULT_FID from a local .env file.
func defaultFIDFromDotEnv(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	for _, rawLine := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != "DEFAULT_FID" {
			continue
		}

		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
				v = v[1 : len(v)-1]
			}
		}
		return strings.TrimSpace(v), nil
	}

	return "", nil
}

// resolveFID applies precedence: CLI arg > env var > .env > built-in default.
func resolveFID(cliFID string, getenv func(string) string, dotEnvPath string) (string, error) {
	fid := defaultFID
	var readErr error

	// 1. Check global ~/.wos-cli-env config (lowest priority config)
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".wos-cli-env")
		if globalFID, err := defaultFIDFromDotEnv(globalPath); err == nil && globalFID != "" {
			fid = globalFID
		}
	}

	// 2. Check local .env (higher priority than global)
	dotEnvFID, err := defaultFIDFromDotEnv(dotEnvPath)
	if err != nil {
		if !os.IsNotExist(err) {
			readErr = err
		}
	} else if dotEnvFID != "" {
		fid = dotEnvFID
	}

	// 3. Check environment variable (higher priority than local .env)
	if envFID := strings.TrimSpace(getenv("DEFAULT_FID")); envFID != "" {
		fid = envFID
	}

	// 4. Check CLI flag (highest priority)
	if cliFID = strings.TrimSpace(cliFID); cliFID != "" {
		fid = cliFID
	}

	return fid, readErr
}

// parseArgs parses the flags -u and -f, and returns positional args if any.
func parseArgs(args []string) (string, string, string, error) {
	fs := flag.NewFlagSet("wos-giftcode-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	userID := fs.String("u", "", "user id")
	codesFile := fs.String("f", "", "path to JSON codes file (use '-' for stdin)")

	if err := fs.Parse(args); err != nil {
		return "", "", "", fmt.Errorf("usage:\n  wos-giftcode-cli [-u userid] <code>\n  wos-giftcode-cli [-u userid] -f <codes.json>")
	}

	rest := fs.Args()
	if *codesFile != "" {
		if len(rest) > 0 {
			return "", "", "", fmt.Errorf("cannot specify both -f and a direct giftcode")
		}
		return strings.TrimSpace(*userID), "", *codesFile, nil
	}

	code := ""
	if len(rest) == 1 {
		code = strings.TrimSpace(rest[0])
	} else if len(rest) > 1 {
		return "", "", "", fmt.Errorf("too many arguments\nusage:\n  wos-giftcode-cli [-u userid] <code>\n  wos-giftcode-cli [-u userid] -f <codes.json>")
	}

	return strings.TrimSpace(*userID), code, "", nil
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

func main() {
	cliFID, cdk, codesFile, err := parseArgs(os.Args[1:])
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

	fid, err := resolveFID(cliFID, os.Getenv, ".env")
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

	for idx, currentCdk := range codesList {
		fmt.Println("\n========================================")
		fmt.Printf("Processing gift code: %s\n", currentCdk)

		if isRedeemed(state, fid, currentCdk) {
			fmt.Printf("Code already redeemed or marked as processed for player %s (cached). Skipping.\n", fid)
			continue
		}

		if idx > 0 {
			fmt.Println("Waiting 10 seconds to avoid API rate limits...")
			time.Sleep(10 * time.Second)
		}

		fmt.Printf("1) Identify player %s...\n", fid)
		player, err := getPlayer(fid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error identifying player: %v. Skipping code.\n", err)
			continue
		}
		fmt.Println(prettyJSON(player))

		codeVal := 0.0
		if cv, ok := player["code"].(float64); ok {
			codeVal = cv
		} else if cv, ok := player["code"].(int); ok {
			codeVal = float64(cv)
		}
		if codeVal != 0 {
			fmt.Printf("Failed to identify player %s. Skipping code.\n", fid)
			continue
		}

		captchaCode := ""
		attempts := 0
		maxAttempts := 4
		delay := 15 * time.Second

		for attempts < maxAttempts {
			attempts++
			fmt.Printf("\n2) Fetching captcha (attempt %d/%d)...\n", attempts, maxAttempts)
			cap, err := getCaptcha(fid)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error fetching captcha:", err)
				break
			}

			capData, _ := cap["data"].(map[string]any)
			raw, _ := capData["img"].(string)
			if raw == "" {
				msg, _ := cap["msg"].(string)
				msgLower := strings.ToLower(msg)
				if strings.Contains(msgLower, "frequent") {
					if attempts < maxAttempts {
						fmt.Printf("CAPTCHA fetch frequent limit hit. Waiting 15 seconds before trying again (attempt %d/%d)...\n", attempts+1, maxAttempts)
						time.Sleep(delay)
						_, _ = getPlayer(fid)
						continue
					}
				}
				fmt.Fprintln(os.Stderr, "No captcha image in response:", prettyJSON(cap))
				break
			}

			ext := "png"
			if m := dataURIRe.FindStringSubmatch(raw); m != nil {
				ext = m[1]
			}
			b64 := dataURIRe.ReplaceAllString(raw, "")
			imgData, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed to decode captcha image:", err)
				break
			}

			imgPath := filepath.Join(os.TempDir(), "wos-captcha."+ext)
			if err := os.WriteFile(imgPath, imgData, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to write captcha image:", err)
				break
			}

			// Try to automatically solve the captcha
			solvedCode := ""
			solverPath := ""
			candidates := []string{
				"./solver/solver.py",
				"../solver/solver.py",
				"./go/../solver/solver.py",
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					solverPath = c
					break
				}
			}

			solverBroken := false
			if solverPath != "" {
				pythonCmd := "python3"
				if runtime.GOOS == "windows" {
					pythonCmd = "python"
				}
				
				dir := filepath.Dir(solverPath)
				venvPython := filepath.Join(dir, "venv", "bin", "python")
				if runtime.GOOS == "windows" {
					venvPython = filepath.Join(dir, "venv", "Scripts", "python.exe")
				}
				if _, err := os.Stat(venvPython); err == nil {
					pythonCmd = venvPython
				}

				cmd := exec.Command(pythonCmd, solverPath, imgPath)
				output, err := cmd.Output()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Solver execution failed:", err)
					solverBroken = true
				} else {
					solved := strings.TrimSpace(string(output))
					if strings.Contains(solved, "ERROR") {
						fmt.Fprintln(os.Stderr, "Solver returned error:", solved)
						solverBroken = true
					} else if len(solved) == 4 {
						solvedCode = solved
					} else {
						fmt.Printf("Solver returned invalid guess length (%d): %s\n", len(solved), solved)
					}
				}
			} else {
				fmt.Fprintln(os.Stderr, "Solver script not found.")
				solverBroken = true
			}

			if solvedCode == "" {
				if solverBroken {
					fmt.Println("Automatic solver is unavailable or broken.")
					if !interactive {
						fmt.Println("Running in non-interactive mode. Skipping this code.")
						break
					}
					openInBrowser(imgPath)
					captchaCode = prompt(reader, "\n4) Enter the captcha code shown: ")
				} else {
					if attempts < maxAttempts {
						fmt.Printf("Bypassing invalid solver guess. Retrying with a new captcha (attempt %d/%d)...\n", attempts+1, maxAttempts)
						time.Sleep(2 * time.Second)
						continue
					} else {
						fmt.Println("Auto-solve failed 4 times (invalid guesses).")
						if !interactive {
							fmt.Println("Running in non-interactive mode. Skipping this code.")
							break
						}
						openInBrowser(imgPath)
						captchaCode = prompt(reader, "\n4) Enter the captcha code shown: ")
					}
				}
			} else {
				fmt.Printf("Automatically solved CAPTCHA: %s\n", solvedCode)
				captchaCode = solvedCode
			}

			fmt.Println("Redeeming gift code...")
			result, err := redeem(fid, currentCdk, captchaCode)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error redeeming:", err)
				break
			}
			fmt.Println(prettyJSON(result))

			cVal := 0.0
			if cv, ok := result["code"].(float64); ok {
				cVal = cv
			} else if cv, ok := result["code"].(int); ok {
				cVal = float64(cv)
			}

			msgVal, _ := result["msg"].(string)


			isSuccess := (cVal == 0)
			msgLower := strings.ToLower(msgVal)
			isCaptchaError := strings.Contains(msgLower, "captcha")

			if isSuccess {
				markRedeemed(state, fid, currentCdk)
				saveState(statePath, state)
				break
			}

			if isCaptchaError {
				if attempts < maxAttempts {
					fmt.Printf("CAPTCHA incorrect. Waiting 15 seconds before trying again (attempt %d/%d)...\n", attempts+1, maxAttempts)
					time.Sleep(delay)
					_, _ = getPlayer(fid)
				} else {
					fmt.Println("\nAuto-solve failed 4 times.")
					if !interactive {
						fmt.Println("Running in non-interactive mode. Skipping this code.")
						break
					}
					openInBrowser(imgPath)
					captchaCode = prompt(reader, "\n4) Enter the captcha code shown: ")
					fmt.Println("\nRedeeming gift code...")
					finalRes, err := redeem(fid, currentCdk, captchaCode)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						break
					}
					fmt.Println(prettyJSON(finalRes))

					fcVal := 0.0
					if fcv, ok := finalRes["code"].(float64); ok {
						fcVal = fcv
					} else if fcv, ok := finalRes["code"].(int); ok {
						fcVal = float64(fcv)
					}
					fMsgVal, _ := finalRes["msg"].(string)


					fMsgLower := strings.ToLower(fMsgVal)
					fIsCaptchaError := strings.Contains(fMsgLower, "captcha")

					if fcVal == 0 || !fIsCaptchaError {
						markRedeemed(state, fid, currentCdk)
						saveState(statePath, state)
					}
					break
				}
			} else {
				// Other terminal error (e.g. CDK code error, claimed, expired) - stop retrying and cache code
				fmt.Printf("Terminal error received: %s. Caching code to prevent retries.\n", msgVal)
				markRedeemed(state, fid, currentCdk)
				saveState(statePath, state)
				break
			}
		}
	}
	fmt.Println("\nAll done!")
}
