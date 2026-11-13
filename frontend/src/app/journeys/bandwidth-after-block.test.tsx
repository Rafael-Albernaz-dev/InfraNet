import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { Equipment } from '../pages/equipment';
import { openDeviceSheet } from '../test-navigation';
import type {
  Device,
  DeviceInput,
} from '../../features/equipment/equipment-api';
import { RadioTransports } from '../../features/equipment/radio-contract';
function testDevice() {
  const stamp = new Date().toISOString();
  return {
    id: 'synthetic-concentrator',
    name: 'Concentrador de teste',
    role: 'concentrator',
    block: '',
    group_id: '11111111-1111-4111-8111-111111111111',
    group_path: 'Torre Norte',
    address: '192.0.2.44',
    port: 22,
    transport: RadioTransports.ssh,
    credential_id: 'synthetic-credential',
    revision: 1,
    active: true,
    fingerprint: 'synthetic-pin',
    trust_origin: 'synthetic',
    result: {
      observed_at: stamp,
      identity: 'Resumo salvo',
      family: 'v6',
      source: 'SSH',
      quality: 'complete',
      free_memory_bytes: 1,
      total_memory_bytes: 2,
      board: 'Synthetic',
      uptime: '1h',
      cpu_percent: 1,
      interfaces: [],
      version: '6.49',
    },
  } as Device;
}
function fixture() {
  const stamp = new Date().toISOString();
  let device = testDevice();
  const row = {
    row_id: 'saved-row',
    login: 'cliente-sintetico',
    remote_id: '*0',
    state: 'ONLINE',
    uptime: { known: true, min: 1e9, max: 1e9 },
  };
  const credential = {
    id: device.credential_id,
    alias: 'Consulta',
    user: 'reader',
    revision: 1,
    links: 1,
  };
  const result = (data: object = {}) => ({
    code: 'OK',
    profile_id: 'company',
    activation_id: 'active',
    data: {
      total: 51,
      total_devices: 1,
      devices: [device],
      credentials: [credential],
      groups: [
        {
          id: '11111111-1111-4111-8111-111111111111',
          parent_id: '',
          name: 'Torre Norte',
        },
        {
          id: '22222222-2222-4222-8222-222222222222',
          parent_id: '',
          name: 'Torre Sul',
        },
      ],
      next_cursor: 0,
      vault_available: true,
      ...data,
    },
  });
  const inventory = {
    ListEquipment: vi.fn(async () => result()),
    ListEquipmentFiltered: vi.fn(async () => result()),
    ListCredentials: vi.fn(async () => result()),
    GetEquipment: vi.fn(async () => result({ device })),
    SaveEquipment: vi.fn(async (_p: string, _a: string, input: DeviceInput) => {
      device = {
        ...device,
        group_id: input.group_id,
        group_path:
          input.group_id === '22222222-2222-4222-8222-222222222222'
            ? 'Torre Sul'
            : 'Torre Norte',
      };
      return result({ device });
    }),
    StartQuery: vi.fn(async () => result()),
    GetDeviceMonitoring: vi.fn(async () => result({ rows: [] })),
  };
  const sessions = {
    GetPPPoECurrent: vi.fn(async () =>
      result({
        current: {
          observation_id: 'saved-before-block',
          count: 1,
          provenance: { ended_at: stamp },
          quality: { status: 'complete' },
        },
      }),
    ),
    ListPPPoEReferences: vi.fn(async () => result({ references: [] })),
    ListPPPoESessions: vi.fn(async () => result({ rows: [row] })),
    PPPoEContextCounts: vi.fn(async () => result({ groups: [] })),
    StartPPPoECollection: vi.fn(async () => result()),
  };
  const reads = {
    StartPPPoERead: vi.fn(async () =>
      result({ job: { id: 'bandwidth', state: 'RUNNING', code: 'OK' } }),
    ),
    PPPoEReadStatus: vi.fn(async () =>
      result({
        job: { id: 'bandwidth', state: 'SUCCEEDED', code: 'OK' },
        read: {
          scope: 'bandwidth',
          started_at: stamp,
          ended_at: stamp,
          duration_ns: 2e9,
          rows: [
            {
              ...row,
              kind: 'session',
              download: { min: 2000000, max: 2000000 },
              upload: { min: 500000, max: 500000 },
            },
          ],
        },
      }),
    ),
    CancelPPPoERead: vi.fn(async () => result()),
    CancelPPPoEReadOperation: vi.fn(async () => result()),
    StopPPPoETraffic: vi.fn(async () => result()),
  };
  vi.stubGlobal('go', {
    presentation: { App: { ...inventory, ...sessions, ...reads } },
  });
  return { inventory, sessions, reads };
}
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
it('edits only the concentrator group, reopens saved sessions and measures bandwidth on the page', async () => {
  const { inventory, sessions, reads } = fixture();
  render(<Equipment profile="company" activation="active" connected />);
  await openDeviceSheet('Concentrador de teste');
  fireEvent.click(screen.getByRole('button', { name: 'Editar equipamento' }));
  fireEvent.change(
    within(
      screen.getByRole('dialog', { name: 'Editar equipamento' }),
    ).getByLabelText('Grupo'),
    {
      target: { value: '22222222-2222-4222-8222-222222222222' },
    },
  );
  fireEvent.click(screen.getByRole('button', { name: 'Salvar equipamento' }));
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  );
  expect(inventory.SaveEquipment).toHaveBeenCalledExactlyOnceWith(
    'company',
    'active',
    expect.objectContaining({
      group_id: '22222222-2222-4222-8222-222222222222',
      revision: 1,
      id: 'synthetic-concentrator',
    }),
  );
  expect(
    screen.getByRole('region', { name: 'Ficha do equipamento' }),
  ).toHaveTextContent('Torre Sul');
  fireEvent.click(screen.getByRole('tab', { name: 'Sessões' }));
  expect(await screen.findByText('cliente-sintetico')).toBeVisible();
  fireEvent.click(await screen.findByRole('button', { name: 'Medir banda' }));
  const result = within(
    screen.getByRole('region', { name: 'Banda por sessão' }),
  );
  expect(await result.findByText('2 Mbps')).toBeVisible();
  expect(result.getByText('500 Kbps')).toBeVisible();
  expect(reads.StartPPPoERead).toHaveBeenCalledExactlyOnceWith(
    'company',
    'active',
    'synthetic-concentrator',
    expect.any(String),
    'bandwidth',
    'saved-before-block',
    ['saved-row'],
    1,
  );
  expect(sessions.StartPPPoECollection).not.toHaveBeenCalled();
  expect(inventory.StartQuery).not.toHaveBeenCalled();
});
