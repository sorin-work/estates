import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router';
import { exportProfitLoss, getProfitLoss } from '../api/reports';
import type { PropertyResult } from '../api/types';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, FormError, Loading } from '../components/States';
import { formatDate, today } from '../lib/dates';
import { expenseCategories, options } from '../lib/labels';
import { formatMoney } from '../lib/money';
import { periodPresets, presetRange, type PeriodPreset } from '../lib/periods';

function Row({ p, total }: { p: PropertyResult; total?: boolean }) {
  const Cell = total ? 'th' : 'td';
  return (
    <tr className={total ? 'total' : undefined}>
      <Cell scope="row">{p.property_name}</Cell>
      <Cell className="num">{formatMoney(p.charged_cents)}</Cell>
      <Cell className="num">{formatMoney(p.collected_cents)}</Cell>
      <Cell className="num">{p.withheld_cents ? formatMoney(p.withheld_cents) : '—'}</Cell>
      <Cell className="num">{formatMoney(p.expenses_cents)}</Cell>
      <Cell className={p.net_cents < 0 ? 'num negative' : 'num'}>{formatMoney(p.net_cents)}</Cell>
    </tr>
  );
}

export default function ReportsPage() {
  const [params, setParams] = useSearchParams();
  const preset = (params.get('period') ?? 'this-year') as PeriodPreset;
  const period =
    preset === 'custom'
      ? { from: params.get('from') ?? `${today().slice(0, 4)}-01-01`, to: params.get('to') ?? today() }
      : presetRange(preset, today());

  const report = useQuery({
    queryKey: ['reports', 'profit-loss', period],
    queryFn: () => getProfitLoss(period),
    placeholderData: keepPreviousData,
  });
  const exportCsv = useMutation({ mutationFn: () => exportProfitLoss(period) });

  function set(next: Record<string, string>) {
    setParams({ period: preset, ...(preset === 'custom' ? period : {}), ...next }, { replace: true });
  }

  const data = report.data;
  const top = data?.totals.by_category[0]?.amount_cents ?? 0;

  return (
    <>
      <PageHeader
        title="Profit and loss"
        subtitle={data && `${formatDate(data.from)} – ${formatDate(data.to)}`}
        actions={
          <>
            <button type="button" className="button secondary" onClick={() => window.print()}>
              Print
            </button>
            <button
              type="button"
              className="button secondary"
              disabled={exportCsv.isPending}
              onClick={() => exportCsv.mutate()}
            >
              Export CSV
            </button>
          </>
        }
      />
      <div className="toolbar no-print">
        <select aria-label="Period" value={preset} onChange={(e) => set({ period: e.target.value })}>
          {options(periodPresets).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        {preset === 'custom' && (
          <>
            <label className="inline">
              From <input type="date" value={period.from} onChange={(e) => set({ from: e.target.value })} />
            </label>
            <label className="inline">
              To <input type="date" value={period.to} onChange={(e) => set({ to: e.target.value })} />
            </label>
          </>
        )}
      </div>
      <FormError error={exportCsv.error} />
      {report.isPending && <Loading />}
      {report.error && <ErrorState error={report.error} />}
      {data && data.properties.length === 0 && (
        <EmptyState title="Nothing to report yet">Add a property with leases and expenses first.</EmptyState>
      )}
      {data && data.properties.length > 0 && (
        <div className="grid-sidebar">
          <div className="card flush">
            <table className="table">
              <caption className="visually-hidden">Income and expenses by property</caption>
              <thead>
                <tr>
                  <th scope="col">Property</th>
                  <th scope="col" className="num">
                    Charged
                  </th>
                  <th scope="col" className="num">
                    Collected
                  </th>
                  <th scope="col" className="num">
                    Deposits kept
                  </th>
                  <th scope="col" className="num">
                    Expenses
                  </th>
                  <th scope="col" className="num">
                    Net
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.properties.map((p) => (
                  <Row key={p.property_id} p={p} />
                ))}
              </tbody>
              <tfoot>
                <Row p={data.totals} total />
              </tfoot>
            </table>
          </div>
          <aside className="card">
            <div className="stat-label">Expenses by category</div>
            {data.totals.by_category.length === 0 && <p className="muted">No expenses in this period.</p>}
            <ul className="bars">
              {data.totals.by_category.map((c) => (
                <li key={c.category}>
                  <div className="bar-label">
                    <span>{expenseCategories[c.category]}</span>
                    <span>{formatMoney(c.amount_cents)}</span>
                  </div>
                  <div className="bar-track">
                    <div className="bar-fill" style={{ width: `${(c.amount_cents / top) * 100}%` }} />
                  </div>
                </li>
              ))}
            </ul>
          </aside>
        </div>
      )}
    </>
  );
}
