import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { Equipment } from '../../app/pages/equipment';
import { equipmentAPI } from '../equipment/equipment-api';
import type { Device } from '../equipment/equipment-api';

vi.mock('../../platform/wails/presentation', async (importOriginal) =>
  (await import('../../shared/testing/presentation-mock')).presentationMock(
    {
      access: { status: vi.fn(async () => ({ code: 'OK', columns: [] })) },
    },
    Object.fromEntries(
      Object.entries(
        await importOriginal<Record<string, Record<string, unknown>>>(),
      ).filter(([name]) => !['access', 'equipmentAPI'].includes(name)),
    ),
  ),
);
const devices = Array.from({ length: 500 }, (_, i) => ({
  id: `fixture-${i}`,
  name: `Sintetico ${String(i).padStart(3, '0')}`,
  role: 'other',
  address: `192.0.2.${(i % 250) + 1}`,
  revision: 1,
  active: true,
})) as Device[];
const result = (page: Device[], next = 0) => ({
  code: 'OK',
  profile_id: 'company',
  activation_id: 'active',
  data: {
    devices: page,
    credentials: [],
    groups: [
      {
        id: '11111111-1111-4111-8111-111111111111',
        parent_id: '',
        name: 'Torre sintética',
      },
    ],
    next_cursor: next,
    vault_available: true,
    total_devices: 500,
  },
});
it('selects all 500 filtered synthetic devices across pages and submits their exact revisions in one bulk operation', async () => {
  vi.mocked(equipmentAPI.list).mockResolvedValue(result(devices) as never);
  vi.mocked(equipmentAPI.credentials).mockResolvedValue(result([]) as never);
  vi.mocked(equipmentAPI.bulk).mockResolvedValue(result([]) as never);
  render(<Equipment profile="company" activation="active" connected />);
  await screen.findByText('Sintetico 000');
  expect(screen.getByText('Página 1 de 10')).toBeVisible();
  fireEvent.click(screen.getByRole('checkbox', { name: 'Selecionar todos' }));
  expect(screen.getByText('500 selecionados')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Definir grupo' }));
  const dialog = screen.getByRole('dialog', { name: 'Definir grupo' });
  fireEvent.change(within(dialog).getByLabelText('Grupo'), {
    target: { value: '11111111-1111-4111-8111-111111111111' },
  });
  fireEvent.click(within(dialog).getByRole('button', { name: 'Aplicar' }));
  await waitFor(() => expect(equipmentAPI.bulk).toHaveBeenCalledOnce());
  const input = vi.mocked(equipmentAPI.bulk).mock.calls[0][3];
  expect(input.items).toEqual(
    devices.map(({ id, revision }) => ({ id, revision })),
  );
  expect(input.group_id).toBe('11111111-1111-4111-8111-111111111111');
  expect(equipmentAPI.start).not.toHaveBeenCalled();
});
