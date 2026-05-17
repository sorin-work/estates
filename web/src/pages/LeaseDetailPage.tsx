import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import {
  addCharge,
  deleteCharge,
  deleteDepositTransaction,
  deleteLease,
  getDeposit,
  getLease,
  getLedger,
  recordDeposit,
  recordPayment,
  terminateLease,
  updateLease,
} from '../api/leases';
import { deletePayment } from '../api/rent';
import type { ChargeInput, DepositInput, LeaseInput, PaymentInput } from '../api/types';
import ChargeForm from '../components/forms/ChargeForm';
import DepositForm from '../components/forms/DepositForm';
import LeaseForm from '../components/forms/LeaseForm';
import PaymentForm from '../components/forms/PaymentForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import StatCard from '../components/StatCard';
import { ErrorState, FormError, Loading } from '../components/States';
import { ChargeStatusBadge, LeaseStatusBadge } from '../components/StatusBadge';
import { formatDate, today } from '../lib/dates';
import { chargeKinds, depositKinds, paymentMethods } from '../lib/labels';
import { formatMoney } from '../lib/money';

type Dialog = 'edit' | 'payment' | 'charge' | 'terminate' | 'deposit' | null;

function ordinal(n: number): string {
  const s = ['th', 'st', 'nd', 'rd'];
  const v = n % 100;
  return n + (s[(v - 20) % 10] ?? s[v] ?? s[0] ?? 'th');
}

export default function LeaseDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [dialog, setDialog] = useState<Dialog>(null);
  const [terminateOn, setTerminateOn] = useState(today());

  const lease = useQuery({ queryKey: ['leases', 'detail', id], queryFn: () => getLease(id) });
  const ledger = useQuery({ queryKey: ['ledger', id], queryFn: () => getLedger(id) });
  const deposit = useQuery({ queryKey: ['deposit', id], queryFn: () => getDeposit(id) });

  const refresh = () => {
    for (const key of ['leases', 'ledger', 'deposit', 'payments', 'balances', 'dashboard', 'units', 'properties']) {
      void queryClient.invalidateQueries({ queryKey: [key] });
    }
  };
  const close = () => setDialog(null);

  const update = useMutation({
    mutationFn: (input: LeaseInput) => updateLease(id, input),
    onSuccess: () => {
      refresh();
      close();
    },
  });
  const pay = useMutation({
    mutationFn: (input: PaymentInput) => recordPayment(id, input),
    onSuccess: () => {
      refresh();
      close();
    },
  });
  const charge = useMutation({
    mutationFn: (input: ChargeInput) => addCharge(id, input),
    onSuccess: () => {
      refresh();
      close();
    },
  });
  const terminate = useMutation({
    mutationFn: () => terminateLease(id, terminateOn),
    onSuccess: () => {
      refresh();
      close();
    },
  });
  const remove = useMutation({
    mutationFn: () => deleteLease(id),
    onSuccess: () => {
      refresh();
      navigate('/leases');
    },
  });
  const saveDeposit = useMutation({
    mutationFn: (input: DepositInput) => recordDeposit(id, input),
    onSuccess: () => {
      refresh();
      close();
    },
  });
  const removeDeposit = useMutation({ mutationFn: deleteDepositTransaction, onSuccess: refresh });
  const removeCharge = useMutation({ mutationFn: deleteCharge, onSuccess: refresh });
  const removePayment = useMutation({ mutationFn: deletePayment, onSuccess: refresh });

  if (lease.isPending) return <Loading />;
  if (lease.error) return <ErrorState error={lease.error} />;
  const l = lease.data;
  const ld = ledger.data;
  const dep = deposit.data;

  return (
    <>
      <PageHeader
        title={
          <>
            {l.tenant_name} <LeaseStatusBadge status={l.status} />
          </>
        }
        subtitle={
          <>
            <Link to={`/properties/${l.property_id}`}>{l.property_name}</Link> · Unit {l.unit_label} ·{' '}
            {formatDate(l.start_date)} – {formatDate(l.end_date)}
            {l.terminated_on && ` (ended early on ${formatDate(l.terminated_on)})`}
          </>
        }
        actions={
          <>
            <button type="button" className="button" onClick={() => setDialog('payment')}>
              Record payment
            </button>
            <button type="button" className="button secondary" onClick={() => setDialog('charge')}>
              Add charge
            </button>
            <button type="button" className="button secondary" onClick={() => setDialog('edit')}>
              Edit
            </button>
            {(l.status === 'active' || l.status === 'upcoming') && (
              <button type="button" className="button secondary" onClick={() => setDialog('terminate')}>
                End early
              </button>
            )}
          </>
        }
      />

      <div className="stats">
        <StatCard
          label="Balance"
          value={ld ? formatMoney(Math.abs(ld.balance_cents)) : '…'}
          detail={ld && ld.balance_cents < 0 ? 'in credit' : ld && ld.balance_cents === 0 ? 'all paid up' : 'owed'}
          tone={ld && ld.balance_cents <= 0 ? 'good' : undefined}
        />
        <StatCard
          label="Overdue"
          value={ld ? formatMoney(ld.overdue_cents) : '…'}
          detail={
            ld?.oldest_overdue ? `since ${formatDate(ld.oldest_overdue)} (${ld.days_overdue} days)` : 'nothing late'
          }
          tone={ld && ld.overdue_cents > 0 ? 'bad' : 'good'}
        />
        <StatCard
          label="Rent"
          value={formatMoney(l.rent_cents)}
          detail={`due on the ${ordinal(l.due_day)}, ${l.grace_days} days grace`}
        />
        <StatCard
          label="Deposit held"
          value={dep ? formatMoney(dep.held_cents) : '…'}
          detail={
            dep && dep.held_cents < l.deposit_cents && dep.received_cents < l.deposit_cents
              ? `${formatMoney(l.deposit_cents - dep.received_cents)} still to collect`
              : `${formatMoney(l.deposit_cents)} agreed`
          }
        />
      </div>
      {l.notes && <p className="notes">{l.notes}</p>}

      <section className="card">
        <div className="card-header">
          <h2>Charges</h2>
          {ld && <span className="muted">{formatMoney(ld.charged_cents)} charged to date</span>}
        </div>
        {ledger.isPending && <Loading />}
        {ld && ld.charges.length === 0 && <p className="muted">No charges yet. Rent is added on each due date.</p>}
        {ld && ld.charges.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Due</th>
                <th>Description</th>
                <th className="num">Amount</th>
                <th className="num">Paid</th>
                <th>Status</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {[...ld.charges].reverse().map((c) => (
                <tr key={c.id}>
                  <td>{formatDate(c.due_date)}</td>
                  <td>
                    <span className="muted">{chargeKinds[c.kind]}</span> {c.description}
                  </td>
                  <td className="num">{formatMoney(c.amount_cents)}</td>
                  <td className="num">{c.paid_cents ? formatMoney(c.paid_cents) : '—'}</td>
                  <td>
                    <ChargeStatusBadge status={c.status} />
                  </td>
                  <td className="row-actions">
                    {c.kind !== 'rent' && (
                      <button
                        type="button"
                        className="link-button danger"
                        onClick={() => {
                          if (confirm('Delete this charge?')) removeCharge.mutate(c.id);
                        }}
                      >
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card">
        <div className="card-header">
          <h2>Payments</h2>
          {ld && <span className="muted">{formatMoney(ld.paid_cents)} received</span>}
        </div>
        {ld && ld.payments.length === 0 && <p className="muted">No payments recorded.</p>}
        {ld && ld.payments.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Method</th>
                <th>Reference</th>
                <th className="num">Amount</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {ld.payments.map((p) => (
                <tr key={p.id}>
                  <td>{formatDate(p.paid_on)}</td>
                  <td>{paymentMethods[p.method]}</td>
                  <td>{p.reference || p.notes || '—'}</td>
                  <td className="num">{formatMoney(p.amount_cents)}</td>
                  <td className="row-actions">
                    <button
                      type="button"
                      className="link-button danger"
                      onClick={() => {
                        if (confirm('Delete this payment?')) removePayment.mutate(p.id);
                      }}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card">
        <div className="card-header">
          <h2>Deposit</h2>
          {dep && (
            <button type="button" className="link-button" onClick={() => setDialog('deposit')}>
              {dep.held_cents > 0 ? 'Return or withhold' : 'Record deposit'}
            </button>
          )}
        </div>
        {dep && dep.transactions.length === 0 && (
          <p className="muted">
            No deposit recorded yet. The lease asks for {formatMoney(l.deposit_cents)}. Deposits are held for the tenant
            and do not count as rent.
          </p>
        )}
        {dep && dep.transactions.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Type</th>
                <th>Notes</th>
                <th className="num">Amount</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {dep.transactions.map((t) => (
                <tr key={t.id}>
                  <td>{formatDate(t.occurred_on)}</td>
                  <td>{depositKinds[t.kind]}</td>
                  <td>{t.notes || '—'}</td>
                  <td className="num">
                    {t.kind === 'received' ? '' : '−'}
                    {formatMoney(t.amount_cents)}
                  </td>
                  <td className="row-actions">
                    <button
                      type="button"
                      className="link-button danger"
                      onClick={() => {
                        if (confirm('Delete this deposit entry?')) removeDeposit.mutate(t.id);
                      }}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <FormError error={removeDeposit.error} />
      </section>

      <div className="danger-zone">
        <button
          type="button"
          className="link-button danger"
          onClick={() => {
            if (confirm('Delete this lease with all its charges and payments? This cannot be undone.')) remove.mutate();
          }}
        >
          Delete lease
        </button>
      </div>

      {dialog === 'edit' && (
        <Modal title="Edit lease" onClose={close}>
          <LeaseForm initial={l} onSubmit={(input) => update.mutateAsync(input)} onCancel={close} />
        </Modal>
      )}
      {dialog === 'payment' && (
        <Modal title={`Payment from ${l.tenant_name}`} onClose={close}>
          <PaymentForm
            suggestedCents={ld?.balance_cents}
            onSubmit={(input) => pay.mutateAsync(input)}
            onCancel={close}
          />
        </Modal>
      )}
      {dialog === 'charge' && (
        <Modal title="Add charge" onClose={close}>
          <ChargeForm onSubmit={(input) => charge.mutateAsync(input)} onCancel={close} />
        </Modal>
      )}
      {dialog === 'deposit' && dep && (
        <Modal title="Deposit" onClose={close}>
          <DepositForm deposit={dep} onSubmit={(input) => saveDeposit.mutateAsync(input)} onCancel={close} />
        </Modal>
      )}
      {dialog === 'terminate' && (
        <Modal title="End lease early" onClose={close}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              terminate.mutate();
            }}
          >
            <FormError error={terminate.error} />
            <p className="muted">
              The lease stays active until this date. Rent due after it is removed and the final month is prorated to
              the days up to it.
            </p>
            <div className="field">
              <label htmlFor="terminate-on">Last day</label>
              <input
                id="terminate-on"
                type="date"
                value={terminateOn}
                onChange={(e) => setTerminateOn(e.target.value)}
              />
            </div>
            <div className="form-actions">
              <button type="button" className="button secondary" onClick={close}>
                Cancel
              </button>
              <button type="submit" className="button danger" disabled={terminate.isPending}>
                End lease
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
