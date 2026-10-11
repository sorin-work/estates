import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import Field from './Field';

describe('Field', () => {
  it('links the input to its error', () => {
    render(
      <Field label="Rent" error="must be more than zero">
        {(id) => <input id={id} />}
      </Field>,
    );
    const input = screen.getByLabelText('Rent');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription('must be more than zero');
  });

  it('describes the input with the hint when there is no error', () => {
    render(
      <Field label="Due day" hint="31 means the last day">
        {(id) => <input id={id} />}
      </Field>,
    );
    const input = screen.getByLabelText('Due day');
    expect(input).not.toHaveAttribute('aria-invalid');
    expect(input).toHaveAccessibleDescription('31 means the last day');
  });
});
