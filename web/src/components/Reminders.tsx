import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { getReminders } from '../api/dashboard';
import type { Reminder } from '../api/types';
import { formatDate } from '../lib/dates';

function target(r: Reminder): string {
  if (r.lease_id) return `/leases/${r.lease_id}`;
  if (r.maintenance_id) return '/maintenance?status=active';
  return '/';
}

const severityLabel = { urgent: 'Urgent', warning: 'Soon', info: 'Coming up' } as const;

/** The to-do list at the top of the dashboard. Hidden when there is nothing to do. */
export default function Reminders() {
  const { data } = useQuery({ queryKey: ['reminders'], queryFn: getReminders });
  if (!data || data.length === 0) return null;

  return (
    <section className="card reminders" aria-labelledby="reminders-title">
      <div className="card-header">
        <h2 id="reminders-title">To do</h2>
        <span className="muted">
          {data.length} {data.length === 1 ? 'reminder' : 'reminders'}
        </span>
      </div>
      <ul className="list">
        {data.map((r, i) => (
          <li key={`${r.kind}-${r.lease_id ?? r.maintenance_id ?? i}-${r.date}`} className={`reminder ${r.severity}`}>
            <Link to={target(r)}>
              <strong>{r.title}</strong>
              <span className="muted">{r.detail}</span>
            </Link>
            <span className="list-end">
              <span
                className={`badge badge-${r.severity === 'urgent' ? 'red' : r.severity === 'warning' ? 'amber' : 'blue'}`}
              >
                {severityLabel[r.severity]}
              </span>
              <span className="muted small">{formatDate(r.date)}</span>
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
