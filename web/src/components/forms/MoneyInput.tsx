import type { InputHTMLAttributes } from 'react';

/** A text input for dollar amounts. Parse the value with parseMoney. */
export default function MoneyInput(props: Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>) {
  return (
    <div className="money-input">
      <span aria-hidden="true">$</span>
      <input type="text" inputMode="decimal" autoComplete="off" placeholder="0.00" {...props} />
    </div>
  );
}
