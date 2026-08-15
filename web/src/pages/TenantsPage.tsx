import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { createTenant, listTenants } from '../api/tenants';
import TenantForm from '../components/forms/TenantForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';

function useDebounced<T>(value: T, ms: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return debounced;
}

export default function TenantsPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [adding, setAdding] = useState(false);
  const [search, setSearch] = useState('');
  const q = useDebounced(search.trim(), 250);

  const { data, isPending, error } = useQuery({
    queryKey: ['tenants', q],
    queryFn: () => listTenants(q),
    placeholderData: keepPreviousData,
  });
  const create = useMutation({
    mutationFn: createTenant,
    onSuccess: (t) => {
      void queryClient.invalidateQueries({ queryKey: ['tenants'] });
      navigate(`/tenants/${t.id}`);
    },
  });

  return (
    <>
      <PageHeader
        title="Tenants"
        actions={
          <button type="button" className="button" onClick={() => setAdding(true)}>
            Add tenant
          </button>
        }
      />
      <div className="toolbar">
        <input
          type="search"
          placeholder="Search by name, email or phone"
          aria-label="Search tenants"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>
      {isPending && <Loading />}
      {error && <ErrorState error={error} />}
      {data && data.length === 0 && (
        <EmptyState title={q ? 'No tenants match your search' : 'No tenants yet'}>
          {!q && 'Add the people renting from you, then create a lease for each.'}
        </EmptyState>
      )}
      {data && data.length > 0 && (
        <div className="card flush">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Phone</th>
                <th className="num">Leases</th>
              </tr>
            </thead>
            <tbody>
              {data.map((t) => (
                <tr key={t.id}>
                  <td>
                    <Link to={`/tenants/${t.id}`}>
                      {t.first_name} {t.last_name}
                    </Link>
                  </td>
                  <td>{t.email ? <a href={`mailto:${t.email}`}>{t.email}</a> : '—'}</td>
                  <td>{t.phone || '—'}</td>
                  <td className="num">{t.lease_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {adding && (
        <Modal title="Add tenant" onClose={() => setAdding(false)}>
          <TenantForm onSubmit={(input) => create.mutateAsync(input)} onCancel={() => setAdding(false)} />
        </Modal>
      )}
    </>
  );
}
