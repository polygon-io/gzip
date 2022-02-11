package gzip

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type gzipHandler struct {
	*Options
	gzPool sync.Pool
}

func newGzipHandler(level int, options ...Option) *gzipHandler {
	handler := &gzipHandler{
		Options: DefaultOptions,
		gzPool: sync.Pool{
			New: func() interface{} {
				gz, err := gzip.NewWriterLevel(ioutil.Discard, level)
				if err != nil {
					panic(err)
				}
				return gz
			},
		},
	}
	for _, setter := range options {
		setter(handler.Options)
	}
	return handler
}

func (g *gzipHandler) Handle(c *gin.Context) {
	if fn := g.DecompressFn; fn != nil && c.Request.Header.Get("Content-Encoding") == "gzip" {
		fn(c)
	}

	if !g.shouldCompress(c.Request) {
		return
	}

	gz := g.gzPool.Get().(*gzip.Writer)
	defer g.gzPool.Put(gz)
	gz.Reset(c.Writer)

	gzWriter := &gzipWriter{
		ResponseWriter: c.Writer,
		writer:         gz,
		minLength:      g.Options.MinLength,
	}
	c.Writer = gzWriter

	c.Next()

	if gzWriter.compress {
		// Just close and flush the gz writer
		if err := gz.Close(); err != nil {
			_ = c.Error(fmt.Errorf("closing and flushing gzip writer: %w", err))
		}
	} else {
		// Reset to the original writer
		gz.Reset(ioutil.Discard)

		// Write the buffered data into the original writer
		if _, err := gzWriter.ResponseWriter.Write(gzWriter.buffer.Bytes()); err != nil {
			_ = c.Error(fmt.Errorf("closing and flushing gzip writer: %w", err))
		}
	}

	// Set the content length if it's still possible
	c.Header("Content-Length", fmt.Sprint(c.Writer.Size()))
}

func (g *gzipHandler) shouldCompress(req *http.Request) bool {
	if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") ||
		strings.Contains(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		return false
	}

	extension := filepath.Ext(req.URL.Path)
	if g.ExcludedExtensions.Contains(extension) {
		return false
	}

	if g.ExcludedPaths.Contains(req.URL.Path) {
		return false
	}
	if g.ExcludedPathesRegexs.Contains(req.URL.Path) {
		return false
	}

	return true
}
