import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { SessionEnlace } from './enlace-session';
import { radioAPI, pppCatalogAPI } from '../../platform/wails/presentation';
import type { Device } from '../equipment/equipment-api';
const scope = { profile_id: 'synthetic', activation_id: 'active' };
afterEach(() => vi.restoreAllMocks());
it('waits for the operator to choose the AP with its signal before mounting', async () => {
  vi.spyOn(pppCatalogAPI, 'list').mockResolvedValue({
    ...scope,
    code: 'OK',
    data: { entries: [], total: 0 },
  });
  vi.spyOn(radioAPI, 'enlace').mockResolvedValue({
    ...scope,
    code: 'OK',
    data: {
      total: 0,
      more: false,
      next_offset: 0,
      enlace: {
        state: 'two',
        stage: 2,
        interface: 'vlan1004',
        login: 'client',
        address: '100.64.0.1',
        ap_ids: ['north', 'south'],
        choices: [
          { ap_id: 'north', name: 'AP-NORTE', signal: '-61 dBm' },
          { ap_id: 'south', name: 'AP-SUL', signal: '-79 dBm' },
        ],
      },
    },
  });
  const confirm = vi
    .spyOn(radioAPI, 'confirmEnlace')
    .mockImplementation(async () => {
      vi.mocked(radioAPI.enlace).mockResolvedValue({
        ...scope,
        code: 'OK',
        data: {
          total: 0,
          more: false,
          next_offset: 0,
          enlace: {
            state: 'mounting',
            stage: 3,
            interface: 'vlan1004',
            login: 'client',
            address: '100.64.0.1',
            ap_ids: ['north', 'south'],
          },
        },
      });
      return { ...scope, code: 'OK' };
    });
  render(
    <SessionEnlace
      profile="synthetic"
      activation="active"
      device={{ id: 'concentrator', name: 'Concentrador' } as Device}
      observation="observation"
      row="row"
      pppoeUptime="1 h"
      blocked={false}
    />,
  );
  const button = await screen.findByRole('button', { name: 'Confirmar AP' });
  expect(button).toBeDisabled();
  expect(confirm).not.toHaveBeenCalled();
  expect(screen.getByText(/-61 dBm/)).toBeVisible();
  expect(screen.getByText(/-79 dBm/)).toBeVisible();
  expect(
    screen.queryByRole('button', { name: 'Atualizar índice' }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('radio', { name: /AP-SUL/ }));
  fireEvent.click(button);
  await waitFor(() =>
    expect(confirm).toHaveBeenCalledWith('synthetic', 'active', {
      device_id: 'concentrator',
      observation_id: 'observation',
      row_id: 'row',
      confirm_ap: 'south',
    }),
  );
  await screen.findByText(/Montando o enlace deste cliente/);
  expect(
    screen.queryByRole('button', { name: 'Confirmar AP' }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByText('Vínculo manual (casos especiais)').closest('details'),
  ).not.toHaveAttribute('open');
});
