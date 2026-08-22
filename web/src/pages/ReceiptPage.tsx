import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';
import { getReceipt } from '../api/rent';
import { ErrorState, Loading } from '../components/States';
import { formatDate } from '../lib/dates';
import { paymentMethods } from '../lib/labels';
import { formatMoney } from '../lib/money';

/** A printable rent receipt. It sits outside the app layout so it prints cleanly. */
export default function ReceiptPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { data, isPending, error } = useQuery({ queryKey: ['receipt', id], queryFn: () => getReceipt(id) });

  if (isPending) return <Loading />;
  if (error) return <ErrorState error={error} />;
  const { payment: p, property: prop } = data;
  const place = [prop.city, [prop.region, prop.postal_code].filter(Boolean).join(' ')].filter(Boolean).join(', ');

  return (
    <div className="receipt-page">
      <div className="receipt-actions no-print">
        <button type="button" className="button secondary" onClick={() => navigate(-1)}>
          Back
        </button>
        <button type="button" className="button" onClick={() => window.print()}>
          Print
        </button>
      </div>
      <article className="receipt" aria-labelledby="receipt-title">
        <header className="receipt-header">
          <div>
            <h1 id="receipt-title">Rent receipt</h1>
            <div className="muted">No. {String(data.number).padStart(5, '0')}</div>
          </div>
          <div className="receipt-from">
            <strong>{data.landlord.name}</strong>
            <div>{data.landlord.email}</div>
          </div>
        </header>

        <dl className="receipt-details">
          <dt>Received from</dt>
          <dd>{data.tenant.name}</dd>
          <dt>For</dt>
          <dd>
            {prop.name}, unit {prop.unit_label}
            <div className="muted">
              {prop.address_line1}
              {prop.address_line2 && `, ${prop.address_line2}`}
              {place && `, ${place}`}
            </div>
          </dd>
          <dt>Date paid</dt>
          <dd>{formatDate(p.paid_on)}</dd>
          <dt>Method</dt>
          <dd>
            {paymentMethods[p.method]}
            {p.reference && ` (${p.reference})`}
          </dd>
          {p.notes && (
            <>
              <dt>Notes</dt>
              <dd>{p.notes}</dd>
            </>
          )}
        </dl>

        <div className="receipt-amount">
          <span>Amount received</span>
          <strong>{formatMoney(p.amount_cents)}</strong>
        </div>
        <p className="muted">
          {data.balance_after_cents > 0
            ? `Balance still owed after this payment: ${formatMoney(data.balance_after_cents)}.`
            : data.balance_after_cents < 0
              ? `The tenant is ${formatMoney(-data.balance_after_cents)} in credit after this payment.`
              : 'The account was fully paid up after this payment.'}
        </p>
        <footer className="muted small">Issued {formatDate(data.issued_on)}</footer>
      </article>
    </div>
  );
}
