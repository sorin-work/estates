import { api, download } from './client';
import type { Document, DocumentTarget } from './types';

/** The largest file the API accepts. */
export const MAX_DOCUMENT_BYTES = 10 * 1024 * 1024;

export const listDocuments = (target: DocumentTarget) => api<Document[]>('/documents', { query: { ...target } });

export function uploadDocument(target: DocumentTarget, file: File, notes: string) {
  const form = new FormData();
  for (const [key, value] of Object.entries(target)) form.set(key, value);
  form.set('notes', notes);
  form.set('file', file);
  return api<Document>('/documents', { method: 'POST', form });
}

export const downloadDocument = (doc: Document) => download(`/documents/${doc.id}/download`, undefined, doc.filename);
export const deleteDocument = (id: string) => api<void>(`/documents/${id}`, { method: 'DELETE' });
