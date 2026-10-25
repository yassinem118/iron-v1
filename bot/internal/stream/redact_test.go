package stream

import (
	"net/http"
	"testing"
)

func TestRedactedHeaders(t *testing.T) {
	in := http.Header{"Authorization": {"Bearer secret"}, "Cookie": {"sid=secret"}, "User-Agent": {"iron-test"}}
	out := RedactedHeaders(in)
	if out.Get("Authorization") != "[REDACTED]" || out.Get("Cookie") != "[REDACTED]" {
		t.Fatal("sensitive headers were not redacted")
	}
	if out.Get("User-Agent") != "iron-test" {
		t.Fatal("ordinary header unexpectedly changed")
	}
	if in.Get("Authorization") != "Bearer secret" {
		t.Fatal("input headers should not be mutated")
	}
}
