package gzip

import (
	"compress/gzip"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// nolint:gochecknoglobals // It's okay that these are globals.
var (
	// DefaultExcludedExtentions is a list of file extensions to exlude from compression by default.
	DefaultExcludedExtentions = NewExcludedExtensions([]string{
		".png", ".gif", ".jpeg", ".jpg",
	})

	// DefaultOptions represents the options used to configure this middleware.
	DefaultOptions = &Options{
		ExcludedExtensions: DefaultExcludedExtentions,
	}
)

type Options struct {
	ExcludedExtensions   ExcludedExtensions
	ExcludedPaths        ExcludedPaths
	ExcludedPathesRegexs ExcludedPathesRegexs
	DecompressFn         func(c *gin.Context)
	MinLength            int
}

type Option func(*Options)

func WithExcludedExtensions(args []string) Option {
	return func(o *Options) {
		o.ExcludedExtensions = NewExcludedExtensions(args)
	}
}

func WithExcludedPaths(args []string) Option {
	return func(o *Options) {
		o.ExcludedPaths = NewExcludedPaths(args)
	}
}

func WithExcludedPathsRegexs(args []string) Option {
	return func(o *Options) {
		o.ExcludedPathesRegexs = NewExcludedPathesRegexs(args)
	}
}

func WithDecompressFn(decompressFn func(c *gin.Context)) Option {
	return func(o *Options) {
		o.DecompressFn = decompressFn
	}
}

func WithMinLength(minLength int) Option {
	return func(o *Options) {
		o.MinLength = minLength
	}
}

// ExcludedExtensions is a map of file extensions to exclude from compression.
type ExcludedExtensions map[string]bool

func NewExcludedExtensions(extensions []string) ExcludedExtensions {
	res := make(ExcludedExtensions)
	for _, e := range extensions {
		res[e] = true
	}
	return res
}

func (e ExcludedExtensions) Contains(target string) bool {
	_, ok := e[target]
	return ok
}

// ExcludedPaths is a list of paths to exclude from compression.
type ExcludedPaths []string

func NewExcludedPaths(paths []string) ExcludedPaths {
	return ExcludedPaths(paths)
}

func (e ExcludedPaths) Contains(requestURI string) bool {
	for _, path := range e {
		if strings.HasPrefix(requestURI, path) {
			return true
		}
	}
	return false
}

// ExcludedPathesRegexs is a list of regular expressions which match paths to exclude from compression.
type ExcludedPathesRegexs []*regexp.Regexp

func NewExcludedPathesRegexs(regexs []string) ExcludedPathesRegexs {
	result := make([]*regexp.Regexp, len(regexs))
	for i, reg := range regexs {
		result[i] = regexp.MustCompile(reg)
	}
	return result
}

func (e ExcludedPathesRegexs) Contains(requestURI string) bool {
	for _, reg := range e {
		if reg.MatchString(requestURI) {
			return true
		}
	}
	return false
}

func DefaultDecompressHandle(c *gin.Context) {
	if c.Request.Body == nil {
		return
	}
	r, err := gzip.NewReader(c.Request.Body)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.Request.Header.Del("Content-Encoding")
	c.Request.Header.Del("Content-Length")
	c.Request.Body = r
}
