package requestutil

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Diff returns a line diff of two response bodies ("- " old, "+ " new).
// JSON bodies are indented first so single-line payloads diff per field.
func Diff(old, new string) string {
	a, b := strings.Split(IndentJSON(old), "\n"), strings.Split(IndentJSON(new), "\n")
	// Trim the shared head and tail so the LCS only covers the changed middle.
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head && a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
	}
	out := make([]string, 0, len(a)+len(b))
	for _, line := range a[:head] {
		out = append(out, "  "+line)
	}
	ma, mb := a[head:len(a)-tail], b[head:len(b)-tail]
	// ponytail: O(n*m) LCS on the changed middle, capped at ~4 MB; switch to Myers if big diffs matter.
	if len(ma)*len(mb) > 1_000_000 {
		return "Perubahan terlalu besar untuk dibandingkan"
	}
	lcs := make([][]int32, len(ma)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(mb)+1)
	}
	for i := len(ma) - 1; i >= 0; i-- {
		for j := len(mb) - 1; j >= 0; j-- {
			if ma[i] == mb[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(ma) || j < len(mb) {
		switch {
		case i < len(ma) && j < len(mb) && ma[i] == mb[j]:
			out = append(out, "  "+ma[i])
			i, j = i+1, j+1
		case i < len(ma) && (j == len(mb) || lcs[i+1][j] >= lcs[i][j+1]):
			out = append(out, "- "+ma[i])
			i++
		default:
			out = append(out, "+ "+mb[j])
			j++
		}
	}
	for _, line := range a[len(a)-tail:] {
		out = append(out, "  "+line)
	}
	return strings.Join(out, "\n")
}

// IndentJSON pretty-prints JSON with two spaces; non-JSON input is returned as is.
func IndentJSON(s string) string {
	var buf bytes.Buffer
	if json.Indent(&buf, []byte(s), "", "  ") != nil {
		return s
	}
	return buf.String()
}
