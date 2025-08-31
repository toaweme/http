package http

import (
	"bufio"
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
	GetStream(ctx context.Context, stream chan StreamResponse, req Request) error
	Post(ctx context.Context, req PostRequest) (*Response, error)
	PostStream(ctx context.Context, stream chan StreamResponse, req PostRequest) error
	Put(ctx context.Context, req PutRequest) (*Response, error)
	Patch(ctx context.Context, req PatchRequest) (*Response, error)
	Delete(ctx context.Context, req Request) (*Response, error)
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Error      error
}

type StreamResponseType string

const (
	StreamResponseTypeEOF     StreamResponseType = "EOF"
	StreamResponseTypeData    StreamResponseType = "DATA"
	StreamResponseTypeEvent   StreamResponseType = "EVENT"
	StreamResponseTypeID      StreamResponseType = "ID"
	StreamResponseTypeRetry   StreamResponseType = "RETRY"
	StreamResponseTypeComment StreamResponseType = "COMMENT"
)

type StreamResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Error      error
	Type       StreamResponseType
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

func (h httpClient) GetStream(ctx context.Context, stream chan StreamResponse, req Request) error {
	return h.doStream(ctx, http.MethodGet, stream, req, nil)
}

func (h httpClient) Post(ctx context.Context, req PostRequest) (*Response, error) {
	return h.do(ctx, http.MethodPost, req.Request, req.Body)
}

func (h httpClient) PostStream(ctx context.Context, stream chan StreamResponse, req PostRequest) error {
	return h.doStream(ctx, http.MethodPost, stream, req.Request, req.Body)
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
		log.Trace("http-client", "type", "request", "method", method, "headers", headers, "url", path, "query", req.Query, "body", string(body))
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
		log.Trace("http-client", "type", "response", "method", method, "url", path, "status", resp.StatusCode, "body", string(data))
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       data,
		Headers:    resp.Header,
	}, nil
}

func (h httpClient) doStream(ctx context.Context, method string, stream chan StreamResponse, req Request, body []byte) error {
	path, headers, err := h.buildRequestParams(req)
	if err != nil {
		return fmt.Errorf("failed to build request URI: %w", err)
	}

	logger := log.Logger.With("type", "stream-request", "method", method, "url", path, "query", req.Query, "req-body", limitBodySize(body, size))

	if h.log {
		logger.Debug("http-client")
	}

	var bodyBuffer *bytes.Buffer
	if body != nil {
		bodyBuffer = bytes.NewBuffer(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, path, bodyBuffer)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	logger = logger.With("status", resp.StatusCode)

	if h.log {
		logger.Debug("http-client", "request", "sent")
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		defer close(stream)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to read error response body: %w", err)
			if h.log {
				logger.Error("http-client", "error", err)
			}
			return err
		}

		err = fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(respBody))

		if h.log {
			logger.Error("http-client", "stream", "started-with-error", "status", resp.StatusCode, "error", err)
		}

		stream <- StreamResponse{
			Type:       StreamResponseTypeEOF,
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Error:      err,
			Body:       respBody,
		}

		return err
	}

	if h.log {
		logger.Debug("http-client", "stream", "started")
	}

	go func() {
		defer resp.Body.Close()
		defer close(stream)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if h.log {
				logger.Debug("http-client", "raw-line", string(line))
			}
			if err != nil {
				stream <- StreamResponse{
					Type:       StreamResponseTypeEOF,
					StatusCode: resp.StatusCode,
					Headers:    resp.Header,
					Error:      fmt.Errorf("failed to read response body: %w", err),
				}
				if h.log {
					logger.Error("http-client", "stream", "ended-with-error", "error", err)
				}
				break
			}

			resType := StreamResponseTypeData
			line = bytes.TrimSpace(line)

			logger = logger.With("type", resType)

			if len(line) == 0 {
				continue
			}
			if h.log {
				logger.Debug("http-client", "pre-processed-line", string(line))
			}
			if bytes.HasPrefix(line, []byte("data: ")) {
				line = bytes.TrimPrefix(line, []byte("data: "))
				if bytes.Equal(line, []byte("[DONE]")) {
					stream <- StreamResponse{
						Type:       StreamResponseTypeEOF,
						StatusCode: resp.StatusCode,
						Headers:    resp.Header,
					}
					return
				}
			} else if bytes.HasPrefix(line, []byte("event: ")) {
				resType = StreamResponseTypeEvent
				line = bytes.TrimPrefix(line, []byte("event: "))
			} else if bytes.HasPrefix(line, []byte("id: ")) {
				resType = StreamResponseTypeID
				line = bytes.TrimPrefix(line, []byte("id: "))
			} else if bytes.HasPrefix(line, []byte("retry: ")) {
				resType = StreamResponseTypeRetry
				line = bytes.TrimPrefix(line, []byte("retry: "))
			} else if bytes.HasPrefix(line, []byte(":")) {
				resType = StreamResponseTypeComment
			}

			stream <- StreamResponse{
				Type:       resType,
				StatusCode: resp.StatusCode,
				Headers:    resp.Header,
				Body:       line,
			}
			if h.log {
				logger.Debug("http-client", "sse-processed-line", string(line))
			}
		}
	}()

	return nil
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
