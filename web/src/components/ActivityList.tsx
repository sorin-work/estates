import type { Activity } from '../api/types';

const timeFormat = new Intl.DateTimeFormat('en-US', { hour: 'numeric', minute: '2-digit' });
const dayFormat = new Intl.DateTimeFormat('en-US', {
  weekday: 'long',
  month: 'short',
  day: 'numeric',
  year: 'numeric',
});

/** Groups entries by the local day they happened on, keeping their order. */
export function groupByDay(entries: Activity[]): { day: string; entries: Activity[] }[] {
  const groups: { day: string; entries: Activity[] }[] = [];
  for (const a of entries) {
    const day = dayFormat.format(new Date(a.created_at));
    const last = groups.at(-1);
    if (last && last.day === day) last.entries.push(a);
    else groups.push({ day, entries: [a] });
  }
  return groups;
}

export default function ActivityList({ entries }: { entries: Activity[] }) {
  return (
    <div className="activity">
      {groupByDay(entries).map((g) => (
        <section key={g.day}>
          <h3 className="activity-day">{g.day}</h3>
          <ul className="list compact">
            {g.entries.map((a) => (
              <li key={a.id}>
                <span>{a.summary}</span>
                <time className="muted small" dateTime={a.created_at}>
                  {timeFormat.format(new Date(a.created_at))}
                </time>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
