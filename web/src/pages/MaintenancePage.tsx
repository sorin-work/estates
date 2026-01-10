import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import {
  createMaintenance,
  deleteMaintenance,
  listMaintenance,
  setMaintenanceStatus,
  updateMaintenance,
} from '../api/maintenance';
import { listProperties } from '../api/properties';
import type { MaintenanceInput, MaintenancePriority, MaintenanceRequest, MaintenanceStatus } from '../api/types';
import MaintenanceForm from '../components/forms/MaintenanceForm';
import MaintenanceActions from '../components/MaintenanceActions';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, FormError, Loading } from '../components/States';
import { MaintenanceStatusBadge, PriorityBadge } from '../components/StatusBadge';
import { daysBetween, formatDate, today } from '../lib/dates';
import { maintenancePriorities, maintenanceStatuses, options } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function MaintenancePage() {
  const queryClient = useQueryClient();
  const [params, setParams] = useSearchParams();
  const propertyId = params.get('property_id') ?? '';
  // "active" means anything not closed; "all" turns the status filter off.
  const status = params.get('status') ?? 'active';
  const priority = (params.get('priority') ?? '') as MaintenancePriority | '';
  const [form, setForm] = useState<MaintenanceRequest | 'new' | null>(null);

  const filter = {
    property_id: propertyId,
    status: (status === 'all' ? '' : status) as MaintenanceStatus | 'active' | '',
    priority,
  };
  const requests = useQuery({ queryKey: ['maintenance', filter], queryFn: () => listMaintenance(filter) });
  const properties = useQuery({ queryKey: ['properties'], queryFn: listProperties });

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['maintenance'] });
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] });
  };
  const save = useMutation({
    mutationFn: (input: MaintenanceInput) =>
      form && form !== 'new' ? updateMaintenance(form.id, input) : createMaintenance(input),
    onSuccess: () => {
      refresh();
      setForm(null);
    },
  });
  const move = useMutation({
    mutationFn: ({ id, to }: { id: string; to: MaintenanceStatus }) => setMaintenanceStatus(id, to),
    onSuccess: refresh,
  });
  const remove = useMutation({ mutationFn: deleteMaintenance, onSuccess: refresh });

  function setFilter(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: true });
  }

  return (
    <>
      <PageHeader
        title="Maintenance"
        actions={
          <button type="button" className="button" onClick={() => setForm('new')}>
            Log request
          </button>
        }
      />
      <div className="toolbar">
        <select aria-label="Property" value={propertyId} onChange={(e) => setFilter('property_id', e.target.value)}>
          <option value="">All properties</option>
          {properties.data?.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <select aria-label="Status" value={status} onChange={(e) => setFilter('status', e.target.value)}>
          <option value="active">Needs attention</option>
          <option value="all">Any status</option>
          {options(maintenanceStatuses).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <select aria-label="Priority" value={priority} onChange={(e) => setFilter('priority', e.target.value)}>
          <option value="">Any priority</option>
          {options(maintenancePriorities).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>
      <FormError error={move.error} />
      {requests.isPending && <Loading />}
      {requests.error && <ErrorState error={requests.error} />}
      {requests.data && requests.data.length === 0 && (
        <EmptyState title={status === 'active' ? 'Nothing needs attention' : 'No requests match'}>
          Log repairs and jobs here to keep track of them until they are done.
        </EmptyState>
      )}
      <div className="requests">
        {requests.data?.map((m) => {
          const age = daysBetween(m.reported_on, today());
          return (
            <article key={m.id} className="card request">
              <div className="request-head">
                <div>
                  <h3>{m.title}</h3>
                  <div className="muted">
                    {m.property_name}
                    {m.unit_label ? ` · Unit ${m.unit_label}` : ' · Common area'} · reported {formatDate(m.reported_on)}
                    {m.status !== 'completed' && m.status !== 'cancelled' && age > 0 && ` (${age} days ago)`}
                  </div>
                </div>
                <div className="row">
                  <PriorityBadge priority={m.priority} />
                  <MaintenanceStatusBadge status={m.status} />
                </div>
              </div>
              {m.description && <p>{m.description}</p>}
              <div className="request-meta muted">
                {m.vendor && <span>Vendor: {m.vendor}</span>}
                {m.cost_cents > 0 && <span>Cost so far: {formatMoney(m.cost_cents)}</span>}
                {m.completed_on && <span>Completed {formatDate(m.completed_on)}</span>}
              </div>
              <div className="request-foot">
                <MaintenanceActions
                  request={m}
                  busy={move.isPending}
                  onChange={(to) => move.mutate({ id: m.id, to })}
                />
                <div className="row">
                  <button type="button" className="link-button" onClick={() => setForm(m)}>
                    Edit
                  </button>
                  <button
                    type="button"
                    className="link-button danger"
                    onClick={() => {
                      if (confirm('Delete this request?')) remove.mutate(m.id);
                    }}
                  >
                    Delete
                  </button>
                </div>
              </div>
            </article>
          );
        })}
      </div>
      {form && (
        <Modal title={form === 'new' ? 'Log maintenance request' : 'Edit request'} onClose={() => setForm(null)}>
          <MaintenanceForm
            initial={form === 'new' ? undefined : form}
            propertyId={propertyId || undefined}
            onSubmit={(input) => save.mutateAsync(input)}
            onCancel={() => setForm(null)}
          />
        </Modal>
      )}
    </>
  );
}
