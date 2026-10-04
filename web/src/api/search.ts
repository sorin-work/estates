import { api } from './client';
import type { SearchHit } from './types';

export const search = (q: string, signal?: AbortSignal) => api<SearchHit[]>('/search', { query: { q }, signal });
