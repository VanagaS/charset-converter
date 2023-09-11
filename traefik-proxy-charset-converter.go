package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// Config the plugin configuration.
type Config struct {
	SourceEncoding string `json:"sourceEncoding"`
}

// CreateConfig creates a new instance of the plugin configuration.
func CreateConfig() *Config {
	return &Config{
		SourceEncoding: "ISO-8859-1", // default source encoding
	}
}

// Utf8ConverterMiddleware is the plugin middleware.
type Utf8ConverterMiddleware struct {
	next           http.Handler
	sourceEncoding string
	name           string
}

// New creates a new instance of the Utf8ConverterMiddleware plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Utf8ConverterMiddleware{
		next:           next,
		sourceEncoding: config.SourceEncoding,
		name:           name,
	}, nil
}

// ServeHTTP handles incoming HTTP requests.
func (m *Utf8ConverterMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a new ResponseWriter to capture the response
	rw := &responseWriter{ResponseWriter: w}

	// Continue the request chain
	m.next.ServeHTTP(rw, r)

	// Get original Content-Type header from response
	originalContentType := w.Header().Get("Content-Type")

	// Update Content-Type header with charset if it doesn't include one
	if !strings.Contains(strings.ToLower(originalContentType), "charset") {
		originalContentType += fmt.Sprintf("; charset=%s", m.sourceEncoding)
		w.Header().Set("Content-Type", originalContentType)
	}

	// Convert the response to UTF-8
	utf8Response, err := convertToUTF8(rw.Bytes(), m.sourceEncoding)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write UTF-8 response back to original ResponseWriter
	w.WriteHeader(rw.statusCode)
	w.Write(utf8Response)
}

// custom ResponseWriter to capture response
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}

// convert input to UTF-8
func convertToUTF8(input []byte, sourceEncoding string) ([]byte, error) {
	// Use the charset.DetermineEncoding function to detect the encoding of the input data.
	reader := bytes.NewReader(input)
	_, encodingName, _ := charset.DetermineEncoding(reader, "")

	// Check if the detected encoding is UTF-8.
	if strings.ToLower(encodingName) == strings.ToLower("utf-8") {
		// The input data is already in UTF-8 encoding
		return input, nil
	}

	var enc encoding.Encoding
	switch sourceEncoding {
	case "ISO-8859-1":
		enc = charmap.ISO8859_1
		// TODO: add more charset
	default:
		return nil, fmt.Errorf("unsupported source encoding: %s", sourceEncoding)
	}

	// Reader to decode input from source encoding
	reader = enc.NewDecoder().Reader(bytes.NewReader(input))

	// Read the decoded bytes.
	utf8Bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return utf8Bytes, nil
}
