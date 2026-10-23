package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
)

// apiError renders a non-2xx response, from its ErrorObject when it has one.
func apiError(resp *http.Response, body []byte) error {
	code := exitError
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		code = exitAuth
	}

	var e *cliError
	var obj api.ErrorObject
	if json.Unmarshal(body, &obj) == nil && obj.Error.Code != "" {
		e = newError(code, "%s: %s (HTTP %d)", obj.Error.Code, obj.Error.Message, resp.StatusCode)
		if id := obj.Error.RequestId; resp.StatusCode >= 500 && id != nil && *id != "" {
			e.with("Request ID: " + *id)
		}
	} else {
		e = newError(code, "HTTP %s", resp.Status)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if wait, ok := retryAfter(resp.Header); ok {
			unit := "seconds"
			if wait == time.Second {
				unit = "second"
			}
			e.with(fmt.Sprintf("Rate limit reached. Wait %d %s, then try again.", int(wait.Seconds()), unit))
		} else {
			e.with("Rate limit reached. Wait a minute, then try again.")
		}
	}
	return e
}
