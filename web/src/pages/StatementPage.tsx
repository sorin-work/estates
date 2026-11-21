import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';
import { getStatement } from '../api/rent';
import type { ChargeKind, PaymentMethod, StatementLine } from '../api/types';
import Field from '../components/Field';
import { ErrorState, Loading } from '../components/States';
import { formatDate } from '../lib/dates';
import { useFilterParams } from '../lib/hooks';
import { chargeKinds, paymentMethods } from '../lib/labels';
import { formatMoney } from '../lib/money';

function describe(line: StatementLine): string {
  if (line.payment) {
    const method = paymentMethods[line.kind as PaymentMethod] ?? line.kind;
    return `Payment, ${method.toLowerCase()}${line.description ? ` (${line.description})` : ''}`;
  }
  const kind = chargeKinds[line.kind as ChargeKind] ?? line.kind;
  return line.description ? `${kind}: ${line.description}` : kind;
}

/** What the balance means, in words. */
function owed(cents: number): string {
  if (cents > 0) return `${formatMoney(cents)} owed`;
  if (cents < 0) return `${formatMoney(-cents)} in credit`;
  return 'Nothing owed';
}

/**
 * A printable statement of a lease's account over a period. It sits
 * outside the app layout, like the receipt, so it prints cleanly. The
 * period is kept in the address.
 */
export default function StatementPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const [params, setFilter] = useFilterParams();
  const from = params.get('from') ?? '';
  const to = params.get('to') ?? '';
  const { data, isPending, error } = useQuery({
    queryKey: ['statement', id, from, to],
    queryFn: () => getStatement(id, { from, to }),
  });

  if (isPending) return <Loading />;
  if (error) return <ErrorState error={error} />;
  const prop = data.property;
  const place = [prop.city, [prop.region, prop.postal_code].filter(Boolean).join(' ')].filter(Boolean).join(', ');

  return (
    <div className="receipt-page">
      <div className="receipt-actions statement-actions no-print">
        <Field label="From">
          {(fid) => (
            <input id={fid} type="date" value={from || data.from} onChange={(e) => setFilter('from', e.target.value)} />
          )}
        </Field>
        <Field label="To">
          {(fid) => (
            <input id={fid} type="date" value={to || data.to} onChange={(e) => setFilter('to', e.target.value)} />
          )}
        </Field>
        <button type="button" className="button secondary" onClick={() => navigate(-1)}>
          Back
        </button>
        <button type="button" className="button" onClick={() => window.print()}>
          Print
        </button>
      </div>
      <article className="receipt" aria-labelledby="statement-title">
        <header className="receipt-header">
          <div>
            <h1 id="statement-title">Rent statement</h1>
            <div className="muted">
              {formatDate(data.from)} to {formatDate(data.to)}
            </div>
          </div>
          <div className="receipt-from">
            <strong>{data.landlord.name}</strong>
            <div>{data.landlord.email}</div>
          </div>
        </header>

        <dl className="receipt-details">
          <dt>Tenant</dt>
          <dd>{data.tenant.name}</dd>
          <dt>Property</dt>
          <dd>
            {prop.name}, unit {prop.unit_label}
            <div className="muted">
              {prop.address_line1}
              {prop.address_line2 && `, ${prop.address_line2}`}
              {place && `, ${place}`}
            </div>
          </dd>
          <dt>Lease</dt>
          <dd>
            {formatDate(data.lease.start_date)} to {formatDate(data.lease.end_date)}
          </dd>
        </dl>

        <div className="statement-lines">
          <table className="table">
            <caption className="visually-hidden">Charges and payments</caption>
            <thead>
              <tr>
                <th scope="col">Date</th>
                <th scope="col">Description</th>
                <th scope="col" className="num">
                  Charged
                </th>
                <th scope="col" className="num">
                  Paid
                </th>
                <th scope="col" className="num">
                  Balance
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>{formatDate(data.from)}</td>
                <td>Opening balance</td>
                <td />
                <td />
                <td className="num">{formatMoney(data.opening_cents)}</td>
              </tr>
              {data.lines.map((line) => (
                <tr key={`${line.payment ? 'p' : 'c'}-${line.id}`}>
                  <td>{formatDate(line.date)}</td>
                  <td>{describe(line)}</td>
                  <td className="num">{line.payment ? '' : formatMoney(line.amount_cents)}</td>
                  <td className="num">{line.payment ? formatMoney(line.amount_cents) : ''}</td>
                  <td className="num">{formatMoney(line.balance_cents)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <th scope="row" colSpan={2}>
                  Totals
                </th>
                <th className="num">{formatMoney(data.charged_cents)}</th>
                <th className="num">{formatMoney(data.paid_cents)}</th>
                <th className="num">{formatMoney(data.closing_cents)}</th>
              </tr>
            </tfoot>
          </table>
        </div>

        <div className="receipt-amount">
          <span>Balance on {formatDate(data.to)}</span>
          <strong>{owed(data.closing_cents)}</strong>
        </div>
        {data.lines.length === 0 && <p className="muted">No charges or payments in this period.</p>}
        <footer className="muted small">Issued {formatDate(data.issued_on)}</footer>
      </article>
    </div>
  );
}
