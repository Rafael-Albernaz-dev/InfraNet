import {
  ChooseInventoryImport,
  InventoryImportStatus,
  CancelInventoryImport,
  SaveDiscardedImport,
  ConfirmInventoryImport,
  ReviewInventoryImport,
  SaveInventoryTemplate,
} from '../../../wailsjs/go/presentation/App';
import type {
  EquipmentResult,
  ImportRow,
} from '../../features/inventory/inventory-import-types';
export type {
  EquipmentResult,
  ImportRow,
  ImportItem,
} from '../../features/inventory/inventory-import-types';
export const inventoryImportAPI = {
  status: (p: string, a: string) =>
    InventoryImportStatus(p, a) as Promise<EquipmentResult>,
  cancel: (p: string, a: string) =>
    CancelInventoryImport(p, a) as Promise<EquipmentResult>,
  choose: (p: string, a: string, b: string, c: string) =>
    ChooseInventoryImport(p, a, b, c) as Promise<
      EquipmentResult & { filename?: string }
    >,
  review: (p: string, a: string, rows: ImportRow[]) =>
    ReviewInventoryImport(
      p,
      a,
      rows.map((row) => ({ ...row, group: row.group || '' })),
    ) as Promise<EquipmentResult>,
  confirm: (p: string, a: string, op: string, rows: ImportRow[]) =>
    ConfirmInventoryImport(
      p,
      a,
      op,
      rows.map((row) => ({ ...row, group: row.group || '' })),
    ) as Promise<EquipmentResult>,
  template: SaveInventoryTemplate,
  discarded: SaveDiscardedImport,
};
