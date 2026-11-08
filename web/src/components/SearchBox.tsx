import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type KeyboardEvent } from 'react';
import { useNavigate } from 'react-router';
import { search } from '../api/search';
import type { SearchHit } from '../api/types';
import { useDebounced } from '../lib/hooks';

const kindLabels: Record<SearchHit['kind'], string> = {
  property: 'Property',
  unit: 'Unit',
  tenant: 'Tenant',
  vendor: 'Vendor',
};

export function hitPath(hit: SearchHit): string {
  switch (hit.kind) {
    case 'property':
      return `/properties/${hit.id}`;
    case 'unit':
      return `/properties/${hit.property_id ?? ''}`;
    case 'tenant':
      return `/tenants/${hit.id}`;
    case 'vendor':
      return `/vendors/${hit.id}`;
  }
}

/**
 * Quick search with a listbox of results, following the ARIA combobox
 * pattern: arrow keys move through the results, Enter opens one and
 * Escape closes the list.
 */
export default function SearchBox({ onNavigate }: { onNavigate?: () => void }) {
  const navigate = useNavigate();
  const listId = useId();
  const [text, setText] = useState('');
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const q = useDebounced(text.trim(), 200);
  const input = useRef<HTMLInputElement>(null);

  // "/" jumps to the search box from anywhere that is not a form field.
  useEffect(() => {
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey) return;
      const t = e.target as HTMLElement | null;
      if (t && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName))) return;
      e.preventDefault();
      input.current?.focus();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);

  const { data: hits = [] } = useQuery({
    queryKey: ['search', q],
    queryFn: ({ signal }) => search(q, signal),
    enabled: q.length >= 2,
    placeholderData: keepPreviousData,
  });
  const showList = open && q.length >= 2;

  function go(hit: SearchHit) {
    navigate(hitPath(hit));
    setText('');
    setOpen(false);
    setActive(-1);
    onNavigate?.();
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setOpen(true);
      setActive((i) => Math.min(i + 1, hits.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActive((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter') {
      const hit = hits[active] ?? hits[0];
      if (showList && hit) {
        e.preventDefault();
        go(hit);
      }
    } else if (e.key === 'Escape') {
      setOpen(false);
      setActive(-1);
    }
  }

  return (
    <div className="search">
      <input
        ref={input}
        type="search"
        role="combobox"
        aria-label="Search"
        aria-expanded={showList}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={showList && active >= 0 ? `${listId}-${active}` : undefined}
        placeholder="Search…  /"
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          setOpen(true);
          setActive(-1);
        }}
        onKeyDown={onKeyDown}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
      />
      {showList && (
        <ul id={listId} role="listbox" className="search-results" aria-label="Search results">
          {hits.length === 0 && (
            <li className="search-empty" role="presentation">
              No matches
            </li>
          )}
          {hits.map((h, i) => (
            <li
              key={`${h.kind}-${h.id}`}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === active}
              className={i === active ? 'active' : undefined}
              // Keep focus in the input so the blur does not close the list first.
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => go(h)}
            >
              <span className="search-kind">{kindLabels[h.kind]}</span>
              <span className="search-title">{h.title}</span>
              {h.detail && <span className="search-detail">{h.detail}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
