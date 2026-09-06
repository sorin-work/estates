const TOKEN_KEY = 'estates.token';

let token: string | null = readToken();

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function getToken(): string | null {
  return token;
}

export function setToken(value: string | null): void {
  token = value;
  try {
    if (value) localStorage.setItem(TOKEN_KEY, value);
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    // Private mode or storage disabled: keep the token in memory only.
  }
}

/** Fired when the API rejects our token so the app can return to login. */
export const UNAUTHORIZED_EVENT = 'estates:unauthorized';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: Record<string, string>;

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

type Query = Record<string, string | number | boolean | undefined | null>;

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  /** A multipart form, for file uploads. Sent instead of body. */
  form?: FormData;
  query?: Query;
  signal?: AbortSignal;
}

export function buildPath(path: string, query?: Query): string {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue;
    params.set(key, String(value));
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

async function send(path: string, options: RequestOptions, accept: string): Promise<Response> {
  const headers: Record<string, string> = { Accept: accept };
  if (options.body !== undefined && !options.form) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;

  let body: BodyInit | undefined;
  if (options.form) body = options.form;
  else if (options.body !== undefined) body = JSON.stringify(options.body);

  let res: Response;
  try {
    res = await fetch(`/api${buildPath(path, options.query)}`, {
      method: options.method ?? 'GET',
      headers,
      body,
      signal: options.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiError(0, 'network', 'Could not reach the server. Check your connection and try again.');
  }
  if (res.ok) return res;

  const data: unknown = await res.json().catch(() => null);
  const error = (data as { error?: { code?: string; message?: string; fields?: Record<string, string> } } | null)
    ?.error;
  if (res.status === 401 && token) {
    setToken(null);
    window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
  }
  throw new ApiError(
    res.status,
    error?.code ?? 'unknown',
    error?.message ?? `Request failed (${res.status})`,
    error?.fields ?? {},
  );
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const res = await send(path, options, 'application/json');
  if (res.status === 204) return undefined as T;
  return (await res.json().catch(() => null)) as T;
}

/** The filename from a Content-Disposition header, if it has one. */
export function filenameFrom(header: string | null, fallback: string): string {
  const match = header?.match(/filename="?([^";]+)"?/);
  return match?.[1] ?? fallback;
}

/**
 * Fetches a file with the signed in user's token and hands it to the
 * browser as a download. A plain link would not send the token.
 */
export async function download(path: string, query?: Query, fallbackName = 'download'): Promise<void> {
  const res = await send(path, { query }, '*/*');
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filenameFrom(res.headers.get('Content-Disposition'), fallbackName);
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
