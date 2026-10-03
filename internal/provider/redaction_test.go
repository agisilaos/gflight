package provider

import (
	"errors"
	"fmt"
	"github.com/agisilaos/gflight/internal/model"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type redactionTransport func(*http.Request) (*http.Response, error)

func (f redactionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderDiagnosticsOmitSecrets(t *testing.T) {
	const secret = "SENTINEL-provider-secret"
	for _, cause := range []error{errors.New(secret), &net.DNSError{Err: secret, Name: secret}, io.EOF} {
		p := SerpAPIProvider{APIKey: secret, Retries: -1, Client: &http.Client{Transport: redactionTransport(func(*http.Request) (*http.Response, error) { return nil, cause })}}
		_, err := p.Search(model.SearchQuery{})
		if err == nil || strings.Contains(fmt.Sprintf("%v %+v", err, err), secret) {
			t.Fatalf("unsafe diagnostic: %v", err)
		}
		if !errors.Is(err, cause) {
			t.Fatal("transport identity lost")
		}
		if isNetworkTransient(&url.Error{Op: "Get", Err: cause}) != errors.Is(err, ErrTransient) {
			t.Fatal("retry classification changed")
		}
	}
	for _, code := range []int{400, 401, 403, 429, 500} {
		p := SerpAPIProvider{APIKey: secret, Retries: -1, Client: &http.Client{Transport: redactionTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Status: secret, Body: io.NopCloser(strings.NewReader(secret))}, nil
		})}}
		_, err := p.Search(model.SearchQuery{})
		if err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), fmt.Sprint(code)) {
			t.Fatalf("unsafe HTTP diagnostic: %v", err)
		}
	}
	p := SerpAPIProvider{APIKey: secret, BaseURL: "https://" + secret + "/%invalid"}
	_, err := p.Search(model.SearchQuery{})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe invalid URL diagnostic: %v", err)
	}
}
