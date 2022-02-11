package gzip

import (
	"bytes"
	"compress/gzip"

	"github.com/gin-gonic/gin"
)

const (
	BestCompression    = gzip.BestCompression
	BestSpeed          = gzip.BestSpeed
	DefaultCompression = gzip.DefaultCompression
	NoCompression      = gzip.NoCompression
)

func Gzip(level int, options ...Option) gin.HandlerFunc {
	return newGzipHandler(level, options...).Handle
}

type gzipWriter struct {
	gin.ResponseWriter
	writer    *gzip.Writer
	buffer    bytes.Buffer
	minLength int
	compress  bool
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	g.Header().Del("Content-Length")
	return g.writer.Write([]byte(s))
}

func (g *gzipWriter) Write(data []byte) (w int, err error) {

	if !g.compress {
		w, err = g.writer.Write(data)
		return
	}

	// If the first chunk of data is already bigger than the minimum size,
	// set the headers and write directly to the gz writer
	if len(data) >= g.minLength {
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")

		g.compress = true
	}

	// Write the data into a buffer
	w, err = g.buffer.Write(data)
	if err != nil {
		return
	}

	// If the buffer is bigger than the minimum size, set the headers and write
	// the buffered data into the gz writer
	if g.buffer.Len() >= g.minLength {
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")

		_, err = g.writer.Write(g.buffer.Bytes())
		g.compress = true
	}

	return

}

// Fix: https://github.com/mholt/caddy/issues/38
func (g *gzipWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}
