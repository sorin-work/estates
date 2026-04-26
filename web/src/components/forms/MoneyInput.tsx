import type { InputHTMLAttributes } from 'react';
import { currencySymbol } from '../../lib/money';

/** A text input for money amounts. Parse the value with parseMoney. */
export default function MoneyInput(props: Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>) {
  return (
    <div className="money-input">
      <span aria-hidden="true">{currencySymbol()}</span>
      <input type="text" inputMode="decimal" autoComplete="off" placeholder="0.00" {...props} />
    </div>
  );
}
