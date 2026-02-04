import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';
import type { ReactElement } from 'react';
import { MemoryRouter } from 'react-router';
import { vi } from 'vitest';

export function renderWithProviders(ui: ReactElement, { route = '/' }: { route?: string } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

/** Replaces fetch with a function that answers by "METHOD /path". */
export function mockFetch(routes: Record<string, { status?: number; body?: unknown }>) {
  const calls: { method: string; url: string; body: unknown }[] = [];
  const fn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? 'GET';
    calls.push({ method, url, body: init?.body ? JSON.parse(String(init.body)) : undefined });
    const path = url.split('?')[0] ?? url;
    const route = routes[`${method} ${path}`];
    if (!route) {
      return new Response(JSON.stringify({ error: { code: 'not_found', message: `no mock for ${method} ${path}` } }), {
        status: 404,
      });
    }
    return new Response(route.status === 204 ? null : JSON.stringify(route.body ?? {}), {
      status: route.status ?? 200,
      headers: { 'Content-Type': 'application/json' },
    });
  });
  vi.stubGlobal('fetch', fn);
  return calls;
}
