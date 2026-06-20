import { useQuery } from '@tanstack/react-query';
import { listVendors } from '../../api/vendors';
import Field from '../Field';

export interface VendorChoice {
  vendorId: string;
  name: string;
}

/**
 * Picks a saved vendor, or falls back to typing a name for someone who is
 * not worth saving (a one-off hardware store, say).
 */
export default function VendorPicker({
  value,
  onChange,
  error,
  hint,
}: {
  value: VendorChoice;
  onChange: (value: VendorChoice) => void;
  error?: string;
  hint?: string;
}) {
  const vendors = useQuery({ queryKey: ['vendors', ''], queryFn: () => listVendors() });

  return (
    <>
      <Field label="Vendor" error={error} hint={hint}>
        {(id) => (
          <select
            id={id}
            value={value.vendorId}
            onChange={(e) => {
              const v = vendors.data?.find((x) => x.id === e.target.value);
              onChange({ vendorId: v?.id ?? '', name: v?.name ?? '' });
            }}
          >
            <option value="">{vendors.data?.length ? 'Not a saved vendor' : 'No saved vendors'}</option>
            {vendors.data?.map((v) => (
              <option key={v.id} value={v.id}>
                {v.name}
              </option>
            ))}
          </select>
        )}
      </Field>
      {!value.vendorId && (
        <Field label="Vendor name">
          {(id) => (
            <input id={id} value={value.name} onChange={(e) => onChange({ vendorId: '', name: e.target.value })} />
          )}
        </Field>
      )}
    </>
  );
}
