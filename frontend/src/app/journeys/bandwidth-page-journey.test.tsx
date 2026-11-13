import { RadioModes } from '../../features/equipment/radio-contract';
import { waitForSessionReady } from '../../shared/testing/session-ready';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { PPPoE } from '../pages/pppoe-with-link';
import type { Device } from '../../features/equipment/equipment-api';
const device = {
  id: 'synthetic-concentrator',
  name: 'Concentrador sintético',
  active: true,
  revision: 1,
  fingerprint: 'synthetic-pin',
} as Device;
function journey() {
  const rows = [1, 2].map((i) => ({
    row_id: `row-${i}`,
    remote_id: `*${i}`,
    login: `cliente-${i}`,
    address: `192.0.2.${i}`,
    state: 'ONLINE',
    uptime: { known: true, min: 1e9, max: 1e9 },
  }));
  const result = (data = {}) => ({
    code: 'OK',
    profile_id: 'company',
    activation_id: 'active',
    data: { total: rows.length, ...data },
  });
  const status = (
    state = 'RUNNING',
    id: string = RadioModes.live,
    selected = rows,
  ) =>
    result({
      job: { id, state, code: 'OK' },
      ...(state === 'SUCCEEDED' || state === 'RUNNING'
        ? {
            read: {
              scope: id === 'traffic' ? 'traffic' : RadioModes.live,
              started_at: '2026-11-04T12:00:00Z',
              ended_at: '2026-11-04T12:00:02Z',
              duration_ns: 2e9,
              rows: selected.map((r) => ({
                ...r,
                kind: 'session',
                download: { min: 2_000_000, max: 2_000_000 },
                upload: { min: 500_000, max: 500_000 },
              })),
            },
          }
        : {}),
    });
  const bindings = {
    GetPPPoECurrent: vi.fn(async () =>
      result({
        current: {
          observation_id: 'saved',
          count: 2,
          provenance: { ended_at: '2026-11-04T12:00:00Z' },
          quality: { status: 'complete' },
        },
      }),
    ),
    ListPPPoEReferences: vi.fn(async () => result({ references: [] })),
    ListPPPoESessions: vi.fn(async () => result({ rows })),
    GetPPPoESession: vi.fn(async () => result({ rows: [rows[0]] })),
    PPPoEContextCounts: vi.fn(async () => result({ groups: [] })),
    GetDeviceMonitoring: vi.fn(async () => result()),
    StartPPPoERead: vi.fn(async (...args: unknown[]) =>
      result({ job: { id: String(args[4]), state: 'RUNNING', code: 'OK' } }),
    ),
    PPPoEReadStatus: vi.fn(async () => status()),
    CancelPPPoERead: vi.fn(async () => result()),
    CancelPPPoEReadOperation: vi.fn(async () => result()),
    StopPPPoETraffic: vi.fn(async () => result()),
  };
  vi.stubGlobal('go', { presentation: { App: bindings } });
  const view = render(
    <PPPoE
      profile="company"
      activation="active"
      device={device}
      back={() => {}}
    />,
  );
  return { rows, bindings, status, view };
}
afterEach(() => {
  cleanup();
  vi.stubGlobal('go', undefined);
});

it('lê automaticamente somente a página e publica o instante depois da resposta', async () => {
  const { bindings, status } = journey();
  let finish!: (r: ReturnType<typeof status>) => void;
  bindings.PPPoEReadStatus.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  await screen.findByRole('columnheader', { name: 'Download' });
  await waitFor(() => expect(bindings.PPPoEReadStatus).toHaveBeenCalledOnce());
  expect(bindings.StartPPPoERead).toHaveBeenCalledExactlyOnceWith(
    'company',
    'active',
    device.id,
    expect.any(String),
    'live',
    'saved',
    ['row-1', 'row-2'],
    1,
  );
  expect(
    screen.queryByRole('button', { name: 'Medir banda da página' }),
  ).not.toBeInTheDocument();
  await act(async () => finish(status()));
  const table = within(screen.getByRole('table', { name: 'Sessões PPPoE' }));
  expect(await table.findAllByText('2 Mbps')).toHaveLength(2);
  expect(table.getAllByText('500 Kbps')).toHaveLength(2);
  expect(bindings.StartPPPoERead).toHaveBeenCalledOnce();
});
it('sair da página cancela a leitura e ignora sua resposta tardia', async () => {
  const { bindings, status, view } = journey();
  let finish!: (r: ReturnType<typeof status>) => void;
  bindings.PPPoEReadStatus.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  await waitFor(() => expect(bindings.PPPoEReadStatus).toHaveBeenCalledOnce());
  view.unmount();
  await waitFor(() =>
    expect(bindings.CancelPPPoERead).toHaveBeenCalledWith(
      'company',
      'active',
      'live',
    ),
  );
  await act(async () => finish(status()));
  expect(screen.queryByText('2 Mbps')).not.toBeInTheDocument();
  expect(bindings.StartPPPoERead).toHaveBeenCalledOnce();
});
it('falha de andamento cancela o job conhecido antes de renovar sem sobreposição', async () => {
  vi.useFakeTimers();
  try {
    const { bindings, status } = journey();
    bindings.PPPoEReadStatus.mockRejectedValueOnce(
      new Error('synthetic lost status'),
    ).mockResolvedValue(status());
    await vi.waitFor(() =>
      expect(bindings.CancelPPPoERead).toHaveBeenCalledWith(
        'company',
        'active',
        'live',
      ),
    );
    expect(bindings.StartPPPoERead).toHaveBeenCalledOnce();
    await act(async () => vi.advanceTimersByTimeAsync(5000));
    await vi.waitFor(() =>
      expect(bindings.StartPPPoERead).toHaveBeenCalledTimes(2),
    );
    expect(screen.getAllByText('2 Mbps')).toHaveLength(2);
  } finally {
    vi.useRealTimers();
  }
});
it('a sessão inicia tráfego só para sua linha e mostra o instante no cabeçalho', async () => {
  const { bindings, rows, status } = journey();
  bindings.PPPoEReadStatus.mockImplementation(async () =>
    status('RUNNING', 'traffic', [rows[0]]),
  );
  fireEvent.click(
    await screen.findByRole('button', { name: 'Detalhar cliente-1 *1' }),
  );
  const panel = within(await waitForSessionReady());
  await waitFor(() =>
    expect(bindings.StartPPPoERead).toHaveBeenCalledWith(
      'company',
      'active',
      device.id,
      expect.any(String),
      'traffic',
      'saved',
      ['row-1'],
      1,
    ),
  );
  expect(await panel.findByText(/Ao vivo ↓ 2 Mbps/)).toBeVisible();
  expect(
    panel.queryByRole('button', { name: 'Medir banda' }),
  ).not.toBeInTheDocument();
});
