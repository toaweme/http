package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"
)

// SlogConfig controls what the SlogMiddleware emits on each request.
// LogRequestBody and LogResponseBody are off by default — they're useful for
// local debugging but should stay off in prod (PII, large payloads, perf).
type SlogConfig struct {
	// LogRequestBody captures and logs the request body.
	LogRequestBody bool
	// LogResponseBody captures and logs the response body.
	LogResponseBody bool
	// LogRequestHeaders logs incoming request headers.
	LogRequestHeaders bool
	// LogResponseHeaders logs outgoing response headers (captured at end of request).
	LogResponseHeaders bool
	// MaxBodyBytes caps how much of each body is captured. 0 means no cap.
	MaxBodyBytes int
}

// SlogMiddleware logs method, url and duration for every request, plus
// optionally the request and response bodies when the config opts in.
func SlogMiddleware(cfg SlogConfig, logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			var reqBody []byte
			if cfg.LogRequestBody && r.Body != nil {
				reqBody, r.Body = readAndReplace(r.Body, cfg.MaxBodyBytes)
			}

			rw := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			if cfg.LogResponseBody {
				rw.buf = &bytes.Buffer{}
				rw.maxBytes = cfg.MaxBodyBytes
			}

			next.ServeHTTP(rw, r)

			args := []any{
				"method", r.Method,
				"url", r.URL.RequestURI(),
				"duration", time.Since(start).String(),
				"code", rw.status,
			}
			if cfg.LogRequestHeaders {
				args = append(args, "request-headers", flattenHeaders(r.Header))
			}
			if cfg.LogRequestBody {
				args = append(args, "request-body", string(reqBody))
			}
			if cfg.LogResponseHeaders {
				args = append(args, "response-headers", flattenHeaders(rw.Header()))
			}
			if cfg.LogResponseBody && rw.buf != nil {
				args = append(args, "response-body", rw.buf.String())
			}

			logger.Info("http", args...)
		})
	}
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 1 {
			out[k] = v[0]
			continue
		}
		out[k] = strings.Join(v, ", ")
	}
	return out
}

func readAndReplace(body io.ReadCloser, maxBytes int) ([]byte, io.ReadCloser) {
	defer body.Close()
	var (
		buf []byte
		err error
	)
	if maxBytes > 0 {
		buf, err = io.ReadAll(io.LimitReader(body, int64(maxBytes)))
	} else {
		buf, err = io.ReadAll(body)
	}
	if err != nil {
		return nil, io.NopCloser(bytes.NewReader(nil))
	}
	// drain any remainder so the handler still sees a full, untruncated body
	rest, _ := io.ReadAll(body)
	full := append(buf, rest...)
	return buf, io.NopCloser(bytes.NewReader(full))
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	buf         *bytes.Buffer
	maxBytes    int
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	if r.buf != nil {
		remaining := len(b)
		if r.maxBytes > 0 {
			if room := r.maxBytes - r.buf.Len(); room > 0 {
				if remaining > room {
					remaining = room
				}
				r.buf.Write(b[:remaining])
			}
		} else {
			r.buf.Write(b)
		}
	}
	return r.ResponseWriter.Write(b)
}
