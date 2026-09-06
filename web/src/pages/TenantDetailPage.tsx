import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { listLeases } from '../api/leases';
import { deleteTenant, getTenant, updateTenant } from '../api/tenants';
import type { TenantInput } from '../api/types';
import TenantForm from '../components/forms/TenantForm';
import Documents from '../components/Documents';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { ErrorState, Loading } from '../components/States';
import { LeaseStatusBadge } from '../components/StatusBadge';
import { formatDate } from '../lib/dates';
import { formatMoney } from '../lib/money';

export default function TenantDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);

  const tenant = useQuery({ queryKey: ['tenants', 'detail', id], queryFn: () => getTenant(id) });
  const leases = useQuery({ queryKey: ['leases', { tenant_id: id }], queryFn: () => listLeases({ tenant_id: id }) });

  const update = useMutation({
    mutationFn: (input: TenantInput) => updateTenant(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tenants'] });
      void queryClient.invalidateQueries({ queryKey: ['leases'] });
      setEditing(false);
    },
  });
  const remove = useMutation({
    mutationFn: () => deleteTenant(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tenants'] });
      navigate('/tenants');
    },
    onError: (err) => alert(err.message),
  });

  if (tenant.isPending) return <Loading />;
  if (tenant.error) return <ErrorState error={tenant.error} />;
  const t = tenant.data;

  return (
    <>
      <PageHeader
        title={`${t.first_name} ${t.last_name}`}
        subtitle={[t.email, t.phone].filter(Boolean).join(' · ') || 'No contact details'}
        actions={
          <>
            <Link to={`/leases?new=1&tenant_id=${t.id}`} className="button">
              New lease
            </Link>
            <button type="button" className="button secondary" onClick={() => setEditing(true)}>
              Edit
            </button>
            <button
              type="button"
              className="button danger"
              onClick={() => {
                if (confirm(`Delete ${t.first_name} ${t.last_name}?`)) remove.mutate();
              }}
            >
              Delete
            </button>
          </>
        }
      />
      {t.notes && <p className="notes">{t.notes}</p>}

      <section className="card">
        <div className="card-header">
          <h2>Leases</h2>
        </div>
        {leases.data && leases.data.length === 0 && <p className="muted">No leases yet.</p>}
        {leases.data && leases.data.length > 0 && (
          <table className="table">
            <thead>
              <tr>
                <th>Property</th>
                <th>Term</th>
                <th className="num">Rent</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {leases.data.map((l) => (
                <tr key={l.id}>
                  <td>
                    <Link to={`/leases/${l.id}`}>
                      {l.property_name} · {l.unit_label}
                    </Link>
                  </td>
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

      <Documents target={{ tenant_id: t.id }} hint="ID copies, references and applications." />

      {editing && (
        <Modal title="Edit tenant" onClose={() => setEditing(false)}>
          <TenantForm initial={t} onSubmit={(input) => update.mutateAsync(input)} onCancel={() => setEditing(false)} />
        </Modal>
      )}
    </>
  );
}
