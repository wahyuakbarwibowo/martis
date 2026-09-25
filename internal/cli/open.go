package cli

import "strings"

// InitialCurl turns "martis <url>" or "martis curl ..." into a cURL command
// for the TUI to open with; it returns "" for anything else.
func InitialCurl(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if args[0] != "curl" {
		if len(args) != 1 || !(strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://")) {
			return ""
		}
		args = []string{"curl", args[0]}
	}
	// Re-quote so arguments with spaces survive the shell-style tokenizer.
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(quoted, " ")
}
