import type { InventoryState } from './inventory-selection-state';
import type { TableFilter } from '../../shared/ui/table-model';
import { rowCells, cellValue } from '../../shared/ui/table-model';
import { roleGroups } from '../equipment/equipment-role-options';
export function inventoryFilters(p: InventoryState): TableFilter[] {
  const options = (values: string[]) =>
    values.map((value) => ({ value, label: value }));
  return [
    {
      id: 'status',
      label: 'Estado',
      options: [
        { value: 'ONLINE', label: 'ONLINE' },
        { value: 'OFFLINE', label: 'OFFLINE' },
        { value: 'unknown', label: 'Sem estado' },
      ],
      matches: (r, v) =>
        String(cellValue(rowCells(r)[2])) === (v === 'unknown' ? '—' : v),
    },
    {
      id: 'role',
      label: 'Papel',
      options: roleGroups.flatMap(([, roles]) =>
        roles.map(([value, label]) => ({ value, label })),
      ),
      matches: (r, v) => r.props['data-role'] === v,
    },
    {
      id: 'block',
      label: 'Bloco',
      options: [
        { value: 'none', label: 'Sem bloco' },
        ...p.blocks.map((v) => ({ value: `b:${v}`, label: v })),
      ],
      matches: (r, v) =>
        r.props['data-block'] === (v === 'none' ? '' : v.slice(2)),
    },
    {
      id: 'model',
      label: 'Modelo',
      options: options(p.models),
      matches: (r, v) => r.props['data-model'] === v,
    },
    {
      id: 'inactive',
      label: 'Mostrar inativos',
      checkbox: true,
      always: true,
      options: [{ value: 'all', label: 'Mostrar inativos' }],
      matches: (r, v) => v === 'all' || r.props['data-active'] === true,
    },
  ];
}
