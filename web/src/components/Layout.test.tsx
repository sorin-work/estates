import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { AuthProvider } from '../auth/AuthContext';
import { mockFetch, renderWithProviders } from '../test/render';
import Layout from './Layout';

describe('Layout', () => {
  it('counts the reminders that need attention next to the dashboard link', async () => {
    mockFetch({
      'GET /api/reminders': {
        body: [
          { kind: 'lease_ending', severity: 'urgent', title: '', detail: '', date: '2026-11-30' },
          { kind: 'maintenance', severity: 'warning', title: '', detail: '', date: '2026-11-09' },
          { kind: 'move_in', severity: 'info', title: '', detail: '', date: '2026-12-01' },
        ],
      },
    });
    renderWithProviders(
      <AuthProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<p>Home</p>} />
          </Route>
        </Routes>
      </AuthProvider>,
    );
    expect(await screen.findByLabelText('2 things need attention')).toHaveTextContent('2');
    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#content');
  });
});
