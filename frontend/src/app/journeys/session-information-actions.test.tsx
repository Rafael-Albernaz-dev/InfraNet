import { RadioModes } from '../../features/equipment/radio-contract';
import type { PPRow } from '../../platform/wails/presentation';
import { createRef } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { PPPoESessionDetail } from '../../features/pppoe/pppoe-session-screen';
import {
  pppoeDiagnosticAPI,
  pppoeReadAPI,
  radioAPI,
} from '../../platform/wails/presentation';
import type { Device } from '../../features/equipment/equipment-api';
import type { PPObservation } from '../../platform/wails/pppoe-types';

vi.mock('../../platform/wails/presentation', async (importOriginal) =>
  (await import('../../shared/testing/presentation-mock')).presentationMock(
    {},
    Object.fromEntries(
      Object.entries(
        await importOriginal<Record<string, Record<string, unknown>>>(),
      ).filter(
        ([name]) =>
          !['pppoeDiagnosticAPI', 'pppoeReadAPI', 'radioAPI'].includes(name),
      ),
    ),
  ),
);
const response = (data: object = {}) => ({
  code: 'OK',
  profile_id: 'company',
  activation_id: 'active',
  data: { total: 0, ...data },
});
const device = {
  id: 'router-fixture',
  name: 'Concentrador sintético',
  revision: 1,
  active: true,
  fingerprint: 'pinned',
} as Device;
const selected = {
  row_id: 'row-fixture',
  login: 'usuario sintético',
  state: 'ONLINE',
  address: '192.0.2.10',
  uptime: { known: true, min: 1, max: 1 },
} as PPRow;
const current = {
  observation_id: 'obs-fixture',
  provenance: { ended_at: '2026-11-06T00:00:00Z' },
  context_omitted: true,
} as PPObservation;
function renderInformation() {
  render(
    <PPPoESessionDetail
      selected={selected}
      current={current}
      detailHeadingRef={createRef()}
      detailTab="information"
      setDetailTab={() => {}}
      profile="company"
      activation="active"
      device={device}
      disabled={false}
      detailIntentRef={{ current: 0 }}
      detailTriggerRef={{ current: null }}
      select={() => {}}
      back={() => {}}
    />,
  );
}
beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(radioAPI.link).mockResolvedValue(response() as never);
  vi.mocked(pppoeDiagnosticAPI.current).mockResolvedValue(response() as never);
  vi.mocked(pppoeDiagnosticAPI.list).mockResolvedValue(response() as never);
  vi.mocked(pppoeDiagnosticAPI.cancel).mockResolvedValue(response() as never);
  vi.mocked(pppoeReadAPI.cancel).mockResolvedValue(response() as never);
  vi.mocked(pppoeReadAPI.cancelOperation).mockResolvedValue(
    response() as never,
  );
});
it('reopens stored ping and cancels the exact terminal after an uncertain poll', async () => {
  vi.mocked(pppoeReadAPI.start).mockImplementation(
    async (...args) =>
      response({ job: { id: args[4], state: 'RUNNING', code: 'OK' } }) as never,
  );
  vi.mocked(pppoeReadAPI.status).mockImplementation(async (_p, _a, job) => {
    if (job === 'ping-window') throw new Error('synthetic status');
    return response({
      job: { id: job, state: 'SUCCEEDED', code: 'OK' },
    }) as never;
  });
  renderInformation();
  fireEvent.click(
    await screen.findByRole('button', { name: 'Reabrir sondagem armazenada' }),
  );
  await waitFor(() =>
    expect(pppoeDiagnosticAPI.current).toHaveBeenCalledTimes(2),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Ping' }));
  await screen.findByRole('dialog');
  await waitFor(() =>
    expect(pppoeReadAPI.start).toHaveBeenCalledWith(
      'company',
      'active',
      'router-fixture',
      expect.any(String),
      'ping-window',
      'obs-fixture',
      ['row-fixture'],
      1,
    ),
  );
  await screen.findByRole('alert');
  expect(
    vi
      .mocked(pppoeReadAPI.status)
      .mock.calls.filter((args) => args[2] === 'ping-window'),
  ).toHaveLength(1);
  fireEvent.click(screen.getByRole('button', { name: 'Fechar' }));
  await waitFor(() =>
    expect(pppoeReadAPI.cancel).toHaveBeenCalledWith(
      'company',
      'active',
      'ping-window',
    ),
  );
  expect(pppoeDiagnosticAPI.ping).not.toHaveBeenCalled();
});

it('starts log and secret evidence from the new session screen for its selected row', async () => {
  vi.mocked(pppoeReadAPI.start).mockImplementation(
    async (...args) =>
      response({
        job: { id: `job-${args[4]}`, state: 'RUNNING', code: 'OK' },
      }) as never,
  );
  vi.mocked(pppoeReadAPI.status).mockImplementation(
    async (_p, _a, id) =>
      response({ job: { id, state: 'RUNNING', code: 'OK' } }) as never,
  );
  renderInformation();
  fireEvent.click(await screen.findByRole('button', { name: 'Log PPP' }));
  fireEvent.click(
    screen.getByRole('button', { name: 'Ver evidência do secret' }),
  );
  await waitFor(() =>
    expect(
      vi
        .mocked(pppoeReadAPI.start)
        .mock.calls.filter((args) => args[4] !== RadioModes.live),
    ).toHaveLength(2),
  );
  const calls = vi
    .mocked(pppoeReadAPI.start)
    .mock.calls.filter((args) => args[4] !== RadioModes.live);
  expect(calls.map((args) => args.slice(4, 8))).toEqual([
    ['log', 'obs-fixture', ['row-fixture'], 1],
    ['secret', 'obs-fixture', ['row-fixture'], 1],
  ]);
});
