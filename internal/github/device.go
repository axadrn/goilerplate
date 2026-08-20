package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultDeviceURL = "https://github.com/login/device/code"
	defaultTokenURL  = "https://github.com/login/oauth/access_token"
	deviceGrantType  = "urn:ietf:params:oauth:grant-type:device_code"
)

var ErrAccessDenied = errors.New("GitHub authorization was denied")
var ErrDeviceExpired = errors.New("GitHub authorization code expired")

type DeviceAuthorization struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DeviceClient struct {
	httpClient *http.Client
	deviceURL  string
	tokenURL   string
	now        func() time.Time
	wait       func(context.Context, time.Duration) error
}

func NewDeviceClient(httpClient *http.Client) *DeviceClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &DeviceClient{
		httpClient: &clientCopy,
		deviceURL:  defaultDeviceURL,
		tokenURL:   defaultTokenURL,
		now:        time.Now,
		wait:       waitContext,
	}
}

func (c *DeviceClient) Start(ctx context.Context, clientID string) (DeviceAuthorization, error) {
	if strings.TrimSpace(clientID) == "" {
		return DeviceAuthorization{}, errors.New("GitHub OAuth client ID is required")
	}
	values := url.Values{"client_id": {clientID}, "scope": {"user:email"}}
	var authorization DeviceAuthorization
	if err := c.postForm(ctx, c.deviceURL, values, &authorization); err != nil {
		return DeviceAuthorization{}, fmt.Errorf("start GitHub device authorization: %w", err)
	}
	if authorization.DeviceCode == "" || authorization.UserCode == "" || authorization.VerificationURI == "" || authorization.ExpiresIn <= 0 {
		return DeviceAuthorization{}, errors.New("GitHub returned an incomplete device authorization")
	}
	if authorization.Interval < 1 {
		authorization.Interval = 5
	}
	return authorization, nil
}

func (c *DeviceClient) Wait(ctx context.Context, clientID string, authorization DeviceAuthorization) (string, error) {
	if strings.TrimSpace(clientID) == "" || authorization.DeviceCode == "" {
		return "", errors.New("GitHub OAuth client ID and device code are required")
	}
	interval := time.Duration(authorization.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	deadline := c.now().Add(time.Duration(authorization.ExpiresIn) * time.Second)

	for c.now().Before(deadline) {
		if err := c.wait(ctx, interval); err != nil {
			return "", err
		}
		values := url.Values{
			"client_id":   {clientID},
			"device_code": {authorization.DeviceCode},
			"grant_type":  {deviceGrantType},
		}
		var response tokenResponse
		if err := c.postForm(ctx, c.tokenURL, values, &response); err != nil {
			return "", fmt.Errorf("poll GitHub device authorization: %w", err)
		}
		switch response.Error {
		case "":
			if response.AccessToken == "" {
				return "", errors.New("GitHub returned an empty OAuth token")
			}
			return response.AccessToken, nil
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "expired_token":
			return "", ErrDeviceExpired
		case "access_denied":
			return "", ErrAccessDenied
		default:
			return "", fmt.Errorf("GitHub device authorization failed: %s", response.Error)
		}
	}
	return "", ErrDeviceExpired
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

func (c *DeviceClient) postForm(ctx context.Context, endpoint string, values url.Values, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "goilerplate")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return fmt.Errorf("read GitHub response: %w", err)
	}
	if len(content) > 1<<20 {
		return errors.New("GitHub response exceeds the size limit")
	}
	if err := json.Unmarshal(content, destination); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
