import type { EquipmentResult } from '../../platform/wails/presentation';

export type InventoryFilters = {
  all?: boolean;
  state?: string;
  role: string;
  block: string;
  block_empty: boolean;
  model: string;
};
export type InventoryBulk = {
  items: { id: string; revision: number }[];
  block?: string;
  group_id?: string;
  credential_id?: string;
};
type FilterInput = InventoryFilters & {
  search: string;
  cursor: number;
  inactive: boolean;
};
export function createInventoryBindings(
  list: (
    profile: string,
    activation: string,
    search: string,
    cursor: number,
    inactive: boolean,
  ) => Promise<EquipmentResult>,
  listFiltered: (
    profile: string,
    activation: string,
    input: FilterInput,
  ) => Promise<EquipmentResult>,
  bulk: (
    profile: string,
    activation: string,
    operation: string,
    input: InventoryBulk,
  ) => Promise<EquipmentResult>,
) {
  return {
    list: (
      profile: string,
      activation: string,
      search: string,
      cursor: number,
      inactive: boolean,
      filters?: InventoryFilters,
    ) => {
      if (!filters) return list(profile, activation, search, cursor, inactive);
      const read = (next: number) =>
        listFiltered(profile, activation, {
          search,
          cursor: next,
          inactive,
          ...filters,
        });
      return filters.all ? completeInventory(read, cursor) : read(cursor);
    },
    bulk,
  };
}

async function completeInventory(
  read: (cursor: number) => Promise<EquipmentResult>,
  cursor: number,
) {
  const first = await read(cursor);
  if (first.code !== 'OK' || !first.data) return first;
  const devices = [...(first.data.devices || [])];
  const seen = new Set<number>();
  let next = first.data.next_cursor;
  while (next) {
    if (seen.has(next)) throw Error('CURSOR_INVALID');
    seen.add(next);
    const r = await read(next);
    if (
      r.code !== 'OK' ||
      !r.data ||
      r.profile_id !== first.profile_id ||
      r.activation_id !== first.activation_id ||
      r.data.total_devices !== first.data.total_devices
    )
      throw Error('INVENTORY_CHANGED');
    devices.push(...(r.data.devices || []));
    next = r.data.next_cursor;
  }
  if (new Set(devices.map((d) => d.id)).size !== devices.length)
    throw Error('INVENTORY_CHANGED');
  return { ...first, data: { ...first.data, devices, next_cursor: 0 } };
}
