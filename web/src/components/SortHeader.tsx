import type { SortState } from '../lib/sort';

/** A column header that sorts the table when clicked. */
export default function SortHeader<K extends string>({
  label,
  column,
  sort,
  onSort,
  className,
}: {
  label: string;
  column: K;
  sort: SortState<K>;
  onSort: (column: K) => void;
  className?: string;
}) {
  const active = sort.key === column;
  return (
    <th
      scope="col"
      className={className}
      aria-sort={active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : undefined}
    >
      <button type="button" className="sort-button" onClick={() => onSort(column)}>
        {label}
        <span aria-hidden="true">{active ? (sort.dir === 'asc' ? ' ▲' : ' ▼') : ''}</span>
      </button>
    </th>
  );
}
