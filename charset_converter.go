package charset_converter

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

// Rewrite holds one rewrite body configuration.
type Charset struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// Config holds the plugin configuration.
type Config struct {
	Charset Charset `json:"charset,omitempty"`
}

// CreateConfig creates a new instance of the plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Charset: Charset{
			From: "utf-8",
			To:   "utf-8",
		},
	}
}

// Utf8ConverterMiddleware is the plugin middleware.
// Utf8ConverterMiddleware is the plugin middleware.
type Utf8ConverterMiddleware struct {
	next    http.Handler
	charset Charset // Use Charset struct directly
	name    string
}

// New creates a new instance of the Utf8ConverterMiddleware plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Utf8ConverterMiddleware{
		next:    next,
		charset: config.Charset, // Use config.Charset to access Charset struct
		name:    name,
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
		originalContentType += fmt.Sprintf("; charset=%s", m.charset.To)
		w.Header().Set("Content-Type", originalContentType)
	}

	// Convert the response to UTF-8
	utf8Response, err := convertToUTF8(rw.body.Bytes(), m.charset.From, m.charset.To)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write UTF-8 response back to original ResponseWriter
	w.WriteHeader(rw.statusCode)
	w.Write(utf8Response)
}

// custom ResponseWriter to capture response
// custom ResponseWriter to capture response
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.body == nil {
		rw.body = new(bytes.Buffer)
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// convert input to UTF-8
func convertToUTF8(input []byte, sourceEncoding string, targetEncoding string) ([]byte, error) {
	// Read the bytes from the input.
	reader := bytes.NewReader(input)
	inputBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Use charset.DetermineEncoding to detect the encoding of the input data.
	_, encodingName, _ := charset.DetermineEncoding(inputBytes, "")

	//TODO: hardcoding for utf-8 as first version
	// Check if the detected encoding is UTF-8.
	if strings.ToLower(encodingName) == strings.ToLower("utf-8") {
		// The input data is already in UTF-8 encoding
		return inputBytes, nil
	}

	var enc encoding.Encoding

	// TODO: add more charset
	switch sourceEncoding {
	case "ISO-8859-1":
		enc = charmap.ISO8859_1
	case "Windows-1252":
		enc = charmap.Windows1252
	default:
		enc = encoding.Nop
	}

	// Reader to decode input from source encoding
	decoderReader := enc.NewDecoder().Reader(bytes.NewReader(inputBytes))

	// Read the decoded bytes.
	utf8Bytes, err := ioutil.ReadAll(decoderReader)
	if err != nil {
		return nil, err
	}

	return utf8Bytes, nil
}
