package cli

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/peppol-sh/peppol-sh-cli/internal/api"
)

const (
	sandboxPrefix = "ps_test_"
	livePrefix    = "ps_live_"
	sandboxURL    = "https://sandbox.peppol.sh"
	liveURL       = "https://api.peppol.sh"

	envAPIKey  = "PEPPOL_SH_API_KEY"
	envBaseURL = "PEPPOL_SH_BASE_URL"
)

type credential struct {
	key string
	// source is where the key came from, in words for whoami.
	source string
}

func (c credential) environment() string {
	if strings.HasPrefix(c.key, sandboxPrefix) {
		return "sandbox"
	}
	return "live"
}

func (c credential) masked() string {
	prefix := c.key[:min(len(c.key), len(sandboxPrefix))]
	if len(c.key) <= len(prefix)+4 {
		return prefix + "..."
	}
	return prefix + "..." + c.key[len(c.key)-4:]
}

func checkPrefix(key string) error {
	if strings.HasPrefix(key, sandboxPrefix) || strings.HasPrefix(key, livePrefix) {
		return nil
	}
	return newError(exitAuth, "the API key must start with %s (sandbox) or %s (live)", sandboxPrefix, livePrefix)
}

// resolveKey returns the key by the first source that has one, or nil.
func (a *app) resolveKey() (*credential, error) {
	if a.apiKey != "" {
		return &credential{key: a.apiKey, source: "--api-key flag"}, nil
	}
	if key := a.env.Getenv(envAPIKey); key != "" {
		return &credential{key: key, source: envAPIKey}, nil
	}
	key, err := loadKey(a.env.ConfigDir)
	if err != nil || key == "" {
		return nil, err
	}
	return &credential{key: key, source: "config file " + configPath(a.env.ConfigDir)}, nil
}

// baseURLFor is the API host for key ("" when there is no key).
func (a *app) baseURLFor(key string) (string, error) {
	base, name := a.baseURL, "--base-url"
	if base == "" {
		base, name = a.env.Getenv(envBaseURL), envBaseURL
	}
	if base != "" {
		u, err := url.Parse(base)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", newError(exitUsage, "%s must be an http or https URL, got %q", name, base)
		}
		return strings.TrimRight(base, "/"), nil
	}
	if strings.HasPrefix(key, sandboxPrefix) {
		return sandboxURL, nil
	}
	return liveURL, nil
}

type keyUse int

const (
	// keyHostOnly: the key picks the host but is not sent.
	keyHostOnly keyUse = iota
	keyOptional
	keyRequired
)

type session struct {
	cred   *credential
	host   string
	client *api.Client
}

func (a *app) open(use keyUse) (*session, error) {
	cred, err := a.resolveKey()
	if err != nil {
		return nil, err
	}
	if cred == nil && use == keyRequired {
		return nil, newError(exitAuth, "no API key").with(`Run "peppol-sh login" or set PEPPOL_SH_API_KEY.`)
	}
	if cred != nil && use != keyRequired && checkPrefix(cred.key) != nil {
		a.info("Warning: ignoring the API key from %s: it does not start with %s or %s.", cred.source, sandboxPrefix, livePrefix)
		cred = nil
	}
	return a.connect(cred, use != keyHostOnly)
}

func (a *app) connect(cred *credential, sendKey bool) (*session, error) {
	key := ""
	if cred != nil {
		if err := checkPrefix(cred.key); err != nil {
			return nil, err
		}
		key = cred.key
	}
	host, err := a.baseURLFor(key)
	if err != nil {
		return nil, err
	}

	opts := []api.ClientOption{api.WithHTTPClient(&http.Client{Transport: a.transport()})}
	if sendKey && key != "" {
		opts = append(opts, api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+key)
			return nil
		}))
	}
	client, err := api.NewClient(host, opts...)
	if err != nil {
		return nil, newError(exitUsage, "invalid API base URL %q: %v", host, err)
	}
	return &session{cred: cred, host: host, client: client}, nil
}
