import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { deleteTenant, getTenant, updateTenant } from '../api/tenants';
import type { TenantInput } from '../api/types';
import TenantForm from '../components/forms/TenantForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { ErrorState, Loading } from '../components/States';

export default function TenantDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);

  const tenant = useQuery({ queryKey: ['tenants', 'detail', id], queryFn: () => getTenant(id) });

  const update = useMutation({
    mutationFn: (input: TenantInput) => updateTenant(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tenants'] });
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

      {editing && (
        <Modal title="Edit tenant" onClose={() => setEditing(false)}>
          <TenantForm initial={t} onSubmit={(input) => update.mutateAsync(input)} onCancel={() => setEditing(false)} />
        </Modal>
      )}
    </>
  );
}
