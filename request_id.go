package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const HeaderXRequestID = "X-Request-ID"

// GenerateRequestID generates a random 16-byte hex string for correlation
func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ExtractOrGenerateRequestID extracts X-Request-ID header from raw HTTP request or generates one
func ExtractOrGenerateRequestID(rawRequest []byte) (string, []byte) {
	lines := strings.Split(string(rawRequest), "\r\n")
	var existingID string
	hasRequestIDHeader := false

	for i, line := range lines {
		if i == 0 {
			continue // Request line
		}
		if line == "" {
			break // End of headers
		}
		if strings.HasPrefix(strings.ToLower(line), "x-request-id:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				existingID = strings.TrimSpace(parts[1])
				hasRequestIDHeader = true
			}
			break
		}
	}

	if hasRequestIDHeader && existingID != "" {
		return existingID, rawRequest
	}

	// Generate new ID and inject into rawRequest headers
	requestID := GenerateRequestID()

	emptyLineIdx := strings.Index(string(rawRequest), "\r\n\r\n")
	if emptyLineIdx != -1 {
		headerInjection := fmt.Sprintf("X-Request-ID: %s\r\n", requestID)
		newRaw := append([]byte(string(rawRequest[:emptyLineIdx+2])), append([]byte(headerInjection), rawRequest[emptyLineIdx+2:]...)...)
		return requestID, newRaw
	}

	return requestID, rawRequest
}

// InjectResponseRequestID ensures X-Request-ID header is present in the response
func InjectResponseRequestID(resp *http.Response, requestID string) {
	if resp != nil {
		if resp.Header == nil {
			resp.Header = make(http.Header)
		}
		resp.Header.Set(HeaderXRequestID, requestID)
	}
}
