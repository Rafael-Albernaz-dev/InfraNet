import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { InventoryTable } from '../inventory/inventory-table';
import type { InventoryState } from '../inventory/inventory-selection-state';
import { Summary } from './equipment-summary-blocks';
import type { Device } from './equipment-api';

it('keeps permission failures unknown in list and summary and orders/filters known metrics numerically', () => {
  const devices = [
    { id: 'blocked', name: 'Sem permissão', active: true, pulse: {} },
    {
      id: 'slow',
      name: 'Lento',
      active: true,
      pulse: { latency_ms: 10, loss_percent: 10 },
    },
    {
      id: 'fast',
      name: 'Rápido',
      active: true,
      pulse: { latency_ms: 2, loss_percent: 0 },
    },
    {
      id: 'enlace-client',
      name: 'CPE Enlace',
      active: true,
      cpe_login: { login: 'synthetic' },
      pulse: { latency_ms: 1, loss_percent: 99 },
    },
    {
      id: 'inactive',
      name: 'Inativo',
      active: false,
      pulse: { latency_ms: 1, loss_percent: 99 },
    },
  ] as Device[];
  const p = {
    devices,
    ready: true,
    selectedRows: {},
    setSelectedRows: vi.fn(),
    openDevice: vi.fn(),
    credentials: [],
    displayNow: Date.now(),
    models: [],
  } as unknown as InventoryState;
  const view = render(<InventoryTable p={p} />);
  fireEvent.click(screen.getByRole('button', { name: 'Colunas' }));
  for (const name of ['Latência', 'Perda'])
    fireEvent.click(screen.getByRole('checkbox', { name }));
  const table = screen.getByRole('table');
  expect(
    within(table).getByText('Sem permissão').closest('tr'),
  ).toHaveTextContent('—');
  expect(
    within(table).getByText('CPE Enlace').closest('tr'),
  ).not.toHaveTextContent('99%');
  fireEvent.click(screen.getByRole('button', { name: 'Latência' }));
  expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Rápido');
  fireEvent.change(screen.getByRole('combobox', { name: 'Perda' }), {
    target: { value: '10' },
  });
  expect(within(table).getAllByRole('row')).toHaveLength(2);
  expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Lento');
  view.unmount();
  render(
    <Summary
      device={
        {
          ...devices[0],
          result: {
            identity: 'synthetic',
            free_memory_bytes: 1,
            total_memory_bytes: 2,
          },
        } as Device
      }
    />,
  );
  expect(
    screen.getByText('Latência / perda').nextElementSibling,
  ).toHaveTextContent('—');
});
