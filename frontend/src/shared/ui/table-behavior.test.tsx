import {
  render,
  screen,
  fireEvent,
  within,
  waitFor,
} from '@testing-library/react';
import { it, expect, vi } from 'vitest';
import { DataTable } from './DataTable';
import { TablePreferences } from './table-preferences';
import type { TablePreference } from './table-model';
const rows = Array.from({ length: 500 }, (_, i) => ({
  id: `row-${i}`,
  name: `Equipamento ${i}`,
  model: i === 450 ? 'RB oculto' : 'Outro',
  cpu: i === 20 ? '—' : i,
}));
function Table({ data = rows }: { data?: typeof rows }) {
  return (
    <DataTable tableId="equipment" manual defaultColumns={['cpu']}>
      <thead>
        <tr>
          <th data-column="name">Equipamento</th>
          <th data-column="model">Modelo</th>
          <th data-column="cpu">CPU</th>
        </tr>
      </thead>
      <tbody>
        {data.map((r) => (
          <tr key={r.id} data-row-id={r.id}>
            <td>{r.name}</td>
            <td>{r.model}</td>
            <td>{r.cpu}</td>
          </tr>
        ))}
      </tbody>
    </DataTable>
  );
}
it('searches hidden columns over all 500 rows before paging and restores filters', () => {
  render(<Table />);
  expect(screen.getByText('Página 1 de 10')).toBeVisible();
  expect(
    screen.queryByRole('columnheader', { name: /Modelo/ }),
  ).not.toBeInTheDocument();
  fireEvent.change(
    screen.getByRole('textbox', { name: 'Buscar em todas as colunas' }),
    { target: { value: 'RB' } },
  );
  expect(screen.getByText('Equipamento 450')).toBeVisible();
  expect(screen.getByText('Página 1 de 1')).toBeVisible();
  expect(screen.getByText('1 linha')).toBeVisible();
  fireEvent.click(screen.getByRole('button', { name: 'Limpar filtros' }));
  expect(screen.getByText('Página 1 de 10')).toBeVisible();
});
it('sorts numerically, keeps missing values last in both directions and returns to manual order across refresh', () => {
  const view = render(<Table />);
  const table = screen.getByRole('table');
  const first = () => within(table).getAllByRole('row')[1];
  fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
  expect(first()).toHaveTextContent('Equipamento 0');
  fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
  expect(first()).toHaveTextContent('Equipamento 499');
  fireEvent.click(screen.getByRole('button', { name: 'Próxima' }));
  expect(screen.getByText('Página 2 de 10')).toBeVisible();
  view.rerender(<Table data={rows.map((r) => ({ ...r }))} />);
  expect(screen.getByText('Página 2 de 10')).toBeVisible();
  fireEvent.click(
    screen.getByRole('button', { name: 'Voltar à ordem manual' }),
  );
  expect(first()).toHaveTextContent('Equipamento 0');
  fireEvent.click(
    within(first()).getByRole('button', { name: 'Mover para o fim' }),
  );
  expect(first()).toHaveTextContent('Equipamento 1');
  fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
  fireEvent.click(
    screen.getByRole('button', { name: 'Voltar à ordem manual' }),
  );
  expect(first()).toHaveTextContent('Equipamento 1');
});
it('keeps the table usable and reports a failed debounced save', async () => {
  const save = vi.fn<(id: string, p: TablePreference) => Promise<void>>(
    async () => {
      throw new Error('disk');
    },
  );
  const load = vi.fn(async () => undefined);
  render(
    <TablePreferences.Provider value={{ scope: 'company', load, save }}>
      <Table data={rows.slice(0, 2)} />
    </TablePreferences.Provider>,
  );
  await waitFor(() => expect(load).toHaveBeenCalledWith('equipment'));
  fireEvent.change(
    screen.getByRole('textbox', { name: 'Buscar em todas as colunas' }),
    { target: { value: 'Equipamento 1' } },
  );
  await screen.findByText('Não foi possível salvar a arrumação');
  expect(screen.getByText('Equipamento 1')).toBeVisible();
  expect(save.mock.calls.at(-1)?.[0]).toBe('equipment');
});
it('loads nullable stored defaults and saves the changed arrangement without leaking across company scopes', async () => {
  const p = {
    columns: ['cpu'],
    filters: null,
    manual: null,
    sort: null,
    search: '450',
    page_size: 25,
  } as unknown as TablePreference;
  const load = vi.fn(async () => p),
    save = vi.fn<(id: string, p: TablePreference) => Promise<void>>(
      async () => {},
    );
  render(
    <TablePreferences.Provider value={{ scope: 'company-X', load, save }}>
      <Table />
    </TablePreferences.Provider>,
  );
  await screen.findByDisplayValue('450');
  expect(screen.getByText('Equipamento 450')).toBeVisible();
  fireEvent.change(screen.getByLabelText('Linhas por página'), {
    target: { value: '100' },
  });
  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save.mock.calls.at(-1)?.[1]).toMatchObject({
    search: '450',
    page_size: 100,
  });
});

it('preserves the manual order on renewal while removing deleted ids and appending a new row', () => {
  const initial = rows.slice(0, 3);
  const view = render(<Table data={initial} />);
  const table = screen.getByRole('table');
  fireEvent.click(
    within(within(table).getAllByRole('row')[1]).getByRole('button', {
      name: 'Mover para o fim',
    }),
  );
  const next = [
    initial[0],
    initial[2],
    { id: 'new', name: 'Equipamento novo', model: 'Novo', cpu: 1 },
  ];
  view.rerender(<Table data={next} />);
  const names = within(table)
    .getAllByRole('row')
    .slice(1)
    .map((row) => within(row).getAllByRole('cell')[1].textContent);
  expect(names).toEqual(['Equipamento 2', 'Equipamento 0', 'Equipamento novo']);
});
