import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { getDashboard } from '../api/dashboard';
import IncomeChart from '../components/charts/IncomeChart';
import PageHeader from '../components/PageHeader';
import Reminders from '../components/Reminders';
import StatCard from '../components/StatCard';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { PriorityBadge } from '../components/StatusBadge';
import { daysBetween, formatDate, formatMonthLong, relativeDays } from '../lib/dates';
import { formatMoney } from '../lib/money';

export default function DashboardPage() {
  const { data, isPending, error } = useQuery({ queryKey: ['dashboard'], queryFn: getDashboard });

  if (isPending) return <Loading />;
  if (error) return <ErrorState error={error} />;

  if (data.occupancy.units === 0) {
    return (
      <>
        <PageHeader title="Dashboard" />
        <EmptyState title="Welcome to Estates">
          Start by <Link to="/properties">adding a property</Link> and its units, then add tenants and leases. Rent
          charges are created automatically from each lease.
        </EmptyState>
      </>
    );
  }

  const { occupancy, this_month: month, outstanding, maintenance } = data;
  const collectionRate = month.charged_cents > 0 ? month.collected_cents / month.charged_cents : 1;

  return (
    <>
      <PageHeader title="Dashboard" subtitle={`Overview for ${formatMonthLong(month.month)}`} />
      <Reminders />

      <div className="stats">
        <StatCard
          label="Occupancy"
          value={`${Math.round(occupancy.rate * 100)}%`}
          detail={`${occupancy.occupied} of ${occupancy.units} units let · ${occupancy.vacant} vacant`}
        />
        <StatCard
          label="Collected this month"
          value={formatMoney(month.collected_cents)}
          detail={`of ${formatMoney(month.charged_cents)} charged (${Math.round(collectionRate * 100)}%)`}
          tone={collectionRate >= 1 ? 'good' : undefined}
        />
        <StatCard
          label="Overdue"
          value={formatMoney(outstanding.overdue_cents)}
          detail={`${outstanding.overdue_leases} ${outstanding.overdue_leases === 1 ? 'lease' : 'leases'} behind · ${formatMoney(outstanding.balance_cents)} owed in total`}
          tone={outstanding.overdue_cents > 0 ? 'bad' : 'good'}
        />
        <StatCard
          label="Net this month"
          value={formatMoney(month.collected_cents - month.expenses_cents)}
          detail={`${formatMoney(month.expenses_cents)} in expenses`}
        />
        <StatCard
          label="Deposits held"
          value={formatMoney(data.deposits.held_cents)}
          detail={`for ${data.deposits.leases} ${data.deposits.leases === 1 ? 'lease' : 'leases'}, not income`}
        />
      </div>

      <section className="card">
        <div className="card-header">
          <h2>Last 12 months</h2>
        </div>
        <IncomeChart months={data.monthly} />
      </section>

      <div className="grid-2">
        <section className="card">
          <div className="card-header">
            <h2>Overdue rent</h2>
            <Link to="/rent">Rent roll</Link>
          </div>
          {data.overdue.length === 0 ? (
            <p className="muted">Everyone is up to date.</p>
          ) : (
            <ul className="list">
              {data.overdue.map((b) => (
                <li key={b.lease.id}>
                  <Link to={`/leases/${b.lease.id}`}>
                    <strong>{b.lease.tenant_name}</strong>
                    <span className="muted">
                      {b.lease.property_name} · {b.lease.unit_label}
                    </span>
                  </Link>
                  <span className="list-end">
                    <strong className="text-bad">{formatMoney(b.overdue_cents)}</strong>
                    <span className="muted">{b.days_overdue} days</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="card">
          <div className="card-header">
            <h2>Leases ending soon</h2>
            <Link to="/leases?status=active">All leases</Link>
          </div>
          {data.expiring_leases.length === 0 ? (
            <p className="muted">No leases end in the next 60 days.</p>
          ) : (
            <ul className="list">
              {data.expiring_leases.map((l) => {
                const end = l.terminated_on && l.terminated_on < l.end_date ? l.terminated_on : l.end_date;
                return (
                  <li key={l.id}>
                    <Link to={`/leases/${l.id}`}>
                      <strong>{l.tenant_name}</strong>
                      <span className="muted">
                        {l.property_name} · {l.unit_label}
                      </span>
                    </Link>
                    <span className="list-end">
                      <span>{formatDate(end)}</span>
                      <span className="muted">{relativeDays(daysBetween(data.as_of, end))}</span>
                    </span>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      </div>

      <section className="card">
        <div className="card-header">
          <h2>Open maintenance</h2>
          <Link to="/maintenance">{maintenance.open} open</Link>
        </div>
        <div className="priority-counts">
          {(['urgent', 'high', 'normal', 'low'] as const).map((p) => (
            <span key={p}>
              <PriorityBadge priority={p} /> {maintenance.by_priority[p] ?? 0}
            </span>
          ))}
        </div>
        {maintenance.recent.length === 0 ? (
          <p className="muted">Nothing needs attention.</p>
        ) : (
          <ul className="list">
            {maintenance.recent.map((m) => (
              <li key={m.id}>
                <Link to={`/maintenance?property_id=${m.property_id}`}>
                  <strong>{m.title}</strong>
                  <span className="muted">
                    {m.property_name}
                    {m.unit_label ? ` · ${m.unit_label}` : ''} · reported {formatDate(m.reported_on)}
                  </span>
                </Link>
                <PriorityBadge priority={m.priority} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}
