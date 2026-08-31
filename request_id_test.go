package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == "" || id2 == "" {
		t.Fatalf("expected non-empty request IDs")
	}
	if id1 == id2 {
		t.Fatalf("expected unique request IDs, got duplicate: %s", id1)
	}
}

func TestExtractOrGenerateRequestID_GeneratesNewID(t *testing.T) {
	rawReq := []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	id, modifiedReq := ExtractOrGenerateRequestID(rawReq)

	if id == "" {
		t.Fatalf("expected generated request ID")
	}
	if !strings.Contains(string(modifiedReq), "X-Request-ID: "+id) {
		t.Fatalf("expected modified request to contain X-Request-ID header, got:\n%s", string(modifiedReq))
	}
}

func TestExtractOrGenerateRequestID_PreservesExistingID(t *testing.T) {
	existingID := "custom-client-id-12345"
	rawReq := []byte("GET / HTTP/1.1\r\nHost: localhost\r\nX-Request-ID: " + existingID + "\r\n\r\n")
	id, modifiedReq := ExtractOrGenerateRequestID(rawReq)

	if id != existingID {
		t.Fatalf("expected preserved ID %s, got %s", existingID, id)
	}
	if !bytes.Equal(rawReq, modifiedReq) {
		t.Fatalf("expected raw request to be unchanged when client supplies ID")
	}
}

func TestInjectResponseRequestID(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	reqID := "test-id-789"

	InjectResponseRequestID(resp, reqID)

	if resp.Header.Get(HeaderXRequestID) != reqID {
		t.Fatalf("expected response header %s to be %s, got %s", HeaderXRequestID, reqID, resp.Header.Get(HeaderXRequestID))
	}
}
