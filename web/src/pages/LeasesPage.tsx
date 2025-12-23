import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useSearchParams } from 'react-router';
import { createLease, listLeases } from '../api/leases';
import type { LeaseStatus } from '../api/types';
import LeaseForm from '../components/forms/LeaseForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { LeaseStatusBadge } from '../components/StatusBadge';
import { formatDate } from '../lib/dates';
import { leaseStatuses, options } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function LeasesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const status = (params.get('status') ?? '') as LeaseStatus | '';
  const adding = params.get('new') === '1';

  const { data, isPending, error } = useQuery({
    queryKey: ['leases', { status }],
    queryFn: () => listLeases({ status }),
  });

  const create = useMutation({
    mutationFn: createLease,
    onSuccess: (l) => {
      void queryClient.invalidateQueries({ queryKey: ['leases'] });
      void queryClient.invalidateQueries({ queryKey: ['units'] });
      void queryClient.invalidateQueries({ queryKey: ['properties'] });
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      navigate(`/leases/${l.id}`);
    },
  });

  function setStatus(value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set('status', value);
    else next.delete('status');
    setParams(next, { replace: true });
  }

  function closeForm() {
    const next = new URLSearchParams(params);
    next.delete('new');
    next.delete('unit_id');
    next.delete('tenant_id');
    setParams(next, { replace: true });
  }

  return (
    <>
      <PageHeader
        title="Leases"
        actions={
          <button
            type="button"
            className="button"
            onClick={() => {
              const next = new URLSearchParams(params);
              next.set('new', '1');
              setParams(next);
            }}
          >
            New lease
          </button>
        }
      />
      <div className="toolbar">
        <div className="segmented" role="group" aria-label="Filter by status">
          {[{ value: '', label: 'All' }, ...options(leaseStatuses)].map((o) => (
            <button
              key={o.value}
              type="button"
              className={status === o.value ? 'active' : ''}
              onClick={() => setStatus(o.value)}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>
      {isPending && <Loading />}
      {error && <ErrorState error={error} />}
      {data && data.length === 0 && (
        <EmptyState title="No leases here">Create a lease to start tracking rent for a unit.</EmptyState>
      )}
      {data && data.length > 0 && (
        <div className="card flush">
          <table className="table">
            <thead>
              <tr>
                <th>Tenant</th>
                <th>Property</th>
                <th>Term</th>
                <th className="num">Rent</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {data.map((l) => (
                <tr key={l.id}>
                  <td>
                    <Link to={`/leases/${l.id}`}>{l.tenant_name}</Link>
                  </td>
                  <td>
                    {l.property_name} · {l.unit_label}
                  </td>
                  <td>
                    {formatDate(l.start_date)} – {formatDate(l.terminated_on ?? l.end_date)}
                  </td>
                  <td className="num">{formatMoney(l.rent_cents)}</td>
                  <td>
                    <LeaseStatusBadge status={l.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {adding && (
        <Modal title="New lease" onClose={closeForm}>
          <LeaseForm
            defaults={{ unit_id: params.get('unit_id') ?? undefined, tenant_id: params.get('tenant_id') ?? undefined }}
            onSubmit={(input) => create.mutateAsync(input)}
            onCancel={closeForm}
          />
        </Modal>
      )}
    </>
  );
}
