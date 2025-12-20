import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { createProperty, listProperties } from '../api/properties';
import PropertyForm from '../components/forms/PropertyForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { propertyKinds } from '../lib/labels';

export default function PropertiesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [adding, setAdding] = useState(false);
  const { data, isPending, error } = useQuery({ queryKey: ['properties'], queryFn: listProperties });

  const create = useMutation({
    mutationFn: createProperty,
    onSuccess: (p) => {
      void queryClient.invalidateQueries({ queryKey: ['properties'] });
      navigate(`/properties/${p.id}`);
    },
  });

  return (
    <>
      <PageHeader
        title="Properties"
        actions={
          <button type="button" className="button" onClick={() => setAdding(true)}>
            Add property
          </button>
        }
      />
      {isPending && <Loading />}
      {error && <ErrorState error={error} />}
      {data && data.length === 0 && (
        <EmptyState title="No properties yet">Add your first building or house to get started.</EmptyState>
      )}
      {data && data.length > 0 && (
        <div className="cards">
          {data.map((p) => {
            const vacant = p.unit_count - p.occupied_units;
            return (
              <Link key={p.id} to={`/properties/${p.id}`} className="card property-card">
                <div className="property-kind">{propertyKinds[p.kind]}</div>
                <h3>{p.name}</h3>
                <div className="muted">
                  {p.address_line1}
                  {p.address_line2 ? `, ${p.address_line2}` : ''}, {p.city}
                  {p.region ? `, ${p.region}` : ''}
                </div>
                <div className="property-units">
                  {p.unit_count === 0 ? (
                    <span className="muted">No units yet</span>
                  ) : (
                    <>
                      <span>
                        {p.occupied_units}/{p.unit_count} let
                      </span>
                      {vacant > 0 && <span className="badge badge-amber">{vacant} vacant</span>}
                    </>
                  )}
                </div>
              </Link>
            );
          })}
        </div>
      )}
      {adding && (
        <Modal title="Add property" onClose={() => setAdding(false)}>
          <PropertyForm onSubmit={(input) => create.mutateAsync(input)} onCancel={() => setAdding(false)} />
        </Modal>
      )}
    </>
  );
}
