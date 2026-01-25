import type { MonthTotal } from '../../api/types';
import { formatMonthShort } from '../../lib/dates';
import { formatMoney, formatMoneyCompact } from '../../lib/money';

const HEIGHT = 180;
const BAR = 10;
const GAP = 3;
const GROUP = BAR * 2 + GAP + 16;

/** Grouped bars of rent collected vs expenses for each month. */
export default function IncomeChart({ months }: { months: MonthTotal[] }) {
  const max = Math.max(1, ...months.flatMap((m) => [m.collected_cents, m.expenses_cents]));
  const width = months.length * GROUP;
  const y = (v: number) => HEIGHT - (v / max) * HEIGHT;

  return (
    <div className="chart">
      <svg
        viewBox={`0 -12 ${width + 48} ${HEIGHT + 36}`}
        role="img"
        aria-label="Rent collected and expenses by month"
        preserveAspectRatio="xMidYMid meet"
      >
        {[0, 0.5, 1].map((t) => (
          <g key={t}>
            <line x1={48} x2={width + 48} y1={y(max * t)} y2={y(max * t)} className="chart-grid" />
            <text x={42} y={y(max * t) + 4} textAnchor="end" className="chart-axis">
              {formatMoneyCompact(max * t)}
            </text>
          </g>
        ))}
        {months.map((m, i) => {
          const x = 48 + i * GROUP + 8;
          return (
            <g key={m.month}>
              <rect
                x={x}
                y={y(m.collected_cents)}
                width={BAR}
                height={HEIGHT - y(m.collected_cents)}
                className="bar-income"
              >
                <title>{`${formatMonthShort(m.month)}: ${formatMoney(m.collected_cents)} collected`}</title>
              </rect>
              <rect
                x={x + BAR + GAP}
                y={y(m.expenses_cents)}
                width={BAR}
                height={HEIGHT - y(m.expenses_cents)}
                className="bar-expense"
              >
                <title>{`${formatMonthShort(m.month)}: ${formatMoney(m.expenses_cents)} expenses`}</title>
              </rect>
              <text x={x + BAR} y={HEIGHT + 16} textAnchor="middle" className="chart-axis">
                {formatMonthShort(m.month)}
              </text>
            </g>
          );
        })}
      </svg>
      <div className="legend">
        <span>
          <i className="swatch income" /> Collected
        </span>
        <span>
          <i className="swatch expense" /> Expenses
        </span>
      </div>
    </div>
  );
}
