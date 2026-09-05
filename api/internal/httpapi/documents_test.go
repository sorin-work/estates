package httpapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sorin-work/estates/api/internal/store"
)

// upload posts a file as a multipart form.
func (c *client) upload(fields map[string]string, filename string, content []byte) response {
	c.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			c.t.Fatal(err)
		}
	}
	if filename != "" {
		fw, err := mw.CreateFormFile("file", filename)
		if err != nil {
			c.t.Fatal(err)
		}
		_, _ = fw.Write(content)
	}
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/api/documents", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func TestCleanFilename(t *testing.T) {
	for in, want := range map[string]string{
		"lease.pdf":                "lease.pdf",
		`C:\Users\me\scan "1".jpg`: "scan 1.jpg",
		"../../etc/passwd":         "passwd",
		"a\nb.txt":                 "ab.txt",
		"/":                        "",
	} {
		if got := cleanFilename(in); got != want {
			t.Errorf("cleanFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDocuments(t *testing.T) {
	c := newDBServer(t, "2026-10-01")
	l := newLease(t, c, nil)
	pdf := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")

	res := c.upload(map[string]string{"lease_id": l.ID, "notes": "Signed copy"}, "Lease 2026.pdf", pdf)
	if res.Status != http.StatusCreated {
		t.Fatalf("upload: %d %s", res.Status, res.Body)
	}
	var doc store.Document
	res.decode(t, &doc)
	if doc.Filename != "Lease 2026.pdf" || doc.ContentType != "application/pdf" || doc.SizeBytes != int64(len(pdf)) {
		t.Errorf("document %+v", doc)
	}

	// Scripts and pages are refused, as are uploads without a target.
	if res := c.upload(map[string]string{"lease_id": l.ID}, "x.html", []byte("<script>alert(1)</script>")); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("html upload: %d", res.Status)
	}
	if res := c.upload(map[string]string{}, "a.pdf", pdf); res.Status != http.StatusBadRequest {
		t.Errorf("no target: %d", res.Status)
	}
	if res := c.upload(map[string]string{"tenant_id": l.TenantID}, "", nil); res.Status != http.StatusBadRequest {
		t.Errorf("no file: %d", res.Status)
	}
	big := bytes.Repeat([]byte("a"), MaxDocumentBytes+1)
	if res := c.upload(map[string]string{"tenant_id": l.TenantID}, "big.txt", big); res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("too big: %d", res.Status)
	}

	var docs []store.Document
	c.mustStatus(http.StatusOK, "GET", "/api/documents?lease_id="+l.ID, nil).decode(t, &docs)
	if len(docs) != 1 || docs[0].Notes != "Signed copy" {
		t.Fatalf("docs %+v", docs)
	}
	c.mustStatus(http.StatusBadRequest, "GET", "/api/documents", nil)

	res = c.mustStatus(http.StatusOK, "GET", "/api/documents/"+doc.ID+"/download", nil)
	if !bytes.Equal(res.Body, pdf) || res.Header.Get("Content-Type") != "application/pdf" ||
		!strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment") ||
		res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("download headers %v", res.Header)
	}

	other := newDBServerFor(t, c, "other@example.com")
	other.mustStatus(http.StatusNotFound, "GET", "/api/documents/"+doc.ID+"/download", nil)
	if res := other.upload(map[string]string{"lease_id": l.ID}, "a.pdf", pdf); res.Status != http.StatusConflict {
		t.Errorf("upload to someone else's lease: %d", res.Status)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", "/api/documents/"+doc.ID, nil)
	c.mustStatus(http.StatusNotFound, "GET", "/api/documents/"+doc.ID+"/download", nil)
}
