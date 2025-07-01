package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/toaweme/log"
)

type Client interface {
	SetClient(client *http.Client)
	Get(ctx context.Context, req GetRequest) (*Response, error)
	Post(ctx context.Context, req PostRequest) (*Response, error)
	Put(ctx context.Context, req PutRequest) (*Response, error)
	Patch(ctx context.Context, req PatchRequest) (*Response, error)
	Delete(ctx context.Context, req Request) (*Response, error)
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type Request struct {
	ID        string
	SessionID string
	Path      string
	Query     url.Values
	Headers   map[string]string
}

type GetRequest struct {
	Request
}

type PostRequest struct {
	Request

	Body []byte
}

type PatchRequest PostRequest
type PutRequest PostRequest

type httpClient struct {
	baseURL string
	agent   string

	client  *http.Client
	headers map[string]string
	log     bool
}

type Config struct {
	BaseURL     string `json:"base_url"`
	UserAgent   string `json:"user_agent"`
	Platform    string `json:"platform"`
	ServiceName string `json:"service_name"`
	AppVersion  string `json:"app_version"`
	ClientID    string `json:"client_id"`
	Log         bool   `json:"log"`

	Headers map[string]string `json:"headers"`
}

func NewHttpClient(config Config) Client {
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	headers := map[string]string{
		ClientUserAgentHeaderName:  config.UserAgent,
		ClientPlatformHeaderName:   config.Platform,
		ClientAppVersionHeaderName: config.AppVersion,
		ClientIDHeaderName:         config.ClientID,
		ServiceNameHeaderName:      config.ServiceName,
	}
	for k, v := range headers {
		if v != "" {
			config.Headers[k] = v
		}
	}
	config.Log = true

	return httpClient{
		client:  http.DefaultClient,
		baseURL: config.BaseURL,
		headers: config.Headers,
		log:     config.Log,
	}
}

func (h httpClient) Get(ctx context.Context, req GetRequest) (*Response, error) {
	return h.do(ctx, http.MethodGet, req.Request, nil)
}

func (h httpClient) Post(ctx context.Context, req PostRequest) (*Response, error) {
	return h.do(ctx, http.MethodPost, req.Request, req.Body)
}

func (h httpClient) Patch(ctx context.Context, req PatchRequest) (*Response, error) {
	return h.do(ctx, http.MethodPatch, req.Request, req.Body)
}

func (h httpClient) Put(ctx context.Context, req PutRequest) (*Response, error) {
	return h.do(ctx, http.MethodPut, req.Request, req.Body)
}

func (h httpClient) Delete(ctx context.Context, req Request) (*Response, error) {
	return h.do(ctx, http.MethodDelete, req, nil)
}

func (h httpClient) SetClient(client *http.Client) {
	h.client = client
}

func limitBodySize(body []byte, maxSize int64) string {
	if int64(len(body)) > maxSize {
		return string(body[:maxSize]) + "..."
	}
	return string(body)
}

const size = 100

func (h httpClient) do(ctx context.Context, method string, req Request, body []byte) (*Response, error) {
	path, headers, err := h.buildRequestParams(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URI: %w", err)
	}

	if h.log {
		log.Trace("http-client", "type", "request", "method", method, "url", path, "query", req.Query, "body", limitBodySize(body, size))
	}

	var httpReq *http.Request
	// prepare request
	if body != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, path, bytes.NewBuffer(body))
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, path, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// set headers
	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}

	// send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if h.log {
		log.Trace("http-client", "type", "response", "method", method, "url", path, "status", resp.StatusCode, "body", limitBodySize(data, size))
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       data,
		Headers:    resp.Header,
	}, nil
}

func (h httpClient) buildRequestParams(req Request) (string, map[string]string, error) {
	headers := make(map[string]string)
	for k, v := range h.headers {
		headers[k] = v
	}
	for k, v := range req.Headers {
		headers[k] = v
	}

	if req.ID != "" {
		headers[ClientRequestIDHeaderName] = req.ID
	}
	if req.SessionID != "" {
		headers[ClientSessionIDHeaderName] = req.SessionID
	}

	// prepare URL
	var path = req.Path
	var err error
	if h.baseURL != "" {
		path, err = url.JoinPath(h.baseURL, req.Path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to join URL: %s: %w", req.Path, err)
		}
	}

	// prepare query
	query := req.Query.Encode()
	if query != "" {
		path += "?" + query
	}

	return path, headers, nil
}
