import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRef, useState, type FormEvent } from 'react';
import { deleteDocument, downloadDocument, listDocuments, MAX_DOCUMENT_BYTES, uploadDocument } from '../api/documents';
import type { DocumentTarget } from '../api/types';
import { formatBytes } from '../lib/format';
import { FormError } from './States';

const dateFormat = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' });

/** Files attached to a tenant, lease or property, with an upload form. */
export default function Documents({ target, hint }: { target: DocumentTarget; hint?: string }) {
  const queryClient = useQueryClient();
  const queryKey = ['documents', target];
  const docs = useQuery({ queryKey, queryFn: () => listDocuments(target) });
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [notes, setNotes] = useState('');
  const [tooBig, setTooBig] = useState(false);

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey });
    void queryClient.invalidateQueries({ queryKey: ['activity'] });
  };
  const upload = useMutation({
    mutationFn: (f: File) => uploadDocument(target, f, notes),
    onSuccess: () => {
      refresh();
      setFile(null);
      setNotes('');
      if (fileInput.current) fileInput.current.value = '';
    },
  });
  const save = useMutation({ mutationFn: downloadDocument });
  const remove = useMutation({ mutationFn: deleteDocument, onSuccess: refresh });

  function submit(e: FormEvent) {
    e.preventDefault();
    if (file && !tooBig) upload.mutate(file);
  }

  return (
    <section className="card">
      <div className="card-header">
        <h2>Documents</h2>
      </div>
      {docs.data && docs.data.length === 0 && <p className="muted">{hint ?? 'No documents yet.'}</p>}
      {docs.data && docs.data.length > 0 && (
        <ul className="list">
          {docs.data.map((d) => (
            <li key={d.id}>
              <span>
                <button type="button" className="link-button doc-name" onClick={() => save.mutate(d)}>
                  {d.filename}
                </button>
                <span className="muted small">
                  {formatBytes(d.size_bytes)} · {dateFormat.format(new Date(d.created_at))}
                  {d.notes && ` · ${d.notes}`}
                </span>
              </span>
              <button
                type="button"
                className="link-button danger"
                aria-label={`Delete ${d.filename}`}
                onClick={() => {
                  if (confirm(`Delete ${d.filename}?`)) remove.mutate(d.id);
                }}
              >
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
      <FormError error={upload.error ?? save.error ?? remove.error} />
      <form className="upload" onSubmit={submit}>
        <input
          ref={fileInput}
          type="file"
          aria-label="File"
          accept=".pdf,.jpg,.jpeg,.png,.gif,.webp,.heic,.txt,.csv,.doc,.docx,.xls,.xlsx,.odt,.ods"
          onChange={(e) => {
            const f = e.target.files?.[0] ?? null;
            setFile(f);
            setTooBig(f !== null && f.size > MAX_DOCUMENT_BYTES);
          }}
        />
        <input
          type="text"
          aria-label="Notes"
          placeholder="Notes (optional)"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
        <button type="submit" className="button small" disabled={!file || tooBig || upload.isPending}>
          {upload.isPending ? 'Uploading…' : 'Upload'}
        </button>
      </form>
      {tooBig && (
        <div className="field-error" role="alert">
          Files can be at most {formatBytes(MAX_DOCUMENT_BYTES)}.
        </div>
      )}
    </section>
  );
}
