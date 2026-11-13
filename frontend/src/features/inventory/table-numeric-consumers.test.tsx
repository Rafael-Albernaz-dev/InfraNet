import { EquipmentPorts } from '../equipment/equipment-health-tables';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { InventoryTable } from './inventory-table';
import type { InventoryState } from './inventory-selection-state';
import type { Device } from '../equipment/equipment-api';
import { RadioClients } from '../radio/radio-equipment-tables';
import { reading } from '../../shared/testing/radio-test-fixtures';
import type { ComponentProps } from 'react';
it('orders displayed inventory voltage and uptime by numeric values while preserving text versions', () => {
  const at = new Date().toISOString();
  const devices = [10, 2].map((n) => ({
    id: String(n),
    name: `Equipamento ${n}`,
    active: true,
    role: 'other',
    address: `192.0.2.${n}`,
    radio_summary: {
      observed_at: at,
      voltage: n,
      uptime: `${n}h`,
      cpu: n,
      version: n === 2 ? '6.49.19' : '7.1',
    },
  })) as Device[];
  const p = {
    devices,
    ready: true,
    selectedRows: {},
    setSelectedRows: vi.fn(),
    openDevice: vi.fn(),
    credentials: [],
    displayNow: Date.now(),
    blocks: [],
    models: [],
  } as unknown as InventoryState;
  render(<InventoryTable p={p} />);
  const first = () => within(screen.getByRole('table')).getAllByRole('row')[1];
  fireEvent.click(screen.getByRole('button', { name: 'Colunas' }));
  fireEvent.click(screen.getByRole('checkbox', { name: 'Tensão' }));
  fireEvent.click(screen.getByRole('button', { name: 'Tensão' }));
  expect(first()).toHaveTextContent('Equipamento 2');
  fireEvent.click(screen.getByRole('button', { name: 'Tensão' }));
  expect(first()).toHaveTextContent('Equipamento 10');
  fireEvent.click(screen.getByRole('button', { name: 'Uptime' }));
  expect(first()).toHaveTextContent('Equipamento 2');
  fireEvent.click(screen.getByRole('checkbox', { name: 'Sistema' }));
  fireEvent.click(screen.getByRole('button', { name: 'Sistema' }));
  expect(first()).toHaveTextContent('6.49.19');
});
it('searches AP client names through their rendered link and sorts signal/CCQ numerically', () => {
  const clients = [10, 2].map((n) => ({
    ...reading.clients[0],
    name: `Cliente ${n}`,
    mac: `02:00:00:00:00:${n}`,
    metrics: [
      ...reading.clients[0].metrics.map((m) => ({ ...m, value: n })),
      { ...reading.clients[0].metrics[0], name: 'ccq_tx', value: n, unit: '%' },
    ],
  }));
  const state = {
    reading: { ...reading, clients },
    devices: [],
  } as unknown as ComponentProps<typeof RadioClients>['state'];
  render(<RadioClients state={state} />);
  fireEvent.change(
    screen.getByRole('textbox', { name: 'Buscar em todas as colunas' }),
    { target: { value: 'Cliente 2' } },
  );
  expect(screen.getByRole('cell', { name: 'Cliente 2' })).toBeVisible();
  expect(
    screen.queryByRole('cell', { name: 'Cliente 10' }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Limpar filtros' }));
  fireEvent.click(screen.getByRole('button', { name: 'CCQ' }));
  expect(
    within(screen.getByRole('table')).getAllByRole('row')[1],
  ).toHaveTextContent('Cliente 2');
});

it('orders exact uint64 port counters without losing the last digits', () => {
  const ports = [
    ['Maior', '18446744073709551611'],
    ['Menor', '18446744073709551610'],
  ].map(([name, link_downs]) => ({
    kind: 'port',
    name,
    port: { link_downs },
  })) as unknown as import('../../platform/wails/presentation').PPReadRow[];
  render(<EquipmentPorts ports={ports} />);
  fireEvent.click(screen.getByRole('button', { name: 'Quedas acumuladas' }));
  expect(
    within(screen.getByRole('table')).getAllByRole('row')[1],
  ).toHaveTextContent('Menor');
  fireEvent.click(screen.getByRole('button', { name: 'Quedas acumuladas' }));
  expect(
    within(screen.getByRole('table')).getAllByRole('row')[1],
  ).toHaveTextContent('Maior');
});
