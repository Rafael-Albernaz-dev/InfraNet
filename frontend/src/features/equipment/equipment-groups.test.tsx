import { fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { EquipmentGroups } from './equipment-groups';
import type { GroupsProps } from './equipment-groups-state';
import { equipmentAPI } from './equipment-api';
import { RadioTransports, RadioRoles } from './radio-contract';

afterEach(() => vi.restoreAllMocks());
it('counts descendants and refuses deletion for occupied groups and parents with subgroups', () => {
  const remove = vi.spyOn(equipmentAPI, 'deleteGroup');
  const state = {
    groups: [
      { id: 'region', parent_id: '', name: 'Região' },
      { id: 'tower', parent_id: 'region', name: 'Torre' },
      { id: 'empty-parent', parent_id: '', name: 'Região vazia' },
      { id: 'empty-child', parent_id: 'empty-parent', name: 'Subgrupo vazio' },
    ],
    devices: [
      {
        id: 'device',
        name: 'AP lido',
        address: '192.0.2.1',
        role: RadioRoles.ap,
        group_id: 'tower',
        group_path: 'Região › Torre',
        active: true,
        revision: 1,
        transport: RadioTransports.ssh,
      },
    ],
    setGroups: vi.fn(),
    reload: vi.fn(),
    openDevice: vi.fn(),
  } as unknown as GroupsProps['s'];
  render(
    <EquipmentGroups
      s={state}
      profile="fixture"
      activation="active"
      locked={false}
    />,
  );
  const tree = screen.getByRole('list', { name: 'Grupos' });
  const region = within(tree).getByRole('button', { name: 'Região1' });
  fireEvent.click(region);
  expect(screen.getByRole('cell', { name: 'AP lido' })).toBeVisible();
  const exclusion = screen.getByRole('button', { name: 'Excluir' });
  expect(exclusion).toBeDisabled();
  expect(exclusion).toHaveAttribute(
    'title',
    'Só grupo vazio, sem equipamentos nem subgrupos, pode ser excluído',
  );
  fireEvent.click(exclusion);
  fireEvent.click(within(tree).getByRole('button', { name: 'Torre1' }));
  expect(exclusion).toBeDisabled();
  fireEvent.click(exclusion);
  fireEvent.click(within(tree).getByRole('button', { name: 'Região vazia0' }));
  expect(exclusion).toBeDisabled();
  fireEvent.click(exclusion);
  expect(remove).not.toHaveBeenCalled();
  fireEvent.click(
    within(tree).getByRole('button', { name: 'Subgrupo vazio0' }),
  );
  expect(exclusion).toBeEnabled();
});
