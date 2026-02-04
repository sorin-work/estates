import { afterEach, describe, expect, it, vi } from 'vitest';
import { mockFetch } from '../test/render';
import { api, ApiError, buildPath, getToken, setToken, UNAUTHORIZED_EVENT } from './client';

afterEach(() => setToken(null));

describe('buildPath', () => {
  it('skips empty values', () => {
    expect(buildPath('/leases', { status: 'active', property_id: '', unit_id: undefined })).toBe(
      '/leases?status=active',
    );
    expect(buildPath('/leases')).toBe('/leases');
  });
});

describe('api', () => {
  it('sends the token and JSON body', async () => {
    setToken('abc');
    const calls = mockFetch({ 'POST /api/tenants': { status: 201, body: { id: 't1' } } });
    const res = await api<{ id: string }>('/tenants', { method: 'POST', body: { first_name: 'Ada' } });
    expect(res.id).toBe('t1');
    expect(calls[0]?.body).toEqual({ first_name: 'Ada' });
    const init = vi.mocked(fetch).mock.calls[0]?.[1];
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer abc');
  });

  it('turns error responses into ApiError with field errors', async () => {
    mockFetch({
      'POST /api/properties': {
        status: 422,
        body: {
          error: { code: 'validation_failed', message: 'some fields are invalid', fields: { name: 'is required' } },
        },
      },
    });
    const err = await api('/properties', { method: 'POST', body: {} }).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(422);
    expect((err as ApiError).fields).toEqual({ name: 'is required' });
  });

  it('clears the token and signals when the API says 401', async () => {
    setToken('expired');
    mockFetch({ 'GET /api/auth/me': { status: 401, body: { error: { code: 'unauthorized', message: 'nope' } } } });
    const listener = vi.fn();
    window.addEventListener(UNAUTHORIZED_EVENT, listener);
    await expect(api('/auth/me')).rejects.toThrow('nope');
    window.removeEventListener(UNAUTHORIZED_EVENT, listener);
    expect(getToken()).toBeNull();
    expect(listener).toHaveBeenCalledOnce();
  });

  it('reports network failures in plain words', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')));
    await expect(api('/properties')).rejects.toMatchObject({ code: 'network' });
  });

  it('returns undefined for 204 responses', async () => {
    mockFetch({ 'DELETE /api/units/u1': { status: 204 } });
    await expect(api('/units/u1', { method: 'DELETE' })).resolves.toBeUndefined();
  });
});
