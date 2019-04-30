package escaper

import (
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
)

// ForHTTP returns an Escaper for an HTTP request. It compresses the response
// as specified in the Accept-Encoding header, and sets the Content-Type and
// Content-Encoding headers appropriately. The returned Closer must be closed
// before the HTTP handler returns.
func ForHTTP(w http.ResponseWriter, r *http.Request) (*Escaper, io.Closer) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c := brotli.HTTPCompressor(w, r)
	return New(c), c
}
