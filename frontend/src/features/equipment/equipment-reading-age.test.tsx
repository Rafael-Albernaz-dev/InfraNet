import { render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { readingAge } from './equipment-reading-format';
import { PPPCatalogPanel } from '../pppoe/ppp-catalog-panel';
import { pppCatalogAPI } from '../../platform/wails/presentation';
afterEach(() => vi.restoreAllMocks());
it('omits missing and invalid reading dates from the PPP servers heading', async () => {
  expect(readingAge()).toBe('');
  expect(readingAge('not-a-date')).toBe('');
  expect(
    readingAge('2026-11-15T12:00:00Z', Date.parse('2026-11-14T12:00:00Z')),
  ).toBe('');
  expect(
    readingAge('2026-11-14T12:00:00Z', Date.parse('2026-11-14T12:01:00Z')),
  ).toMatch(/^há /);
  vi.spyOn(pppCatalogAPI, 'list').mockResolvedValue({
    code: 'OK',
    profile_id: 'synthetic',
    activation_id: 'active',
    data: { entries: [], total: 0 },
  });
  render(
    <PPPCatalogPanel
      profile="synthetic"
      activation="active"
      screen="servers"
    />,
  );
  await screen.findByText('Servidores PPPoE');
  expect(screen.queryByText(/há Não informado/)).not.toBeInTheDocument();
});
