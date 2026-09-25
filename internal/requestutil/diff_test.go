package requestutil

import (
	"fmt"
	"strings"
	"testing"
)

func TestDiffJSONFields(t *testing.T) {
	got := Diff(`{"id":1,"name":"a"}`, `{"id":1,"name":"b"}`)
	want := "  {\n    \"id\": 1,\n-   \"name\": \"a\"\n+   \"name\": \"b\"\n  }"
	if got != want {
		t.Fatalf("got:\n%s", got)
	}
	if got := Diff("x\ny", "x\ny\nz"); got != "  x\n  y\n+ z" {
		t.Fatalf("got:\n%s", got)
	}
}

func TestDiffLargeBodySmallChangeStaysCheap(t *testing.T) {
	var a, b []string
	for i := 0; i < 50_000; i++ {
		a = append(a, fmt.Sprint(i))
	}
	b = append(append(b, a...), "new")
	got := Diff(strings.Join(a, "\n"), strings.Join(b, "\n"))
	if !strings.HasSuffix(got, "\n+ new") {
		t.Fatalf("unexpected tail %q", got[len(got)-20:])
	}
}
