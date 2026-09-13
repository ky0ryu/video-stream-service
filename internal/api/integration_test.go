package api_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
)

func TestVideoUpload_Integration(t *testing.T) {
	apiURL := "http://localhost:8080/upload"

	// build multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add fields
	_ = writer.WriteField("title", "Video Stream Service Integration Test")
	_ = writer.WriteField("description", "Testing the API using native Go tags")

	// custom part with custom header to satisfy file type sniffing
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.mp4`)
	h.Set("Content-Type", "video/mp4")
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create multipart file section: %v", err)
	}

	// write MP4 magic bytes to bypass the Sniff Validator
	magicBytes := []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6d, 0x70, 0x34, 0x32}
	_, _ = part.Write(magicBytes)
	_, _ = part.Write([]byte("mock-data"))
	_ = writer.Close()

	// issue the client HTTP call
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to fire call. Is server running? Error: %v", err)
	}
	defer resp.Body.Close()

	// verify status code
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201 Created, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
}
