import { PPPMenuProvider } from './ppp-menu-provider';
import { PPPMenuLinks } from './ppp-menu-links';
import { useContext, useState } from 'react';
import {
  render,
  screen,
  waitFor,
  within,
  fireEvent,
} from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { PPPMenu } from './ppp-menu';
import {
  SessionRoute,
  type SessionRouteTarget,
} from '../equipment/session-navigation';
import {
  pppCatalogAPI,
  pppoeReadAPI,
  equipmentAPI,
} from '../../platform/wails/presentation';
import type {
  Device,
  PPPEntry,
  PPResult,
  PPPQuery,
} from '../../platform/wails/presentation';
vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock(),
);
vi.mock('./ppp-page-samples', () => ({
  usePPPPageSamples: (p: { entries: PPPEntry[] }) => ({
    entries: p.entries,
    samplers: null,
  }),
}));
vi.mock('./pppoe', () => ({
  PPPoE: () => {
    const n = useContext(SessionRoute);
    return (
      <div>
        <h1>Sessão aberta</h1>
        <button onClick={() => n?.back()}>Voltar</button>
      </div>
    );
  },
}));
const devices = [1, 2, 3].map((n) => ({
  id: 'd' + n,
  name: 'CCR ' + n,
  role: 'concentrator',
  active: true,
  fingerprint: 'key',
  credential_id: 'c',
  revision: 1,
})) as Device[];
function Harness({ available = false }: { available?: boolean }) {
  const [target, setTarget] = useState<SessionRouteTarget>();
  return (
    <PPPMenuProvider>
      <PPPMenuLinks navigate={() => {}} />
      <SessionRoute.Provider
        value={{
          active: !!target,
          target,
          open: setTarget,
          back: () => setTarget(undefined),
          leave: () => setTarget(undefined),
        }}
      >
        <PPPMenu
          profile="p"
          activation="a"
          devices={devices}
          available={available}
          openEquipment={() => {}}
        />
      </SessionRoute.Provider>
    </PPPMenuProvider>
  );
}
function response(
  entries: PPPEntry[],
  total = entries.length,
  next = '',
): PPResult {
  return {
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: {
      entries,
      total,
      next_cursor: next,
      options: { state: ['ONLINE'], concentrator: ['CCR 1', 'CCR 2', 'CCR 3'] },
      catalog_dates: Object.fromEntries(
        devices.map((d) => [
          d.id,
          new Date(Date.now() - 120_000).toISOString(),
        ]),
      ),
    },
  };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(pppCatalogAPI.list).mockImplementation(async (_p, _a, _d, q) =>
    response(
      devices.map((d) => ({
        id: d.id,
        kind: q.screen === 'servers' ? 'server' : 'profile',
        device_id: d.id,
        concentrator: d.name,
        name: d.name,
        online: 0,
        registrations: 0,
      })),
    ),
  );
  vi.mocked(equipmentAPI.get).mockImplementation(async (_p, _a, id) => ({
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: {
      device: devices.find((d) => d.id === id),
      devices: [],
      credentials: [],
      next_cursor: 0,
      vault_available: true,
    },
  }));
  vi.mocked(pppoeReadAPI.start).mockResolvedValue({
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: { total: 0, job: { id: 'j', state: 'SUCCEEDED', code: 'OK' } },
  });
  vi.mocked(pppoeReadAPI.status).mockResolvedValue({
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: { total: 0, job: { id: 'j', state: 'SUCCEEDED', code: 'OK' } },
  });
});
it('opens saved data from three concentrators and reads each stale catalog only once across five screens', async () => {
  render(<Harness available />);
  await waitFor(() => expect(pppoeReadAPI.start).toHaveBeenCalledTimes(3));
  fireEvent.click(screen.getByRole('link', { name: 'Servidores PPPoE' }));
  const table = await screen.findByRole('table', { name: 'Servidores PPPoE' });
  expect(within(table).getAllByText('CCR 1')).toHaveLength(2);
  fireEvent.click(screen.getByRole('link', { name: 'Planos' }));
  fireEvent.click(screen.getByRole('link', { name: 'Cadastros' }));
  expect(pppoeReadAPI.start).toHaveBeenCalledTimes(3);
  expect(
    new Set(vi.mocked(pppoeReadAPI.start).mock.calls.map((c) => c[2])),
  ).toEqual(new Set(['d1', 'd2', 'd3']));
});
it('keeps disabled and invalid servers visible with red invalid state', async () => {
  vi.mocked(pppCatalogAPI.list).mockResolvedValue(
    response([
      {
        id: '1',
        kind: 'server',
        device_id: 'd1',
        concentrator: 'CCR 1',
        name: 'desativado',
        disabled: true,
        online: 0,
        registrations: 0,
      },
      {
        id: '2',
        kind: 'server',
        device_id: 'd1',
        concentrator: 'CCR 1',
        name: 'inválido',
        invalid: true,
        online: 0,
        registrations: 0,
      },
    ]),
  );
  render(<Harness />);
  fireEvent.click(screen.getByRole('link', { name: 'Servidores PPPoE' }));
  const table = await screen.findByRole('table', { name: 'Servidores PPPoE' });
  await waitFor(() =>
    expect(within(table).getAllByText('Inválido')).toHaveLength(2),
  );
  for (const state of within(table).getAllByText('Inválido'))
    expect(state).toHaveClass('st-off');
  expect(within(table).getByText('desativado')).toBeInTheDocument();
});
it('returns from a session to the same PPP search filters and page', async () => {
  const entry: PPPEntry = {
    id: 'r',
    kind: 'session',
    device_id: 'd1',
    concentrator: 'CCR 1',
    name: 'ana',
    observation_id: 'o',
    online: 0,
    registrations: 0,
    session: {
      row_id: 'r',
      remote_id: '*1',
      session_id: 's',
      login: 'ana',
      address: '192.0.2.1',
      caller_id: '',
      state: 'ONLINE',
      uptime: { known: false, min: 0, max: 0 },
      duplicates: 1,
    },
  };
  vi.mocked(pppCatalogAPI.list).mockImplementation(
    async (_p, _a, _d, q: PPPQuery, cursor) =>
      response(
        q.screen === 'sessions' ? [entry] : [],
        100,
        cursor ? '' : 'next',
      ),
  );
  render(<Harness />);
  const table = await screen.findByRole('table', { name: 'Sessões' });
  const region = table.closest('.block') || table.parentElement!.parentElement!;
  const search = within(region as HTMLElement).getByPlaceholderText(
    'Buscar em todas as colunas',
  );
  await waitFor(() =>
    expect(
      within(region as HTMLElement).getByRole('option', { name: 'ONLINE' }),
    ).toBeInTheDocument(),
  );
  fireEvent.change(search, { target: { value: 'Ana' } });
  fireEvent.change(
    within(region as HTMLElement).getByRole('combobox', { name: 'Estado' }),
    { target: { value: 'ONLINE' } },
  );
  await waitFor(() =>
    expect(
      within(region as HTMLElement).getByRole('button', { name: 'Próxima' }),
    ).not.toBeDisabled(),
  );
  fireEvent.click(
    within(region as HTMLElement).getByRole('button', { name: 'Próxima' }),
  );
  await waitFor(() =>
    expect(
      within(region as HTMLElement).getByText('Página 2 de 2'),
    ).toBeInTheDocument(),
  );
  fireEvent.click(within(table).getByRole('button', { name: 'ana' }));
  await screen.findByRole('heading', { name: 'Sessão aberta' });
  fireEvent.click(screen.getByRole('button', { name: 'Voltar' }));
  expect(search).toHaveValue('Ana');
  expect(
    within(region as HTMLElement).getByRole('combobox', { name: 'Estado' }),
  ).toHaveValue('ONLINE');
  expect(
    within(region as HTMLElement).getByText('Página 2 de 2'),
  ).toBeInTheDocument();
});
