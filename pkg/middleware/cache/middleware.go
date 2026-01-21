package cache

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Options struct {
	TTL    time.Duration
	Prefix string
}

type cachedResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func Middleware(cache Cache, opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		key := buildKey(c, opts.Prefix)

		if data, ok := cache.Get(c.Request.Context(), key); ok {
			var cached cachedResponse
			if err := decode(data, &cached); err == nil {
				for k, v := range cached.Header {
					for _, val := range v {
						c.Header(k, val)
					}
				}
				c.Header("X-Cache-Status", "HIT")
				c.Data(cached.Status, cached.Header.Get("Content-Type"), cached.Body)
				c.Abort()
				return
			}
		}

		w := responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = w
		c.Next()

		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			cached := cachedResponse{
				Status: c.Writer.Status(),
				Header: c.Writer.Header().Clone(),
				Body:   w.body.Bytes(),
			}
			if data, err := encode(&cached); err == nil {
				_ = cache.Set(c.Request.Context(), key, data, opts.TTL)
			}
		}
		c.Header("X-Cache-Status", "MISS")
	}
}

func buildKey(c *gin.Context, prefix string) string {
	var sb strings.Builder

	if prefix != "" {
		sb.WriteString(prefix)
		sb.WriteString(":")
	}
	sb.WriteString(c.Request.Method)
	sb.WriteString(":")
	sb.WriteString(c.Request.URL.Path)

	query := c.Request.URL.Query()
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for k := range query {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sb.WriteString("?")
		for i, k := range keys {
			if i > 0 {
				sb.WriteString("&")
			}
			values := query[k]
			sort.Strings(values)
			for j, v := range values {
				if j > 0 {
					sb.WriteString("&")
				}
				sb.WriteString(k)
				sb.WriteString("=")
				sb.WriteString(v)
			}
		}
	}
	return sb.String()
}

func decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func encode(v any) ([]byte, error) {
	return json.Marshal(v)
}
