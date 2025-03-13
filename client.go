package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	
	"github.com/toaweme/log"
)

type Client interface {
	Get(req GetRequest) (*Response, error)
	Post(req PostRequest) (*Response, error)
	Patch(req PatchRequest) (*Response, error)
	Delete(req Request) (*Response, error)
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type Request struct {
	ID      string
	Path    string
	Query   url.Values
	Headers map[string]string
}

type GetRequest struct {
	Request
}

type PostRequest struct {
	Request
	
	Body []byte
}

type PatchRequest PostRequest

type httpClient struct {
	baseURL string
	agent   string
	
	client  *http.Client
	headers map[string]string
}

func NewHttpClient(baseURL, agent string, headers map[string]string) Client {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers[log.ClientAgentHeaderName] = agent
	
	return httpClient{
		baseURL: baseURL,
		client:  http.DefaultClient,
		headers: headers,
	}
}

func (h httpClient) Get(req GetRequest) (*Response, error) {
	return h.do(http.MethodGet, req.Request, nil)
}

func (h httpClient) Post(req PostRequest) (*Response, error) {
	return h.do(http.MethodPost, req.Request, req.Body)
}

func (h httpClient) Patch(req PatchRequest) (*Response, error) {
	return h.do(http.MethodPatch, req.Request, req.Body)
}

func (h httpClient) Delete(req Request) (*Response, error) {
	return h.do(http.MethodDelete, req, nil)
}

func (h httpClient) do(method string, req Request, body []byte) (*Response, error) {
	log.Debug("http request", "method", method, "path", req.Path, "query", req.Query, "headers", req.Headers, "body", string(body))
	
	path, headers, err := h.buildRequestParams(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request URI: %w", err)
	}
	
	var httpReq *http.Request
	// prepare request
	if body != nil {
		httpReq, err = http.NewRequest(method, path, bytes.NewBuffer(body))
	} else {
		httpReq, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create DELETE request: %w", err)
	}
	
	// set headers
	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}
	
	// send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()
	
	// read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	log.Debug("http response", "status", resp.StatusCode, "headers", resp.Header, "body", string(data))
	
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
		headers[log.ClientIDHeaderName] = req.ID
	} else {
		headers[log.ClientIDHeaderName] = log.ID()
	}
	
	// prepare URL
	path, err := url.JoinPath(h.baseURL, req.Path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to join URL: %s: %w", req.Path, err)
	}
	
	// prepare query
	query := req.Query.Encode()
	if query != "" {
		path += "?" + query
	}
	
	return path, headers, nil
}
