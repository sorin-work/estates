import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';
import type { VendorTrade } from '../api/types';
import { createVendor, listVendors } from '../api/vendors';
import VendorForm from '../components/forms/VendorForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { formatDate } from '../lib/dates';
import { options, vendorTrades } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function VendorsPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const trade = (params.get('trade') ?? '') as VendorTrade | '';
  const [adding, setAdding] = useState(false);

  const { data, isPending, error } = useQuery({ queryKey: ['vendors', trade], queryFn: () => listVendors(trade) });
  const create = useMutation({
    mutationFn: createVendor,
    onSuccess: (v) => {
      void queryClient.invalidateQueries({ queryKey: ['vendors'] });
      navigate(`/vendors/${v.id}`);
    },
  });

  return (
    <>
      <PageHeader
        title="Vendors"
        subtitle="Plumbers, electricians and everyone else who works on your properties."
        actions={
          <button type="button" className="button" onClick={() => setAdding(true)}>
            Add vendor
          </button>
        }
      />
      <div className="toolbar">
        <select
          aria-label="Trade"
          value={trade}
          onChange={(e) => setParams(e.target.value ? { trade: e.target.value } : {}, { replace: true })}
        >
          <option value="">All trades</option>
          {options(vendorTrades).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>
      {isPending && <Loading />}
      {error && <ErrorState error={error} />}
      {data && data.length === 0 && (
        <EmptyState title={trade ? 'No vendors in this trade' : 'No vendors yet'}>
          {!trade && 'Add the people you call for repairs to link them to maintenance requests and expenses.'}
        </EmptyState>
      )}
      {data && data.length > 0 && (
        <div className="card flush">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Trade</th>
                <th>Phone</th>
                <th className="num">Open jobs</th>
                <th className="num">Spent</th>
                <th>Last bill</th>
              </tr>
            </thead>
            <tbody>
              {data.map((v) => (
                <tr key={v.id}>
                  <td>
                    <Link to={`/vendors/${v.id}`}>{v.name}</Link>
                    {v.contact_name && <div className="muted small">{v.contact_name}</div>}
                  </td>
                  <td>{vendorTrades[v.trade]}</td>
                  <td>{v.phone ? <a href={`tel:${v.phone}`}>{v.phone}</a> : '—'}</td>
                  <td className="num">{v.open_jobs || '—'}</td>
                  <td className="num">{formatMoney(v.spent_cents)}</td>
                  <td>{formatDate(v.last_used_on)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {adding && (
        <Modal title="Add vendor" onClose={() => setAdding(false)}>
          <VendorForm onSubmit={(input) => create.mutateAsync(input)} onCancel={() => setAdding(false)} />
        </Modal>
      )}
    </>
  );
}
