package escaper

import (
	"compress/gzip"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/golang/gddo/httputil"
)

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// ForHTTP returns an Escaper for an HTTP request. It compresses the response
// as specified in the Accept-Encoding header, and sets the Content-Type and
// Content-Encoding headers appropriately. The returned Closer must be closed
// before the HTTP handler returns.
func ForHTTP(w http.ResponseWriter, r *http.Request) (*Escaper, io.Closer) {
	var dest io.Writer = w
	var closer io.Closer = nopCloser{}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if w.Header().Get("Content-Encoding") == "" {
		encoding := httputil.NegotiateContentEncoding(r, []string{"br", "gzip"})
		switch encoding {
		case "br":
			bw := brotli.NewWriter(dest, brotli.WriterOptions{Quality: 5})
			dest = bw
			closer = bw
			w.Header().Set("Content-Encoding", "br")
		case "gzip":
			gw := gzip.NewWriter(dest)
			dest = gw
			closer = gw
			w.Header().Set("Content-Encoding", "gzip")
		}
	}

	return New(dest), closer
}
