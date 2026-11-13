import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import {
  DeviceForm,
  ProtectedDialog,
} from '../../features/equipment/equipment-forms';
import type {
  Credential,
  Device,
  EquipmentResult,
} from '../../features/equipment/equipment-api';
import {
  RadioRoles,
  RadioTransports,
} from '../../features/equipment/radio-contract';

it('keeps group edits protected and applies the default credential when changing radio role', async () => {
  const device = {
    id: 'synthetic',
    revision: 1,
    name: 'Switch',
    role: 'switch',
    group_id: '11111111-1111-4111-8111-111111111111',
    address: '192.0.2.1',
    port: 22,
    transport: RadioTransports.ssh,
    credential_id: 'old',
    active: true,
  } as Device;
  const credentials = [
    { id: 'old', alias: 'Anterior', user: 'u', revision: 1, links: 1 },
    { id: 'radio-default', alias: 'Rádio', user: 'u', revision: 1, links: 0 },
  ] as Credential[];
  const save = vi
    .fn()
    .mockResolvedValue({ code: 'OK', data: { device } } as EquipmentResult);
  render(
    <ProtectedDialog title="Editar equipamento" close={vi.fn()}>
      <DeviceForm
        defaults={{ [RadioRoles.ap]: 'radio-default' }}
        device={device}
        groups={[
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
        ]}
        credentials={credentials}
        disabled={false}
        save={save}
      />
    </ProtectedDialog>,
  );
  expect(screen.getByLabelText('Grupo')).toHaveValue(
    '11111111-1111-4111-8111-111111111111',
  );
  fireEvent.change(screen.getByLabelText('Grupo'), {
    target: { value: '22222222-2222-4222-8222-222222222222' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Cancelar' }));
  expect(
    await screen.findByRole('dialog', { name: 'Descartar alterações?' }),
  ).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Continuar editando' }));
  fireEvent.change(screen.getByLabelText('Papel do equipamento'), {
    target: { value: RadioRoles.ap },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Salvar equipamento' }));
  await waitFor(() =>
    expect(save).toHaveBeenCalledWith(
      expect.objectContaining({
        group_id: '22222222-2222-4222-8222-222222222222',
        role: RadioRoles.ap,
        credential_id: 'radio-default',
      }),
    ),
  );
});
