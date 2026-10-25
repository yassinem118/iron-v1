package stream

import (
	"net/http"
	"strings"
)

// SensitiveHeaderNames are never emitted by the diagnostic logger.
var SensitiveHeaderNames = map[string]struct{}{
	"authorization": {}, "proxy-authorization": {}, "cookie": {}, "set-cookie": {},
	"x-api-key": {}, "x-auth-token": {}, "x-session-token": {},
}

// RedactedHeaders returns a copy suitable for diagnostic logs. It does not
// mutate the original request and never exposes credentials/cookies.
func RedactedHeaders(in http.Header) http.Header {
	out := make(http.Header, len(in))
	for k, values := range in {
		if _, sensitive := SensitiveHeaderNames[strings.ToLower(k)]; sensitive {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		out[k] = append([]string(nil), values...)
	}
	return out
}
