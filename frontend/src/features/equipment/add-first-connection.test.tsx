import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { AddEquipment } from './add-equipment';
import { equipmentAPI } from './equipment-api';
import { pendingEnrollments } from '../access/equipment-receipts';
import type { useEquipmentState } from './equipment-state';

it('shows the registration result and leaves through the explicit list action', async () => {
  pendingEnrollments.clear();
  const result = {
    profile_id: 'fixture',
    activation_id: 'fixture',
    code: 'OK',
  };
  vi.spyOn(equipmentAPI, 'save').mockResolvedValue(result);
  const continueEnrollment = vi.fn();
  const back = vi.fn();
  const s = {
    busy: false,
    credentials: [],
    credentialDefaults: {},
    blocks: [],
    setCredentialForm: vi.fn(),
    reloadCredentials: vi.fn(),
    reload: vi.fn(),
    continueEnrollment,
    perform: async (call: () => Promise<unknown>) => await call(),
  } as unknown as ReturnType<typeof useEquipmentState>;
  render(
    <AddEquipment
      s={s}
      profile="fixture"
      activation="fixture"
      locked={false}
      back={back}
    />,
  );
  expect(screen.queryByLabelText('Nome')).not.toBeInTheDocument();
  fireEvent.change(screen.getByLabelText('IP de gerenciamento'), {
    target: { value: '192.0.2.31' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Salvar' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Ver na lista' }));
  await waitFor(() => expect(back).toHaveBeenCalledOnce());
  expect(continueEnrollment).not.toHaveBeenCalled();
});
