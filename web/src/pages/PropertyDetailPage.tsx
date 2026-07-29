import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { listExpenses } from '../api/expenses';
import { listLeases } from '../api/leases';
import { listMaintenance } from '../api/maintenance';
import {
  createUnit,
  deleteProperty,
  deleteUnit,
  getProperty,
  listUnits,
  updateProperty,
  updateUnit,
} from '../api/properties';
import type { Unit } from '../api/types';
import PropertyForm from '../components/forms/PropertyForm';
import UnitForm from '../components/forms/UnitForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { ErrorState, Loading } from '../components/States';
import { LeaseStatusBadge, MaintenanceStatusBadge, PriorityBadge } from '../components/StatusBadge';
import { formatDate, today } from '../lib/dates';
import { expenseCategories, propertyKinds } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function PropertyDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [unitForm, setUnitForm] = useState<Unit | 'new' | null>(null);

  const property = useQuery({ queryKey: ['properties', id], queryFn: () => getProperty(id) });
  const units = useQuery({ queryKey: ['units', id], queryFn: () => listUnits(id) });
  const leases = useQuery({
    queryKey: ['leases', { property_id: id }],
    queryFn: () => listLeases({ property_id: id }),
  });
  const maintenance = useQuery({
    queryKey: ['maintenance', { property_id: id, status: 'active' }],
    queryFn: () => listMaintenance({ property_id: id, status: 'active' }),
  });
  const yearStart = `${today().slice(0, 4)}-01-01`;
  const expenses = useQuery({
    queryKey: ['expenses', { property_id: id, from: yearStart }],
    queryFn: () => listExpenses({ property_id: id, from: yearStart }),
  });

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['properties'] });
    void queryClient.invalidateQueries({ queryKey: ['units'] });
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] });
  };

  const update = useMutation({
    mutationFn: (input: Parameters<typeof updateProperty>[1]) => updateProperty(id, input),
    onSuccess: () => {
      refresh();
      setEditing(false);
    },
  });
  const remove = useMutation({
    mutationFn: () => deleteProperty(id),
    onSuccess: () => {
      refresh();
      navigate('/properties');
    },
    onError: (err) => alert(err.message),
  });
  const saveUnit = useMutation({
    mutationFn: (input: Parameters<typeof createUnit>[1]) =>
      unitForm && unitForm !== 'new' ? updateUnit(unitForm.id, input) : createUnit(id, input),
    onSuccess: () => {
      refresh();
      setUnitForm(null);
    },
  });
  const removeUnit = useMutation({
    mutationFn: deleteUnit,
    onSuccess: refresh,
    onError: (err) => alert(err.message),
  });

  if (property.isPending) return <Loading />;
  if (property.error) return <ErrorState error={property.error} />;
  const p = property.data;

  return (
    <>
      <PageHeader
        title={p.name}
        subtitle={
          <>
            {propertyKinds[p.kind]} · {p.address_line1}
            {p.address_line2 ? `, ${p.address_line2}` : ''}, {p.city} {p.region} {p.postal_code}
          </>
        }
        actions={
          <>
            <button type="button" className="button secondary" onClick={() => setEditing(true)}>
              Edit
            </button>
            <button
              type="button"
              className="button danger"
              onClick={() => {
                if (confirm(`Delete ${p.name} and all its units?`)) remove.mutate();
              }}
            >
              Delete
            </button>
          </>
        }
      />
      {p.notes && <p className="notes">{p.notes}</p>}

      <section className="card">
        <div className="card-header">
          <h2>Units</h2>
          <button type="button" className="button small" onClick={() => setUnitForm('new')}>
            Add unit
          </button>
        </div>
        {units.data && units.data.length === 0 && <p className="muted">Add the units you rent out.</p>}
        {units.data && units.data.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Unit</th>
                <th>Beds / baths</th>
                <th>Sq ft</th>
                <th className="num">Market rent</th>
                <th>Tenant</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {units.data.map((u) => (
                <tr key={u.id}>
                  <td>
                    <strong>{u.label}</strong>
                  </td>
                  <td>
                    {u.bedrooms} bd / {u.bathrooms} ba
                  </td>
                  <td>{u.square_feet ?? '—'}</td>
                  <td className="num">{u.market_rent_cents ? formatMoney(u.market_rent_cents) : '—'}</td>
                  <td>
                    {u.current_lease_id ? (
                      <Link to={`/leases/${u.current_lease_id}`}>{u.current_tenant}</Link>
                    ) : (
                      <span className="badge badge-amber">Vacant</span>
                    )}
                  </td>
                  <td className="row-actions">
                    {!u.current_lease_id && (
                      <Link to={`/leases?new=1&unit_id=${u.id}`} className="button small secondary">
                        New lease
                      </Link>
                    )}
                    <button type="button" className="link-button" onClick={() => setUnitForm(u)}>
                      Edit
                    </button>
                    <button
                      type="button"
                      className="link-button danger"
                      onClick={() => {
                        if (confirm(`Delete unit ${u.label}?`)) removeUnit.mutate(u.id);
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

      <div className="grid-2">
        <section className="card">
          <div className="card-header">
            <h2>Open maintenance</h2>
            <Link to={`/maintenance?property_id=${id}`}>View all</Link>
          </div>
          {maintenance.data && maintenance.data.length === 0 && <p className="muted">Nothing open.</p>}
          <ul className="list">
            {maintenance.data?.map((m) => (
              <li key={m.id}>
                <span>
                  <strong>{m.title}</strong>
                  <span className="muted">
                    {m.unit_label ?? 'Common area'} · {formatDate(m.reported_on)}
                  </span>
                </span>
                <span className="list-end row">
                  <PriorityBadge priority={m.priority} />
                  <MaintenanceStatusBadge status={m.status} />
                </span>
              </li>
            ))}
          </ul>
        </section>

        <section className="card">
          <div className="card-header">
            <h2>Expenses this year</h2>
            <Link to={`/expenses?property_id=${id}`}>View all</Link>
          </div>
          {expenses.data && (
            <>
              <div className="stat-value">{formatMoney(expenses.data.total_cents)}</div>
              <ul className="list compact">
                {expenses.data.by_category.map((c) => (
                  <li key={c.category}>
                    <span>{expenseCategories[c.category]}</span>
                    <span>{formatMoney(c.amount_cents)}</span>
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>
      </div>

      <section className="card">
        <div className="card-header">
          <h2>Lease history</h2>
        </div>
        {leases.data && leases.data.length === 0 && <p className="muted">No leases yet.</p>}
        {leases.data && leases.data.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Tenant</th>
                <th>Unit</th>
                <th>Term</th>
                <th className="num">Rent</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {leases.data.map((l) => (
                <tr key={l.id}>
                  <td>
                    <Link to={`/leases/${l.id}`}>{l.tenant_name}</Link>
                  </td>
                  <td>{l.unit_label}</td>
                  <td>
                    {formatDate(l.start_date)} – {formatDate(l.terminated_on ?? l.end_date)}
                  </td>
                  <td className="num">{formatMoney(l.current_rent_cents)}</td>
                  <td>
                    <LeaseStatusBadge status={l.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {editing && (
        <Modal title="Edit property" onClose={() => setEditing(false)}>
          <PropertyForm
            initial={p}
            onSubmit={(input) => update.mutateAsync(input)}
            onCancel={() => setEditing(false)}
          />
        </Modal>
      )}
      {unitForm && (
        <Modal
          title={unitForm === 'new' ? 'Add unit' : `Edit unit ${unitForm.label}`}
          onClose={() => setUnitForm(null)}
        >
          <UnitForm
            initial={unitForm === 'new' ? undefined : unitForm}
            onSubmit={(input) => saveUnit.mutateAsync(input)}
            onCancel={() => setUnitForm(null)}
          />
        </Modal>
      )}
    </>
  );
}
