import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link } from 'react-router';
import { exportPayments, listBalances, listPayments, runBilling } from '../api/rent';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { formatDate, formatMonthLong, monthRange, today } from '../lib/dates';
import { paymentMethods } from '../lib/labels';
import { formatMoney } from '../lib/money';

const PAGE_SIZE = 25;

export default function RentPage() {
  const queryClient = useQueryClient();
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [month, setMonth] = useState(today().slice(0, 7));
  const [page, setPage] = useState(0);

  const balances = useQuery({ queryKey: ['balances', overdueOnly], queryFn: () => listBalances(overdueOnly) });
  const range = monthRange(month);
  const exportCsv = useMutation({ mutationFn: () => exportPayments(range) });
  const payments = useQuery({
    queryKey: ['payments', { month, page }],
    queryFn: () => listPayments({ ...range, limit: PAGE_SIZE, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  const run = useMutation({
    mutationFn: runBilling,
    onSuccess: () => {
      for (const key of ['balances', 'ledger', 'dashboard']) {
        void queryClient.invalidateQueries({ queryKey: [key] });
      }
    },
  });

  const totalOwed = balances.data?.reduce((sum, b) => sum + Math.max(0, b.balance_cents), 0) ?? 0;
  const totalPages = payments.data ? Math.max(1, Math.ceil(payments.data.total / PAGE_SIZE)) : 1;

  return (
    <>
      <PageHeader
        title="Rent"
        subtitle="Who owes what, and what has come in."
        actions={
          <button type="button" className="button secondary" disabled={run.isPending} onClick={() => run.mutate()}>
            {run.isPending ? 'Updating…' : 'Update charges now'}
          </button>
        }
      />
      {run.data && (
        <div className="notice">
          Added {run.data.rent_charges} rent {run.data.rent_charges === 1 ? 'charge' : 'charges'} and{' '}
          {run.data.late_fees} late {run.data.late_fees === 1 ? 'fee' : 'fees'}.
        </div>
      )}

      <section className="card">
        <div className="card-header">
          <h2>Rent roll</h2>
          <label className="checkbox">
            <input type="checkbox" checked={overdueOnly} onChange={(e) => setOverdueOnly(e.target.checked)} />
            Only overdue
          </label>
        </div>
        {balances.isPending && <Loading />}
        {balances.error && <ErrorState error={balances.error} />}
        {balances.data && balances.data.length === 0 && (
          <EmptyState title={overdueOnly ? 'Nothing overdue' : 'No active leases'} />
        )}
        {balances.data && balances.data.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Tenant</th>
                <th>Property</th>
                <th className="num">Rent</th>
                <th className="num">Balance</th>
                <th className="num">Overdue</th>
                <th>Late since</th>
              </tr>
            </thead>
            <tbody>
              {balances.data.map((b) => (
                <tr key={b.lease.id} className={b.overdue_cents > 0 ? 'row-bad' : undefined}>
                  <td>
                    <Link to={`/leases/${b.lease.id}`}>{b.lease.tenant_name}</Link>
                  </td>
                  <td>
                    {b.lease.property_name} · {b.lease.unit_label}
                  </td>
                  <td className="num">{formatMoney(b.lease.current_rent_cents)}</td>
                  <td className="num">
                    {b.balance_cents < 0 ? (
                      <span className="text-good">{formatMoney(-b.balance_cents)} credit</span>
                    ) : (
                      formatMoney(b.balance_cents)
                    )}
                  </td>
                  <td className="num">{b.overdue_cents > 0 ? formatMoney(b.overdue_cents) : '—'}</td>
                  <td>{b.oldest_overdue ? `${formatDate(b.oldest_overdue)} (${b.days_overdue}d)` : '—'}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={3}>Total owed</td>
                <td className="num">{formatMoney(totalOwed)}</td>
                <td colSpan={2} />
              </tr>
            </tfoot>
          </table>
        )}
      </section>

      <section className="card">
        <div className="card-header">
          <h2>Payments in {formatMonthLong(month)}</h2>
          <button
            type="button"
            className="link-button"
            disabled={exportCsv.isPending}
            onClick={() => exportCsv.mutate()}
          >
            Export CSV
          </button>
          <input
            type="month"
            aria-label="Month"
            value={month}
            onChange={(e) => {
              if (!e.target.value) return;
              setMonth(e.target.value);
              setPage(0);
            }}
          />
        </div>
        {payments.isPending && <Loading />}
        {payments.data && payments.data.payments.length === 0 && <p className="muted">No payments this month.</p>}
        {payments.data && payments.data.payments.length > 0 && (
          <>
            <table className="table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Tenant</th>
                  <th>Property</th>
                  <th>Method</th>
                  <th className="num">Amount</th>
                </tr>
              </thead>
              <tbody>
                {payments.data.payments.map((p) => (
                  <tr key={p.id}>
                    <td>{formatDate(p.paid_on)}</td>
                    <td>
                      <Link to={`/leases/${p.lease_id}`}>{p.tenant_name}</Link>
                    </td>
                    <td>
                      {p.property_name} · {p.unit_label}
                    </td>
                    <td>
                      {paymentMethods[p.method]}
                      {p.reference && <span className="muted"> {p.reference}</span>}
                    </td>
                    <td className="num">{formatMoney(p.amount_cents)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="pager">
              <span className="muted">
                {payments.data.total} {payments.data.total === 1 ? 'payment' : 'payments'}
              </span>
              <button
                type="button"
                className="button small secondary"
                disabled={page === 0}
                onClick={() => setPage(page - 1)}
              >
                Previous
              </button>
              <span>
                Page {page + 1} of {totalPages}
              </span>
              <button
                type="button"
                className="button small secondary"
                disabled={page + 1 >= totalPages}
                onClick={() => setPage(page + 1)}
              >
                Next
              </button>
            </div>
          </>
        )}
      </section>
    </>
  );
}
