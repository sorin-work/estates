import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { listExpenses } from '../api/expenses';
import { listMaintenance } from '../api/maintenance';
import type { VendorInput } from '../api/types';
import { deleteVendor, getVendor, updateVendor } from '../api/vendors';
import VendorForm from '../components/forms/VendorForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import StatCard from '../components/StatCard';
import { ErrorState, FormError, Loading } from '../components/States';
import { MaintenanceStatusBadge, PriorityBadge } from '../components/StatusBadge';
import { formatDate } from '../lib/dates';
import { expenseCategories, vendorTrades } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function VendorDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);

  const vendor = useQuery({ queryKey: ['vendors', 'detail', id], queryFn: () => getVendor(id) });
  const jobs = useQuery({
    queryKey: ['maintenance', { vendor_id: id }],
    queryFn: () => listMaintenance({ vendor_id: id }),
  });
  const expenses = useQuery({
    queryKey: ['expenses', { vendor_id: id }],
    queryFn: () => listExpenses({ vendor_id: id }),
  });

  const update = useMutation({
    mutationFn: (input: VendorInput) => updateVendor(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['vendors'] });
      setEditing(false);
    },
  });
  const remove = useMutation({
    mutationFn: () => deleteVendor(id),
    onSuccess: () => {
      for (const key of ['vendors', 'maintenance', 'expenses']) {
        void queryClient.invalidateQueries({ queryKey: [key] });
      }
      navigate('/vendors');
    },
  });

  if (vendor.isPending) return <Loading />;
  if (vendor.error) return <ErrorState error={vendor.error} />;
  const v = vendor.data;

  return (
    <>
      <PageHeader
        title={v.name}
        subtitle={
          <>
            {vendorTrades[v.trade]}
            {v.contact_name && ` · ${v.contact_name}`}
            {v.phone && (
              <>
                {' · '}
                <a href={`tel:${v.phone}`}>{v.phone}</a>
              </>
            )}
            {v.email && (
              <>
                {' · '}
                <a href={`mailto:${v.email}`}>{v.email}</a>
              </>
            )}
          </>
        }
        actions={
          <button type="button" className="button secondary" onClick={() => setEditing(true)}>
            Edit
          </button>
        }
      />
      <div className="stats">
        <StatCard label="Open jobs" value={String(v.open_jobs)} />
        <StatCard label="Spent" value={formatMoney(v.spent_cents)} detail="all time" />
        <StatCard label="Last bill" value={formatDate(v.last_used_on)} />
      </div>
      {v.notes && <p className="notes">{v.notes}</p>}

      <section className="card">
        <div className="card-header">
          <h2>Jobs</h2>
        </div>
        {jobs.data && jobs.data.length === 0 && <p className="muted">No maintenance requests for this vendor.</p>}
        {jobs.data && jobs.data.length > 0 && (
          <ul className="list">
            {jobs.data.map((m) => (
              <li key={m.id}>
                <div>
                  <strong>{m.title}</strong>
                  <div className="muted small">
                    <Link to={`/properties/${m.property_id}`}>{m.property_name}</Link>
                    {m.unit_label && ` · Unit ${m.unit_label}`} · reported {formatDate(m.reported_on)}
                  </div>
                </div>
                <div className="list-badges">
                  <PriorityBadge priority={m.priority} />
                  <MaintenanceStatusBadge status={m.status} />
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="card">
        <div className="card-header">
          <h2>Expenses</h2>
          {expenses.data && <span className="muted">{formatMoney(expenses.data.total_cents)}</span>}
        </div>
        {expenses.data && expenses.data.expenses.length === 0 && <p className="muted">No expenses recorded.</p>}
        {expenses.data && expenses.data.expenses.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Property</th>
                <th>Category</th>
                <th>Description</th>
                <th className="num">Amount</th>
              </tr>
            </thead>
            <tbody>
              {expenses.data.expenses.map((e) => (
                <tr key={e.id}>
                  <td>{formatDate(e.spent_on)}</td>
                  <td>{e.property_name}</td>
                  <td>{expenseCategories[e.category]}</td>
                  <td>{e.description || '—'}</td>
                  <td className="num">{formatMoney(e.amount_cents)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <div className="danger-zone">
        <FormError error={remove.error} />
        <button
          type="button"
          className="link-button danger"
          onClick={() => {
            if (confirm('Delete this vendor? Jobs and expenses keep the name but are no longer linked.'))
              remove.mutate();
          }}
        >
          Delete vendor
        </button>
      </div>

      {editing && (
        <Modal title="Edit vendor" onClose={() => setEditing(false)}>
          <VendorForm initial={v} onSubmit={(input) => update.mutateAsync(input)} onCancel={() => setEditing(false)} />
        </Modal>
      )}
    </>
  );
}
