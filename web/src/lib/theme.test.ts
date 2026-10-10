import { afterEach, describe, expect, it } from 'vitest';
import { applyTheme, getTheme, setTheme } from './theme';

afterEach(() => setTheme('system'));

describe('theme', () => {
  it('remembers the choice and sets data-theme', () => {
    expect(getTheme()).toBe('system');
    setTheme('dark');
    expect(getTheme()).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    setTheme('system');
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
  });

  it('ignores unknown stored values', () => {
    localStorage.setItem('estates.theme', 'purple');
    expect(getTheme()).toBe('system');
    applyTheme(getTheme());
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
  });
});
