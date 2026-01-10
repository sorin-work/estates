import type { MaintenanceRequest, MaintenanceStatus } from '../api/types';
import { maintenanceActions } from '../lib/labels';

/** Buttons for the status changes the API says are allowed. */
export default function MaintenanceActions({
  request,
  busy,
  onChange,
}: {
  request: MaintenanceRequest;
  busy?: boolean;
  onChange: (status: MaintenanceStatus) => void;
}) {
  // Most likely next step first.
  const order: MaintenanceStatus[] = ['in_progress', 'completed', 'on_hold', 'open', 'cancelled'];
  const next = order.filter((s) => request.next_statuses.includes(s));
  return (
    <div className="status-actions">
      {next.map((s) => (
        <button
          key={s}
          type="button"
          className={s === 'cancelled' ? 'button small secondary danger' : 'button small secondary'}
          disabled={busy}
          onClick={() => onChange(s)}
        >
          {maintenanceActions[s]}
        </button>
      ))}
    </div>
  );
}
