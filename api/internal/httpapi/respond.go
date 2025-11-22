package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// apiError is an error with an HTTP status and a stable code the web app can
// switch on.
type apiError struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *apiError) Error() string { return e.Message }

func errBadRequest(msg string) error {
	return &apiError{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

func errNotFound(what string) error {
	return &apiError{Status: http.StatusNotFound, Code: "not_found", Message: what + " not found"}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": apiError{Code: code, Message: msg}})
}

// handlerFunc is an http.HandlerFunc that returns an error instead of writing
// it, so every handler reports errors the same way.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handle(fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err == nil {
			return
		}

		var apiErr *apiError
		var conflictErr *store.ConflictError
		switch {
		case errors.As(err, &apiErr):
			writeJSON(w, apiErr.Status, map[string]any{"error": apiErr})
		case errors.As(err, &conflictErr):
			writeError(w, http.StatusConflict, "conflict", conflictErr.Msg)
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "not found")
		default:
			s.log.Error("request failed",
				"method", r.Method, "path", r.URL.Path,
				"request_id", middleware.GetReqID(r.Context()), "err", err)
			writeError(w, http.StatusInternalServerError, "internal", "something went wrong")
		}
	}
}

const maxBodyBytes = 1 << 20

// decode reads a JSON request body into dst.
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxErr *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return errBadRequest("request body is empty")
		case errors.As(err, &syntaxErr):
			return errBadRequest(fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset))
		case errors.As(err, &typeErr):
			return errBadRequest(fmt.Sprintf("%s has the wrong type", typeErr.Field))
		case errors.As(err, &maxErr):
			return errBadRequest("request body is too large")
		default:
			return errBadRequest(err.Error())
		}
	}
	return nil
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool { return uuidRE.MatchString(s) }

// pathID returns the {id} URL parameter. Anything that is not a UUID cannot
// exist, so it is reported as not found rather than a database error.
func pathID(r *http.Request, what string) (string, error) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		return "", errNotFound(what)
	}
	return id, nil
}

// queryID returns an optional UUID query parameter.
func queryID(r *http.Request, name string) (string, error) {
	v := strings.TrimSpace(r.URL.Query().Get(name))
	if v != "" && !isUUID(v) {
		return "", errBadRequest(name + " must be a valid id")
	}
	return v, nil
}

// queryDate returns an optional YYYY-MM-DD query parameter.
func queryDate(r *http.Request, name string) (domain.Date, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return domain.Date{}, nil
	}
	d, err := domain.ParseDate(v)
	if err != nil {
		return domain.Date{}, errBadRequest(name + ": " + err.Error())
	}
	return d, nil
}

// queryInt returns an optional non-negative integer query parameter.
func queryInt(r *http.Request, name string, fallback int) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, errBadRequest(name + " must be a non-negative integer")
	}
	return n, nil
}

// notFoundAs turns store.ErrNotFound into a not found error naming the
// record, and passes other errors through.
func notFoundAs(err error, what string) error {
	if errors.Is(err, store.ErrNotFound) {
		return errNotFound(what)
	}
	return err
}
