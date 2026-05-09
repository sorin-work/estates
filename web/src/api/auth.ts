import { api } from './client';
import type { AuthResponse, User } from './types';

export function login(email: string, password: string) {
  return api<AuthResponse>('/auth/login', { method: 'POST', body: { email, password } });
}

export function register(input: { name: string; email: string; password: string }) {
  return api<AuthResponse>('/auth/register', { method: 'POST', body: input });
}

export function me() {
  return api<User>('/auth/me');
}

export function updateMe(input: { name: string; currency: string }) {
  return api<User>('/auth/me', { method: 'PUT', body: input });
}

export function changePassword(input: { current_password: string; new_password: string }) {
  return api<void>('/auth/password', { method: 'PUT', body: input });
}
