import { cloneElement, isValidElement, useId, type ReactElement, type ReactNode } from 'react';

/**
 * A labelled form control with an optional error message. Children get the
 * generated id so the label is linked to the input, and the input is marked
 * invalid and described by the error or hint so screen readers announce it.
 */
export default function Field({
  label,
  error,
  hint,
  children,
}: {
  label: string;
  error?: string;
  hint?: string;
  children: (id: string) => ReactNode;
}) {
  const id = useId();
  const noteId = `${id}-note`;
  const note = error || hint;
  let control = children(id);
  if (isValidElement(control)) {
    control = cloneElement(control as ReactElement<Record<string, unknown>>, {
      'aria-invalid': error ? true : undefined,
      'aria-describedby': note ? noteId : undefined,
    });
  }
  return (
    <div className={error ? 'field has-error' : 'field'}>
      <label htmlFor={id}>{label}</label>
      {control}
      {error ? (
        <div className="field-error" id={noteId}>
          {error}
        </div>
      ) : (
        hint && (
          <div className="field-hint" id={noteId}>
            {hint}
          </div>
        )
      )}
    </div>
  );
}
