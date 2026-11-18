package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type managedPlatform struct {
	origin string
	token  string
	client *http.Client
}

func newManagedPlatform(address, token string) (*managedPlatform, error) {
	origin, err := url.Parse(address)
	if err != nil || origin == nil || origin.Hostname() == "" ||
		(origin.Scheme != "https" && origin.Scheme != "http") || origin.User != nil ||
		(origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("managed platform address must be an HTTP(S) origin")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("managed platform token is required")
	}
	return &managedPlatform{origin: strings.TrimRight(origin.String(), "/"), token: token, client: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (p *managedPlatform) post(path string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, p.origin+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("managed platform returned HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(response)
}
