import type { ReactNode } from 'react';

export default function StatCard({
  label,
  value,
  detail,
  tone,
}: {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  tone?: 'good' | 'bad';
}) {
  return (
    <div className={tone ? `stat stat-${tone}` : 'stat'}>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {detail && <div className="stat-detail">{detail}</div>}
    </div>
  );
}
