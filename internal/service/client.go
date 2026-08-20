package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/axadrn/goilerplate/api"
)

const maxResponseSize = 1 << 20
const maxProjectArchiveSize = 64 << 20

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API URL %q", baseURL)
	}
	hostIP := net.ParseIP(parsed.Hostname())
	loopback := parsed.Hostname() == "localhost" || hostIP != nil && hostIP.IsLoopback()
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
		return nil, errors.New("API URL must use HTTPS outside localhost")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return nil, errors.New("API URL must contain only scheme and host")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{baseURL: parsed, httpClient: &clientCopy}, nil
}

func (c *Client) LoginWithGitHub(ctx context.Context, githubToken string) (api.GitHubLoginResponse, error) {
	if strings.TrimSpace(githubToken) == "" {
		return api.GitHubLoginResponse{}, errors.New("GitHub OAuth token is required")
	}
	request := api.GitHubLoginRequest{OAuthAccessToken: githubToken}
	var response api.GitHubLoginResponse
	if err := c.doJSON(ctx, http.MethodPost, api.PathGitHubLogin, "", request, &response); err != nil {
		return api.GitHubLoginResponse{}, err
	}
	if response.SessionToken == "" {
		return api.GitHubLoginResponse{}, errors.New("service returned an empty session token")
	}
	return response, nil
}

func (c *Client) WhoAmI(ctx context.Context, sessionToken string) (api.WhoAmIResponse, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return api.WhoAmIResponse{}, errors.New("Goilerplate session token is required")
	}
	var response api.WhoAmIResponse
	if err := c.doJSON(ctx, http.MethodGet, api.PathWhoAmI, sessionToken, nil, &response); err != nil {
		return api.WhoAmIResponse{}, err
	}
	return response, nil
}

func (c *Client) Logout(ctx context.Context, sessionToken string) error {
	if strings.TrimSpace(sessionToken) == "" {
		return errors.New("Goilerplate session token is required")
	}
	return c.doJSON(ctx, http.MethodPost, api.PathLogout, sessionToken, nil, nil)
}

func (c *Client) ActivationStatus(ctx context.Context, sessionToken string) (api.ActivationStatusResponse, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return api.ActivationStatusResponse{}, errors.New("Goilerplate session token is required")
	}
	var response api.ActivationStatusResponse
	if err := c.doJSON(ctx, http.MethodGet, api.PathActivation, sessionToken, nil, &response); err != nil {
		return api.ActivationStatusResponse{}, err
	}
	return response, nil
}

func (c *Client) ResendActivation(ctx context.Context, sessionToken string) (api.ActivationStatusResponse, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return api.ActivationStatusResponse{}, errors.New("Goilerplate session token is required")
	}
	var response api.ActivationStatusResponse
	if err := c.doJSON(ctx, http.MethodPost, api.PathActivationResend, sessionToken, nil, &response); err != nil {
		return api.ActivationStatusResponse{}, err
	}
	return response, nil
}

func (c *Client) Generate(
	ctx context.Context,
	sessionToken string,
	requestBody api.GenerateRequest,
	destination io.Writer,
) (string, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return "", errors.New("Goilerplate session token is required")
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("encode generation request: %w", err)
	}
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: api.PathGenerate})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/gzip")
	request.Header.Set("Authorization", "Bearer "+sessionToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "goilerplate")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("call Goilerplate service: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", responseError(response)
	}
	written, err := io.Copy(destination, io.LimitReader(response.Body, maxProjectArchiveSize+1))
	if err != nil {
		return "", fmt.Errorf("download generated project: %w", err)
	}
	if written > maxProjectArchiveSize {
		return "", errors.New("generated project exceeds the download limit")
	}
	version := strings.TrimSpace(response.Header.Get("X-Goilerplate-Version"))
	if version == "" {
		return "", errors.New("service returned no template version")
	}
	return version, nil
}

func (c *Client) doJSON(ctx context.Context, method, requestPath, sessionToken string, body, destination any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: requestPath})
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "goilerplate")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if sessionToken != "" {
		request.Header.Set("Authorization", "Bearer "+sessionToken)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call Goilerplate service: %w", err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("read Goilerplate response: %w", err)
	}
	if len(content) > maxResponseSize {
		return errors.New("Goilerplate response exceeds the size limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return apiErrorFromContent(response.StatusCode, content)
	}
	if destination == nil {
		return nil
	}
	if err := json.Unmarshal(content, destination); err != nil {
		return fmt.Errorf("decode Goilerplate response: %w", err)
	}
	return nil
}

func responseError(response *http.Response) error {
	content, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("read Goilerplate response: %w", err)
	}
	if len(content) > maxResponseSize {
		return errors.New("Goilerplate response exceeds the size limit")
	}
	return apiErrorFromContent(response.StatusCode, content)
}

func apiErrorFromContent(status int, content []byte) error {
	var apiError api.ErrorResponse
	if json.Unmarshal(content, &apiError) == nil && apiError.Message != "" {
		return fmt.Errorf("Goilerplate service: %s", apiError.Message)
	}
	return fmt.Errorf("Goilerplate service returned HTTP %d", status)
}
