package store

import (
	"context"
	"time"
)

// Document is an uploaded file attached to a tenant, lease or property. The
// content is only loaded for downloads.
type Document struct {
	ID          string    `db:"id" json:"id"`
	TenantID    *string   `db:"tenant_id" json:"tenant_id"`
	LeaseID     *string   `db:"lease_id" json:"lease_id"`
	PropertyID  *string   `db:"property_id" json:"property_id"`
	Filename    string    `db:"filename" json:"filename"`
	ContentType string    `db:"content_type" json:"content_type"`
	SizeBytes   int64     `db:"size_bytes" json:"size_bytes"`
	Notes       string    `db:"notes" json:"notes"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// DocumentTarget is what a document is attached to; exactly one is set.
type DocumentTarget struct {
	TenantID   string
	LeaseID    string
	PropertyID string
}

type DocumentInput struct {
	DocumentTarget
	Filename    string
	ContentType string
	Content     []byte
	Notes       string
}

const documentColumns = `d.id, d.tenant_id, d.lease_id, d.property_id, d.filename, d.content_type, d.size_bytes, d.notes, d.created_at`

func (s *Store) ListDocuments(ctx context.Context, ownerID string, t DocumentTarget) ([]Document, error) {
	w := newFilter(ownerID)
	w.raw("d.owner_id = $1")
	if t.TenantID != "" {
		w.add("d.tenant_id = ?", t.TenantID)
	}
	if t.LeaseID != "" {
		w.add("d.lease_id = ?", t.LeaseID)
	}
	if t.PropertyID != "" {
		w.add("d.property_id = ?", t.PropertyID)
	}
	rows, err := s.pool.Query(ctx, `SELECT `+documentColumns+` FROM documents d`+w.where()+` ORDER BY d.created_at DESC`, w.args...)
	return collectAll[Document](rows, err)
}

// DocumentContent returns a document with its file.
func (s *Store) DocumentContent(ctx context.Context, ownerID, id string) (Document, []byte, error) {
	var d Document
	var content []byte
	err := s.pool.QueryRow(ctx, `
		SELECT d.id, d.filename, d.content_type, d.size_bytes, d.created_at, d.content
		  FROM documents d WHERE d.owner_id = $1 AND d.id = $2`, ownerID, id,
	).Scan(&d.ID, &d.Filename, &d.ContentType, &d.SizeBytes, &d.CreatedAt, &content)
	return d, content, notFound(err)
}

// CreateDocument stores a file. The tenant, lease or property it is
// attached to must belong to the owner.
func (s *Store) CreateDocument(ctx context.Context, ownerID string, in DocumentInput) (Document, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT CASE
		         WHEN $2 <> '' THEN EXISTS (SELECT 1 FROM tenants WHERE id::text = $2 AND owner_id = $1)
		         WHEN $3 <> '' THEN EXISTS (SELECT 1 FROM leases l`+ownedLease+` WHERE l.id::text = $3)
		         ELSE EXISTS (SELECT 1 FROM properties WHERE id::text = $4 AND owner_id = $1)
		       END`, ownerID, in.TenantID, in.LeaseID, in.PropertyID).Scan(&ok)
	if err != nil {
		return Document{}, err
	}
	if !ok {
		return Document{}, conflict("the record to attach this document to was not found")
	}
	rows, err := s.pool.Query(ctx, `
		INSERT INTO documents AS d (owner_id, tenant_id, lease_id, property_id, filename, content_type, size_bytes, content, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+documentColumns,
		ownerID, nullIfEmpty(in.TenantID), nullIfEmpty(in.LeaseID), nullIfEmpty(in.PropertyID),
		in.Filename, in.ContentType, len(in.Content), in.Content, in.Notes)
	return collectOne[Document](rows, err)
}

func (s *Store) DeleteDocument(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `DELETE FROM documents WHERE owner_id = $1 AND id = $2`, ownerID, id))
}
