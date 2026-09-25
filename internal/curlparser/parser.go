package curlparser

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"
)

// ParsedCurl merepresentasikan hasil parse dari perintah cURL
type ParsedCurl struct {
	Method     string
	URL        string
	Headers    map[string]string
	Body       string
	AuthHeader string
	FormKey    string
	FormPath   string
}

// Parse mengurai string perintah cURL menjadi struct ParsedCurl.
// Mendukung flag: -X, -H, -d/--data/--data-raw, -u/--user, --url, dan URL posisional.
func Parse(raw string) (ParsedCurl, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\\\n", " ")
	raw = strings.ReplaceAll(raw, "\\\r\n", " ")

	tokens, err := tokenize(raw)
	if err != nil {
		return ParsedCurl{}, fmt.Errorf("tokenize error: %w", err)
	}

	if len(tokens) == 0 || strings.ToLower(tokens[0]) != "curl" {
		return ParsedCurl{}, fmt.Errorf("bukan perintah curl yang valid")
	}

	result := ParsedCurl{
		Method:  "GET",
		Headers: make(map[string]string),
	}

	explicitMethod := false
	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]

		switch tok {
		case "-X", "--request", "-H", "--header", "-d", "--data", "--data-raw", "--data-binary", "-u", "--user", "--url", "-F", "--form":
			if i+1 >= len(tokens) {
				return ParsedCurl{}, fmt.Errorf("missing value for %s", tok)
			}
		}
		switch tok {
		case "-X", "--request":
			if i+1 < len(tokens) {
				i++
				result.Method = strings.ToUpper(tokens[i])
				explicitMethod = true
			}

		case "-H", "--header":
			if i+1 < len(tokens) {
				i++
				key, val, ok := strings.Cut(tokens[i], ":")
				if ok && strings.TrimSpace(key) != "" {
					key = strings.TrimSpace(key)
					val = strings.TrimSpace(val)
					if strings.EqualFold(key, "authorization") {
						result.AuthHeader = val
					} else {
						result.Headers[key] = val
					}
				} else {
					return ParsedCurl{}, fmt.Errorf("invalid header %q: expected Key: Value", tokens[i])
				}
			}

		case "-d", "--data", "--data-raw", "--data-binary":
			if i+1 < len(tokens) {
				i++
				if tok != "--data-raw" && strings.HasPrefix(tokens[i], "@") {
					return ParsedCurl{}, fmt.Errorf("file body import is unsupported; paste the body with --data-raw")
				}
				result.Body = tokens[i]
				if !explicitMethod {
					result.Method = "POST"
				}
			}

		case "-u", "--user":
			if i+1 < len(tokens) {
				i++
				result.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(tokens[i]))
			}

		case "-F", "--form":
			i++
			key, value, ok := strings.Cut(tokens[i], "=@")
			if !ok || strings.TrimSpace(key) == "" || value == "" {
				return ParsedCurl{}, fmt.Errorf("only file multipart fields are supported")
			}
			// cURL permits attributes after the path, for example
			// `-F 'avatar=@photo.png;type=image/png'`.
			if path, _, found := strings.Cut(value, ";"); found {
				value = path
			}
			value = strings.TrimSpace(value)
			if value == "" {
				return ParsedCurl{}, fmt.Errorf("multipart file path is empty")
			}
			if result.FormKey != "" {
				return ParsedCurl{}, fmt.Errorf("only one multipart file is supported")
			}
			result.FormKey, result.FormPath = key, value
			if !explicitMethod {
				result.Method = "POST"
			}
		case "-I", "--head":
			result.Method = "HEAD"
			explicitMethod = true
		case "--url":
			if i+1 < len(tokens) {
				i++
				result.URL = tokens[i]
			}

		// Flag yang punya value argument - skip value-nya
		case "-o", "--output", "-c", "--cookie-jar", "-b", "--cookie",
			"--max-time", "--connect-timeout", "-A", "--user-agent",
			"--proxy", "-x":
			if i+1 < len(tokens) {
				i++
			}

		// Flag boolean tanpa value - abaikan saja
		case "-L", "--location", "-s", "--silent", "-v", "--verbose",
			"-i", "--include",
			"--compressed", "--http1.1", "--http2", "--no-keepalive":
			// abaikan

		default:
			if strings.HasPrefix(tok, "-") {
				return ParsedCurl{}, fmt.Errorf("unsupported curl option: %s", tok)
			}
			// Argumen bebas yang bukan flag = URL
			if !strings.HasPrefix(tok, "-") && result.URL == "" {
				result.URL = tok
			}
		}
	}

	if result.URL == "" {
		return ParsedCurl{}, fmt.Errorf("URL tidak ditemukan dalam perintah curl")
	}

	if !strings.HasPrefix(result.URL, "http://") && !strings.HasPrefix(result.URL, "https://") {
		result.URL = "https://" + result.URL
	}

	return result, nil
}

// tokenize memecah string menjadi token shell-style (mendukung single/double quote)
func tokenize(s string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune
	started, escaped := false, false
	for _, ch := range s {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			started = true
			continue
		}
		if ch == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				current.WriteRune(ch)
			}
			continue
		}
		switch {
		case ch == '\'' || ch == '"':
			quote = ch
			started = true
		case unicode.IsSpace(ch):
			if started {
				tokens = append(tokens, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(ch)
			started = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unclosed quote or escape in curl command")
	}
	if started {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}
