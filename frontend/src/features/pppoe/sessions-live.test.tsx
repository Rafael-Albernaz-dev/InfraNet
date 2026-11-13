import { render, screen, within, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { PPPoESessionsTable } from './pppoe-sessions-view';
import type { PPRow } from '../../platform/wails/pppoe-types';
import type { Device } from '../equipment/equipment-api';
const calls = vi.hoisted(() => ({
  start: vi.fn(),
  status: vi.fn(),
  cancel: vi.fn(),
  cancelOperation: vi.fn(),
}));
vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock({
    pppoeReadAPI: calls,
  }),
);

const row = (i: number): PPRow => ({
  row_id: `r${i}`,
  remote_id: `*${i}`,
  session_id: `s${i}`,
  login: `fixture-${i}`,
  address: '192.0.2.1',
  caller_id: '02:00:00:00:00:01',
  uptime: { known: true, min: 1e12, max: 1e12 },
  duplicates: 1,
});
it('shows missing rates for OFFLINE even if its saved observation has numeric rates', () => {
  render(
    <PPPoESessionsTable
      rows={[
        {
          ...row(0),
          state: 'OFFLINE',
          download: { min: 0, max: 0 },
          upload: { min: 9e6, max: 9e6 },
        },
      ]}
      onDetail={() => {}}
    />,
  );
  const cells = within(
    screen.getByRole('row', { name: /fixture-0/ }),
  ).getAllByRole('cell');
  expect(cells[3]).toHaveTextContent('—');
  expect(cells[4]).toHaveTextContent('—');
});
it('starts one live read for only the fifty visible ONLINE rows and cancels on leaving', async () => {
  calls.start.mockResolvedValue({
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: { job: { id: 'job', state: 'RUNNING', code: 'OK' } },
  });
  calls.status.mockImplementation(() => new Promise(() => {}));
  calls.cancel.mockResolvedValue({ code: 'OK' });
  calls.cancelOperation.mockResolvedValue({ code: 'OK' });
  const rows = Array.from({ length: 50 }, (_, i) => row(i));
  const view = render(
    <PPPoESessionsTable
      rows={rows}
      device={{ id: 'd', revision: 1 } as Device}
      profile="p"
      activation="a"
      observation="o"
      onDetail={() => {}}
    />,
  );
  await waitFor(() => expect(calls.start).toHaveBeenCalledTimes(1));
  expect(calls.start.mock.calls[0][4]).toBe('live');
  expect(calls.start.mock.calls[0][6]).toEqual(rows.map((r) => r.row_id));
  await waitFor(() => expect(calls.status).toHaveBeenCalledTimes(1));
  view.unmount();
  await waitFor(() =>
    expect(calls.cancel).toHaveBeenCalledWith('p', 'a', 'job'),
  );
  expect(calls.start).toHaveBeenCalledTimes(1);
});
