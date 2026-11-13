import type { Device } from '../equipment/equipment-api';
import type { InventoryState } from './inventory-selection-state';
import {
  columns,
  defaultColumns,
  inventorySortValue,
} from './inventory-columns';
import { inventoryFilters } from './inventory-table-filters';
import { Checkbox } from '../../shared/ui/Checkbox';
import { DataTable } from '../../shared/ui/DataTable';

export function InventoryTable({ p }: { p: InventoryState }) {
  const definitions = columns;
  const setOne = (d: Device, checked: boolean) =>
    p.setSelectedRows((old) => {
      const next = { ...old };
      if (checked) next[d.id] = d;
      else delete next[d.id];
      return next;
    });
  return (
    <div className="inventory-standard">
      <DataTable
        tableId="equipment"
        emptyMessage={emptyInventoryMessage}
        manual
        ready={p.ready}
        defaultColumns={defaultColumns.slice(1)}
        filters={inventoryFilters(p)}
        selection={inventorySelection(p)}
        aria-label="Lista de equipamentos"
      >
        <thead>
          <tr>
            <th className="inventory-check"></th>
            <th data-column="equipment">Equipamento</th>
            {definitions.map((column) => (
              <th key={column.id} data-column={column.id}>
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {p.devices.map((d) => (
            <tr
              key={d.id}
              data-device-id={d.id}
              data-row-id={d.id}
              data-active={d.active}
              data-role={d.role}
              data-block={d.block}
              data-model={d.result?.board || d.radio_summary?.model || ''}
              className="click"
              tabIndex={0}
              onClick={() => p.openDevice(d)}
              onKeyDown={(e) => {
                if (e.target !== e.currentTarget) return;
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  p.openDevice(d);
                }
              }}
            >
              <td className="inventory-check">
                <Checkbox
                  aria-label={`Selecionar ${d.name}`}
                  checked={Boolean(p.selectedRows[d.id])}
                  onChange={(e) => setOne(d, e.target.checked)}
                />
              </td>
              <td>
                {d.name}
                {!d.active && <span className="muted"> (inativo)</span>}
              </td>
              {definitions.map((column) => (
                <td
                  key={column.id}
                  data-value={inventorySortValue(column.id, {
                    device: d,
                    now: p.displayNow,
                    credentials: p.credentials,
                  })}
                >
                  {column.cell({
                    device: d,
                    now: p.displayNow,
                    credentials: p.credentials,
                  })}
                </td>
              ))}
            </tr>
          ))}
          {p.ready && !p.listError && !p.devices.length && (
            <tr>
              <td colSpan={definitions.length + 2}>
                Nenhum equipamento cadastrado
              </td>
            </tr>
          )}
        </tbody>
      </DataTable>
    </div>
  );
}

function emptyInventoryMessage(
  state: import('../../shared/ui/table-model').TablePreference,
  hasRows: boolean,
) {
  if (state.search || Object.values(state.filters).some(Boolean))
    return 'Nenhum equipamento encontrado';
  return hasRows ? 'Nenhum equipamento ativo' : 'Nenhum equipamento cadastrado';
}

function inventorySelection(p: InventoryState) {
  return {
    ids: Object.keys(p.selectedRows),
    change: (ids: string[]) =>
      p.setSelectedRows(
        Object.fromEntries(
          p.devices.filter((d) => ids.includes(d.id)).map((d) => [d.id, d]),
        ),
      ),
  };
}
