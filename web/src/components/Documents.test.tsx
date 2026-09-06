import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import Documents from './Documents';

const doc = {
  id: 'd1',
  tenant_id: 't1',
  lease_id: null,
  property_id: null,
  filename: 'Drivers licence.jpg',
  content_type: 'image/jpeg',
  size_bytes: 245760,
  notes: 'Front and back',
  created_at: '2026-09-28T19:12:00Z',
};

describe('Documents', () => {
  it('lists files and uploads a new one as a form', async () => {
    const calls = mockFetch({
      'GET /api/documents': { body: [doc] },
      'POST /api/documents': { status: 201, body: { ...doc, id: 'd2', filename: 'Application.pdf' } },
    });
    renderWithProviders(<Documents target={{ tenant_id: 't1' }} />);

    expect(await screen.findByRole('button', { name: 'Drivers licence.jpg' })).toBeInTheDocument();
    expect(screen.getByText(/240\.0 KB · Sep 28, 2026 · Front and back/)).toBeInTheDocument();

    const file = new File(['%PDF-1.7'], 'Application.pdf', { type: 'application/pdf' });
    await userEvent.upload(screen.getByLabelText('File'), file);
    await userEvent.type(screen.getByLabelText('Notes'), 'Signed');
    await userEvent.click(screen.getByRole('button', { name: 'Upload' }));

    const form = calls.find((c) => c.method === 'POST')?.body as FormData;
    expect(form.get('tenant_id')).toBe('t1');
    expect(form.get('notes')).toBe('Signed');
    expect((form.get('file') as File).name).toBe('Application.pdf');
    // The form is cleared once the upload is done.
    expect(await screen.findByRole('button', { name: 'Upload' })).toBeDisabled();
    expect(screen.getByLabelText('Notes')).toHaveValue('');
  });

  it('refuses files over the limit before uploading', async () => {
    mockFetch({ 'GET /api/documents': { body: [] } });
    renderWithProviders(<Documents target={{ lease_id: 'l1' }} hint="Nothing here" />);
    await screen.findByText('Nothing here');

    const huge = new File(['x'], 'scan.pdf', { type: 'application/pdf' });
    Object.defineProperty(huge, 'size', { value: 11 * 1024 * 1024 });
    await userEvent.upload(screen.getByLabelText('File'), huge);
    expect(screen.getByRole('alert')).toHaveTextContent('Files can be at most 10.0 MB.');
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled();
  });
});
