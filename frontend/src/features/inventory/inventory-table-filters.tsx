import { groupTreeOptions, groupContains } from '../equipment/groups';
import type { InventoryState } from './inventory-selection-state';
import type { TableFilter } from '../../shared/ui/table-model';
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
      matches: (r, v) => r.props['data-pulse'] === (v === 'unknown' ? '—' : v),
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
      id: 'group',
      label: 'Grupo',
      options: [
        { value: 'none', label: 'Sem grupo' },
        ...groupTreeOptions(p.groups || []),
      ],
      matches: (r, v) =>
        v === 'none'
          ? !r.props['data-group']
          : groupContains(
              p.groups || [],
              v,
              String(r.props['data-group'] || ''),
            ),
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
