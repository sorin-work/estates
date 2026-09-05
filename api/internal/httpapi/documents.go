package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/store"
)

type documentStore interface {
	ListDocuments(ctx context.Context, ownerID string, t store.DocumentTarget) ([]store.Document, error)
	DocumentContent(ctx context.Context, ownerID, id string) (store.Document, []byte, error)
	CreateDocument(ctx context.Context, ownerID string, in store.DocumentInput) (store.Document, error)
	DeleteDocument(ctx context.Context, ownerID, id string) error
}

// MaxDocumentBytes is the largest file that can be uploaded.
const MaxDocumentBytes = 10 << 20

// documentTypes are the files accepted, by extension. Anything that a
// browser would run, like HTML or SVG, is refused.
var documentTypes = map[string]string{
	".pdf":  "application/pdf",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".heic": "image/heic",
	".txt":  "text/plain; charset=utf-8",
	".csv":  "text/csv; charset=utf-8",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
}

func (s *Server) documentRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listDocuments))
	r.Post("/", s.handle(s.uploadDocument))
	r.Get("/{id}/download", s.handle(s.downloadDocument))
	r.Delete("/{id}", s.handle(s.deleteDocument))
}

func documentTarget(get func(string) string) (store.DocumentTarget, error) {
	t := store.DocumentTarget{
		TenantID:   strings.TrimSpace(get("tenant_id")),
		LeaseID:    strings.TrimSpace(get("lease_id")),
		PropertyID: strings.TrimSpace(get("property_id")),
	}
	set := 0
	for _, id := range []string{t.TenantID, t.LeaseID, t.PropertyID} {
		if id == "" {
			continue
		}
		if !isUUID(id) {
			return t, errBadRequest("not a valid id: " + id)
		}
		set++
	}
	if set != 1 {
		return t, errBadRequest("give exactly one of tenant_id, lease_id or property_id")
	}
	return t, nil
}

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) error {
	t, err := documentTarget(r.URL.Query().Get)
	if err != nil {
		return err
	}
	docs, err := s.store.ListDocuments(r.Context(), userID(r), t)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, docs)
	return nil
}

// cleanFilename keeps the base name of an upload and drops characters that
// would break a Content-Disposition header.
func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == '"' || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if name == "." || name == "/" {
		return ""
	}
	if utf8.RuneCountInString(name) > 200 {
		ext := filepath.Ext(name)
		name = string([]rune(name)[:200-len([]rune(ext))]) + ext
	}
	return name
}

func (s *Server) uploadDocument(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, MaxDocumentBytes+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errTooLarge()
		}
		return errBadRequest("expected a multipart form with a file")
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	t, err := documentTarget(r.FormValue)
	if err != nil {
		return err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return errBadRequest("no file was uploaded")
	}
	defer file.Close()

	var v validator
	name := cleanFilename(header.Filename)
	contentType, known := documentTypes[strings.ToLower(filepath.Ext(name))]
	v.check(name != "", "file", "has no name")
	v.check(known, "file", "must be a PDF, image, text or office document")
	v.check(header.Size <= MaxDocumentBytes, "file", fmt.Sprintf("must be at most %d MB", MaxDocumentBytes>>20))
	notes := strings.TrimSpace(r.FormValue("notes"))
	v.maxLen(notes, "notes", 1000)
	if err := v.err(); err != nil {
		return err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		v.check(false, "file", "is empty")
		return v.err()
	}

	doc, err := s.store.CreateDocument(r.Context(), userID(r), store.DocumentInput{
		DocumentTarget: t, Filename: name, ContentType: contentType, Content: content, Notes: notes,
	})
	if err != nil {
		return err
	}
	s.audit(r, "document", doc.ID, t.LeaseID, "created", "Uploaded "+doc.Filename)
	writeJSON(w, http.StatusCreated, doc)
	return nil
}

func errTooLarge() error {
	return &apiError{Status: http.StatusRequestEntityTooLarge, Code: "too_large",
		Message: fmt.Sprintf("files can be at most %d MB", MaxDocumentBytes>>20)}
}

// downloadDocument always sends files as attachments, so an uploaded file is
// never shown inside the app's origin.
func (s *Server) downloadDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "document")
	if err != nil {
		return err
	}
	doc, content, err := s.store.DocumentContent(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "document")
	}
	h := w.Header()
	h.Set("Content-Type", doc.ContentType)
	h.Set("Content-Length", strconv.Itoa(len(content)))
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": doc.Filename})
	if disposition == "" {
		disposition = "attachment"
	}
	h.Set("Content-Disposition", disposition)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "private, no-store")
	_, err = w.Write(content)
	return err
}

func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "document")
	if err != nil {
		return err
	}
	if err := s.store.DeleteDocument(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "document")
	}
	s.audit(r, "document", id, "", "deleted", "Deleted a document")
	w.WriteHeader(http.StatusNoContent)
	return nil
}
