import { render, screen } from '@testing-library/react';
import { expect, it } from 'vitest';
import { ProductBrand } from './ProductBrand';

it('shows InfraNet with the approved Fenda symbol using theme colors', () => {
  const { container } = render(<ProductBrand />);
  expect(screen.getByText('InfraNet')).toBeTruthy();
  expect(container.querySelector('svg')?.getAttribute('viewBox')).toBe(
    '0 0 36 36',
  );
  expect(container.querySelector('rect')?.getAttribute('fill')).toBe(
    'var(--brand-background)',
  );
  expect(container.querySelector('g')?.getAttribute('fill')).toBe(
    'var(--brand-foreground)',
  );
  expect(
    [...container.querySelectorAll('path')].map((path) =>
      path.getAttribute('d'),
    ),
  ).toEqual([
    'M5 29 L5 12 L11 5 L16 5 L29 24 L24 31 L20 25 L11 13 L11 29 Z',
    'M25 5 L31 5 L31 18 L25 12 Z',
  ]);
});
