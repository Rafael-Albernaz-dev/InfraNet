import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, afterEach, expect, it, vi } from 'vitest';
import { RadioTransports } from './radio-contract';
import { AddEquipment } from './add-equipment';
import { equipmentAPI } from './equipment-api';
import type { EquipmentResult, Device } from './equipment-api';
import type { useEquipmentState } from './equipment-state';
import { inventoryImportAPI } from '../../platform/wails/presentation';
import { pendingEnrollments } from '../access/equipment-receipts';

function result(
  data: Partial<NonNullable<EquipmentResult['data']>> = {},
): EquipmentResult {
  return {
    code: 'OK',
    profile_id: 'fixture',
    activation_id: 'active',
    data: {
      devices: [],
      credentials: [],
      next_cursor: 0,
      vault_available: true,
      ...data,
    },
  };
}
function mountRegistration() {
  const state = {
    busy: false,
    vault: true,
    credentials: [
      {
        id: 'credential',
        alias: 'Reader',
        user: 'reader',
        revision: 1,
        links: 0,
      },
    ],
    credentialDefaults: { ap: 'credential' },
    groups: [],
    reload: vi.fn(),
    reloadCredentials: vi.fn(),
    setCredentialForm: vi.fn(),
    perform: async (call: () => Promise<EquipmentResult>) => call(),
  } as unknown as ReturnType<typeof useEquipmentState>;
  const back = vi.fn();
  const mounted = render(
    <AddEquipment
      s={state}
      profile="fixture"
      activation="active"
      locked={false}
      back={back}
    />,
  );
  return { ...mounted, back };
}
beforeEach(() => {
  pendingEnrollments.clear();
  vi.spyOn(inventoryImportAPI, 'status').mockResolvedValue(result());
});
afterEach(() => vi.restoreAllMocks());

it('recovers a failed registration-status response with the original receipt and one submission', async () => {
  let deviceID = '';
  const job = {
    id: 'register-job',
    device_id: '',
    kind: 'inventory.register',
    state: 'RUNNING',
    code: 'OK',
  };
  vi.spyOn(equipmentAPI, 'save').mockImplementation(async (_p, _a, input) => {
    deviceID = input.id;
    job.device_id = input.id;
    return result({ job });
  });
  const device = {
    id: '',
    name: 'CCR lido',
    address: '192.0.2.1',
    port: 22,
    active: true,
    revision: 1,
    transport: RadioTransports.ssh,
    credential_id: 'credential',
    block: '',
    fingerprint: 'SHA256:fixture',
    trust_origin: 'automatic-tofu',
    result: {
      identity: 'CCR lido',
      board: 'CCR2004',
      version: '7.2',
      family: 'v7',
      uptime: '1h',
      cpu_percent: 1,
      free_memory_bytes: 1,
      total_memory_bytes: 2,
      interfaces: [],
      observed_at: new Date().toISOString(),
      source: 'SSH',
      quality: 'complete',
    },
  } as Device;
  vi.spyOn(equipmentAPI, 'registrationStatus')
    .mockRejectedValueOnce(new Error('lost response'))
    .mockImplementation(async () =>
      result({
        job: { ...job, state: 'SUCCEEDED' },
        device: { ...device, id: deviceID },
      }),
    );
  vi.spyOn(equipmentAPI, 'enrollmentStatus').mockImplementation(async () =>
    result({ operation_state: 'RUNNING', job }),
  );
  mountRegistration();
  fireEvent.change(screen.getByLabelText('IP de gerenciamento'), {
    target: { value: '192.0.2.1' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Salvar' }));
  await screen.findByRole('button', { name: 'Conferir cadastro' });
  expect(equipmentAPI.save).toHaveBeenCalledTimes(1);
  const input = vi.mocked(equipmentAPI.save).mock.calls[0][2];
  expect(pendingEnrollments.get('fixture|active')).toEqual({
    id: input.id,
    operation_id: input.operation_id,
  });
  fireEvent.click(screen.getByRole('button', { name: 'Conferir cadastro' }));
  expect(
    await screen.findByRole('button', { name: 'Ver na lista' }),
  ).toBeEnabled();
  expect(
    screen.getByText('Cadastrado: CCR lido · CCR2004 · 7.2'),
  ).toBeVisible();
  expect(equipmentAPI.enrollmentStatus).toHaveBeenCalledWith(
    'fixture',
    'active',
    input.id,
    input.operation_id,
  );
  expect(equipmentAPI.save).toHaveBeenCalledTimes(1);
  expect(pendingEnrollments.has('fixture|active')).toBe(false);
});

it('restores the service import queue after leaving and cancels only the remaining rows', async () => {
  let cancelled = false;
  const row = {
    line: 1,
    name: '',
    address: '192.0.2.1',
    port: '22',
    role: 'AP',
    block: '',
    credential: 'Reader',
    group: 'Norte',
  };
  const running = () =>
    result({
      operation_state: cancelled ? 'COMPLETED' : 'RUNNING',
      import_items: [
        { row, status: 'registered', identity: 'AP lido' },
        {
          row: { ...row, line: 2, address: '192.0.2.2' },
          status: cancelled ? 'cancelled' : 'queued',
          reason: cancelled ? 'Cancelada' : undefined,
        },
      ],
      import_receipt: cancelled ? { created: 1, ignored: 1 } : undefined,
    });
  vi.mocked(inventoryImportAPI.status).mockImplementation(async () =>
    running(),
  );
  vi.spyOn(inventoryImportAPI, 'cancel').mockImplementation(async () => {
    cancelled = true;
    return running();
  });
  vi.spyOn(inventoryImportAPI, 'confirm').mockResolvedValue(result());
  const first = mountRegistration();
  fireEvent.click(screen.getByRole('tab', { name: 'Importar arquivo' }));
  expect(await screen.findByText('Carregando – 50%')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Voltar' }));
  expect(first.back).not.toHaveBeenCalled();
  expect(
    screen.getByRole('dialog', { name: 'Sair da importação?' }),
  ).toBeVisible();
  expect(
    screen.getByText(
      'A fila continua no serviço. Os equipamentos cadastrados aparecem na lista.',
    ),
  ).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Ficar na importação' }));
  expect(first.back).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Voltar' }));
  fireEvent.click(screen.getByRole('button', { name: 'Sair da tela' }));
  expect(first.back).toHaveBeenCalledOnce();
  expect(inventoryImportAPI.cancel).not.toHaveBeenCalled();
  first.unmount();
  mountRegistration();
  fireEvent.click(screen.getByRole('tab', { name: 'Importar arquivo' }));
  expect(await screen.findByText('Carregando – 50%')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Cancelar o restante' }));
  expect(await screen.findByText('Importação concluída')).toBeVisible();
  expect(screen.getByText('Cancelada')).toBeVisible();
  expect(
    screen.getByRole('button', { name: 'Baixar linhas descartadas (.csv)' }),
  ).toBeVisible();
  await waitFor(() =>
    expect(inventoryImportAPI.cancel).toHaveBeenCalledExactlyOnceWith(
      'fixture',
      'active',
    ),
  );
  expect(inventoryImportAPI.confirm).not.toHaveBeenCalled();
});

it('updates the open sheet name from the next inventory read without starting another remote query', async () => {
  const { Equipment } = await import('../../app/pages/equipment');
  let current = {
    id: 'device',
    name: 'Identidade antiga',
    identity: 'Identidade antiga',
    block: '',
    group_id: '',
    group_path: '',
    address: '192.0.2.1',
    port: 22,
    transport: RadioTransports.ssh,
    role: 'other',
    credential_id: 'credential',
    revision: 1,
    active: true,
    fingerprint: '',
    trust_origin: '',
  } as Device;
  vi.spyOn(equipmentAPI, 'list').mockImplementation(async () =>
    result({ devices: [current] }),
  );
  vi.spyOn(equipmentAPI, 'get').mockImplementation(async () =>
    result({ device: current }),
  );
  vi.spyOn(equipmentAPI, 'credentials').mockResolvedValue(result());
  vi.spyOn(equipmentAPI, 'start').mockResolvedValue(result());
  vi.spyOn(equipmentAPI, 'inspect').mockResolvedValue(result());
  const { baseElement } = render(
    <Equipment profile="fixture" activation="active" connected />,
  );
  fireEvent.click(
    await screen.findByRole('cell', { name: 'Identidade antiga' }),
  );
  await screen.findByRole('heading', { name: 'Identidade antiga' });
  await waitFor(() => expect(equipmentAPI.get).toHaveBeenCalledTimes(1));
  current = {
    ...current,
    name: 'Identidade nova',
    identity: 'Identidade nova',
  };
  fireEvent(baseElement.ownerDocument, new Event('visibilitychange'));
  expect(
    await screen.findByRole('heading', { name: 'Identidade nova' }),
  ).toBeVisible();
  expect(equipmentAPI.get).toHaveBeenCalledTimes(1);
  expect(equipmentAPI.start).not.toHaveBeenCalled();
  expect(equipmentAPI.inspect).not.toHaveBeenCalled();
});
