package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client performs HTTP requests against a configured base URL, buffering responses
// by default and streaming them (body or Server-Sent Events) when asked.
type Client interface {
	Get(ctx context.Context, req GetRequest) (*Response, error)
	GetStream(ctx context.Context, stream chan StreamResponse, req Request) error
	Post(ctx context.Context, req PostRequest) (*Response, error)
	PostStream(ctx context.Context, stream chan StreamResponse, req PostRequest) error
	Put(ctx context.Context, req PutRequest) (*Response, error)
	Patch(ctx context.Context, req PatchRequest) (*Response, error)
	Delete(ctx context.Context, req Request) (*Response, error)
}

// Response is the outcome of a request: a buffered Body or, for a streamed request,
// a live Reader, alongside the status code and headers.
type Response struct {
	StatusCode int
	// Body holds the fully-read response for a buffered request (Request.Stream
	// false, the default). It is nil for a streamed request, where Reader carries
	// the live body instead.
	Body []byte
	// Reader is the live, unread response body of a streamed request (Request.Stream
	// true). The caller owns it and must Close it (Response itself is an io.ReadCloser
	// over it, so io.Copy(dst, resp) then resp.Close() works). It is nil for a
	// buffered request.
	Reader  io.ReadCloser
	Headers http.Header
	Error   error
}

var _ io.ReadCloser = (*Response)(nil)

// Read streams a streamed response's body, letting a Response be passed straight to
// io.Copy. It reads nothing for a buffered response (use Body for those).
func (r *Response) Read(p []byte) (int, error) {
	if r.Reader == nil {
		return 0, io.EOF
	}
	return r.Reader.Read(p)
}

// Close releases a streamed response's body. It is a no-op for a buffered response,
// so callers can defer it unconditionally.
func (r *Response) Close() error {
	if r.Reader == nil {
		return nil
	}
	return r.Reader.Close()
}

// StreamResponseType classifies a decoded Server-Sent Events frame.
type StreamResponseType string

// Server-Sent Events frame types decoded from a stream.
const (
	StreamResponseTypeEOF     StreamResponseType = "EOF"
	StreamResponseTypeData    StreamResponseType = "DATA"
	StreamResponseTypeEvent   StreamResponseType = "EVENT"
	StreamResponseTypeID      StreamResponseType = "ID"
	StreamResponseTypeRetry   StreamResponseType = "RETRY"
	StreamResponseTypeComment StreamResponseType = "COMMENT"
)

// StreamResponse is one decoded Server-Sent Events frame from a streaming request.
type StreamResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Error      error
	Type       StreamResponseType
}

// Request is the shared shape of every request: path, query, headers, identifiers,
// and per-request flags.
type Request struct {
	ID        string
	SessionID string
	Path      string
	Query     url.Values
	Headers   map[string]string
	// Stream, when true, skips buffering the response body into Response.Body and
	// hands the live stream back as Response.Reader instead, so large downloads never
	// round-trip through memory. The caller must Close the Response. Default false
	// keeps the buffered Body behavior every other caller relies on.
	Stream bool
}

// GetRequest is a GET request.
type GetRequest struct {
	Request
}

// PostRequest is a POST request carrying a body.
type PostRequest struct {
	Request

	Body []byte
}

// PatchRequest is a PATCH request carrying a body.
type PatchRequest PostRequest

// PutRequest is a PUT request carrying a body.
type PutRequest PostRequest

type httpClient struct {
	baseURL string

	client  *http.Client
	headers map[string]string
	logger  Logger
}

var _ Client = httpClient{}

// Config is the static, construction-time configuration of a Client: its base URL
// and the identifying headers stamped onto every request.
type Config struct {
	BaseURL     string            `json:"base_url"`
	UserAgent   string            `json:"user_agent"`
	Platform    string            `json:"platform"`
	ServiceName string            `json:"service_name"`
	AppVersion  string            `json:"app_version"`
	ClientID    string            `json:"client_id"`
	Headers     map[string]string `json:"headers"`
}

// Option configures a Client at construction time.
type Option func(*httpClient)

// WithLogger injects the logger the client writes diagnostics to. Without it
// the client uses a nil logger and stays silent.
func WithLogger(logger Logger) Option {
	return func(h *httpClient) {
		if logger != nil {
			h.logger = logger
		}
	}
}

// WithHTTPClient swaps the underlying *http.Client used for every request, so
// callers can set custom timeouts, transports, or inject a stub in tests.
// Without it the client uses http.DefaultClient. A nil client is ignored.
func WithHTTPClient(client *http.Client) Option {
	return func(h *httpClient) {
		if client != nil {
			h.client = client
		}
	}
}

// NewClient builds a Client from config and options, defaulting to
// http.DefaultClient and a silent logger when none are supplied.
func NewClient(config Config, opts ...Option) Client {
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

	h := httpClient{
		client:  http.DefaultClient,
		baseURL: config.BaseURL,
		headers: config.Headers,
		logger:  nopLogger{},
	}
	for _, opt := range opts {
		opt(&h)
	}
	return h
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

// logArgs returns a fresh slice of base followed by extra. It copies so the
// base context can be reused across goroutines without aliasing its backing
// array.
func logArgs(base []any, extra ...any) []any {
	out := make([]any, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
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

	h.logger.Trace("http-client", "type", "request", "method", method, "headers", headers, "url", path, "query", req.Query, "body", string(body))

	var httpReq *http.Request
	// prepare request
	if body != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, path, bytes.NewBuffer(body))
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, path, http.NoBody)
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

	// a streamed request hands the live body back to the caller unread, so large
	// downloads never round-trip through memory. The caller owns Close.
	if req.Stream {
		h.logger.Trace("http-client", "type", "response", "method", method, "url", path, "status", resp.StatusCode, "body", "<streamed>")
		return &Response{
			StatusCode: resp.StatusCode,
			Reader:     resp.Body,
			Headers:    resp.Header,
		}, nil
	}

	defer resp.Body.Close()

	// read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	h.logger.Trace("http-client", "type", "response", "method", method, "url", path, "status", resp.StatusCode, "body", string(data))

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

	logCtx := []any{"type", "stream-request", "method", method, "url", path, "query", req.Query, "req-body", limitBodySize(body, size)}

	h.logger.Debug("http-client", logCtx...)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")

	//nolint:bodyclose // body is closed by the deferred close in the non-OK branch below and in the consumer goroutine on success
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	logCtx = logArgs(logCtx, "status", resp.StatusCode)

	h.logger.Debug("http-client", logArgs(logCtx, "request", "sent")...)

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		defer close(stream)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to read error response body: %w", err)
			h.logger.Error("http-client", logArgs(logCtx, "error", err)...)
			return err
		}

		err = fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, string(respBody))

		h.logger.Error("http-client", logArgs(logCtx, "stream", "started-with-error", "error", err)...)

		stream <- StreamResponse{
			Type:       StreamResponseTypeEOF,
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Error:      err,
			Body:       respBody,
		}

		return err
	}

	h.logger.Debug("http-client", logArgs(logCtx, "stream", "started")...)

	go func() {
		defer resp.Body.Close()
		defer close(stream)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			h.logger.Debug("http-client", logArgs(logCtx, "raw-line", string(line))...)
			if err != nil {
				stream <- StreamResponse{
					Type:       StreamResponseTypeEOF,
					StatusCode: resp.StatusCode,
					Headers:    resp.Header,
					Error:      fmt.Errorf("failed to read response body: %w", err),
				}
				h.logger.Error("http-client", logArgs(logCtx, "stream", "ended-with-error", "error", err)...)
				break
			}

			resType := StreamResponseTypeData
			line = bytes.TrimSpace(line)

			if len(line) == 0 {
				continue
			}
			h.logger.Debug("http-client", logArgs(logCtx, "type", resType, "pre-processed-line", string(line))...)
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
			h.logger.Debug("http-client", logArgs(logCtx, "type", resType, "sse-processed-line", string(line))...)
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
