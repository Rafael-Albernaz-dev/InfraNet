import { expect, it, vi } from 'vitest';
import { completePPTable, completeRadioIndex } from './stored-table-pages';
import type { PPResult } from './pppoe-types';
import type { RadioResult } from '../../features/equipment/radio-api';
const page = (rows: string[], next = ''): PPResult => ({
  code: 'OK',
  profile_id: 'X',
  activation_id: 'a',
  data: {
    total: 51,
    next_cursor: next,
    read: {
      scope: 'health',
      started_at: 'start',
      ended_at: 'snapshot',
      duration_ns: 1,
      rows: rows.map((name) => ({
        name,
        kind: 'uplink',
        uptime: { known: false, min: 0, max: 0 },
      })),
    },
  },
});
it('collects the complete stored table and rejects a failed or changed continuation without returning partial data', async () => {
  const first = page(['primeiro'], 'next');
  const read = vi.fn(async () => page(['último']));
  const full = await completePPTable(first, read, 'read');
  expect(full.data?.read?.rows.map((r) => r.name)).toEqual([
    'primeiro',
    'último',
  ]);
  expect(full.data?.next_cursor).toBe('');
  expect(first.data?.read?.rows).toHaveLength(1);
  for (const bad of [
    { ...page([]), code: 'CURSOR_INVALID' },
    { ...page([]), profile_id: 'Y' },
    page(['outro'], 'next'),
  ]) {
    await expect(
      completePPTable(first, async () => bad, 'read'),
    ).rejects.toThrow();
  }
  const changed = page([]);
  changed.data!.read!.ended_at = 'new snapshot';
  await expect(
    completePPTable(first, async () => changed, 'read'),
  ).rejects.toThrow();
});
it('collects every AP index page and refuses another index or company', async () => {
  const first: RadioResult = {
    code: 'OK',
    profile_id: 'X',
    activation_id: 'a',
    data: {
      total: 2,
      more: true,
      next_offset: 8,
      index: {
        targets: { one: 'OK' },
        labels: { one: 'AP um' },
        observed_at: 'snapshot',
        complete: true,
        total: 2,
        more: true,
        next_offset: 8,
      },
    },
  };
  const last: RadioResult = {
    ...first,
    data: {
      total: 2,
      more: false,
      next_offset: 0,
      index: {
        ...first.data!.index!,
        targets: { two: 'OK' },
        labels: { two: 'AP dois' },
        more: false,
      },
    },
  };
  const read = vi.fn(async () => last);
  const full = await completeRadioIndex(first, read);
  expect(Object.keys(full.data!.index!.targets)).toEqual(['one', 'two']);
  expect(full.data!.index!.labels).toEqual({ one: 'AP um', two: 'AP dois' });
  expect(read).toHaveBeenCalledWith(8);
  expect(first.data!.index!.targets).toEqual({ one: 'OK' });
  const changed = { ...last, profile_id: 'Y' };
  await expect(
    completeRadioIndex(first, async () => changed),
  ).rejects.toThrow();
});
