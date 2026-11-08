import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import SearchBox, { hitPath } from './SearchBox';

const hits = [
  { kind: 'property', id: 'p1', title: 'Maple Street Duplex', detail: '2218 SE Maple St, Portland' },
  { kind: 'unit', id: 'u1', title: 'Maple Street Duplex · A', detail: 'Unit', property_id: 'p1' },
  { kind: 'tenant', id: 't1', title: 'Maria Gonzalez', detail: 'maria.gonzalez@example.com' },
];

describe('SearchBox', () => {
  it('links each kind of hit to its page', () => {
    expect(hitPath({ kind: 'unit', id: 'u1', title: '', detail: '', property_id: 'p1' })).toBe('/properties/p1');
    expect(hitPath({ kind: 'vendor', id: 'v1', title: '', detail: '' })).toBe('/vendors/v1');
  });

  it('opens a result with the keyboard', async () => {
    const calls = mockFetch({ 'GET /api/search': { body: hits } });
    renderWithProviders(
      <Routes>
        <Route path="/" element={<SearchBox />} />
        <Route path="/tenants/:id" element={<div>Tenant page</div>} />
      </Routes>,
    );

    const box = screen.getByRole('combobox', { name: 'Search' });
    await userEvent.type(box, 'ma');
    expect(await screen.findByRole('option', { name: /Maria Gonzalez/ })).toBeInTheDocument();
    expect(calls[0]?.url).toBe('/api/search?q=ma');

    await userEvent.keyboard('{ArrowDown}{ArrowDown}{ArrowDown}');
    expect(screen.getByRole('option', { name: /Maria Gonzalez/ })).toHaveAttribute('aria-selected', 'true');
    expect(box).toHaveAttribute('aria-activedescendant', screen.getByRole('option', { name: /Maria/ }).id);
    await userEvent.keyboard('{Enter}');
    expect(await screen.findByText('Tenant page')).toBeInTheDocument();
  });

  it('closes on Escape and says when nothing matches', async () => {
    mockFetch({ 'GET /api/search': { body: [] } });
    renderWithProviders(<SearchBox />);
    await userEvent.type(screen.getByRole('combobox'), 'zzz');
    expect(await screen.findByText('No matches')).toBeInTheDocument();
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  });

  it('is focused by pressing /', async () => {
    mockFetch({});
    renderWithProviders(
      <>
        <SearchBox />
        <input aria-label="Notes" />
      </>,
    );
    await userEvent.keyboard('/');
    expect(screen.getByRole('combobox')).toHaveFocus();

    // Typing a slash in another field is left alone.
    await userEvent.click(screen.getByLabelText('Notes'));
    await userEvent.keyboard('a/b');
    expect(screen.getByLabelText('Notes')).toHaveValue('a/b');
  });
});
