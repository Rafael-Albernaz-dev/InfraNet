import { completePPTable } from './stored-table-pages';
import {
  CancelPPPoECollection,
  CancelPPPoEDiagnostic,
  CancelPPPoERead,
  CancelPPPoEReadOperation,
  ComparePPPoE,
  DeletePPPoEReference,
  ExportPPPoE,
  ExportPPPoEOperational,
  GetDeviceMonitoring,
  GetPPPoECurrent,
  GetPPPoEDiagnostic,
  GetPPPoEObservation,
  GetPPPoESession,
  ListPPPoEDiagnostic,
  ListPPPoEReferences,
  ListPPPoESessions,
  PPPoECollectionStatus,
  PPPoEContextCounts,
  PPPoEDiagnosticStatus,
  PPPoEReadPage,
  PPPoEReadStatus,
  SavePPPoEReference,
  StartPPPoECatalog,
  StartPPPoECollection,
  StartPPPoEPing,
  StartPPPoERadius,
  StartPPPoERead,
  StopPPPoETraffic,
} from '../../../wailsjs/go/presentation/App';
import type { equipment } from '../../../wailsjs/go/models';
import type { PPQuery, PPResult } from './pppoe-types';
// Aridade fixada pelo método Go do binding; a tupla rotulada documenta cada posição.
type DiagnosticListArgs = [
  profile: string,
  activation: string,
  device: string,
  scope: string,
  observation: string,
  cursor: string,
  kind?: string,
];
type SessionListArgs = [
  profile: string,
  activation: string,
  device: string,
  observation: string,
  query: PPQuery,
  cursor: string,
];
type CompareArgs = [
  profile: string,
  activation: string,
  device: string,
  base: string,
  target: string,
  query: PPQuery,
  cursor: string,
];
const pdBindings = {
  catalog: StartPPPoECatalog,
  radius: StartPPPoERadius,
  ping: StartPPPoEPing,
  status: PPPoEDiagnosticStatus,
  cancel: CancelPPPoEDiagnostic,
  current: GetPPPoEDiagnostic,
  list: async (...[p, a, d, s, o, c, kind = '']: DiagnosticListArgs) =>
    completePPTable(
      await ListPPPoEDiagnostic(p, a, d, s, o, c, kind),
      (next) => ListPPPoEDiagnostic(p, a, d, s, o, next, kind),
      'diagnostic_rows',
    ),
};
export const pppoeDiagnosticAPI: {
  [K in keyof typeof pdBindings]: (
    ...args: Parameters<(typeof pdBindings)[K]>
  ) => Promise<PPResult>;
} = pdBindings;
const ppBindings = {
  start: StartPPPoECollection,
  status: PPPoECollectionStatus,
  cancel: CancelPPPoECollection,
  current: GetPPPoECurrent,
  observation: GetPPPoEObservation,
  list: (...[p, a, d, o, q, c]: SessionListArgs) =>
    ListPPPoESessions(
      p,
      a,
      d,
      o,
      { ...q, category: '' } as equipment.PPQuery,
      c,
    ),
  detail: GetPPPoESession,
  groups: PPPoEContextCounts,
  save: SavePPPoEReference,
  references: ListPPPoEReferences,
  remove: DeletePPPoEReference,
  compare: async (...[p, a, d, b, t, q, c]: CompareArgs) => {
    const query = {
      category: q.category,
      state: q.state || '',
    } as equipment.PPQuery;
    return completePPTable(
      await ComparePPPoE(p, a, d, b, t, query, c),
      (next) => ComparePPPoE(p, a, d, b, t, query, next),
      'deltas',
    );
  },
};
export const pppoeAPI: {
  [K in keyof typeof ppBindings]: (
    ...args: Parameters<(typeof ppBindings)[K]>
  ) => Promise<PPResult>;
} & {
  export: typeof ExportPPPoE;
  exportOperational: typeof ExportPPPoEOperational;
} = {
  ...ppBindings,
  export: ExportPPPoE,
  exportOperational: ExportPPPoEOperational,
};
const readBindings = {
  start: StartPPPoERead,
  stop: StopPPPoETraffic,
  status: async (...args: Parameters<typeof PPPoEReadStatus>) => {
    const first = await PPPoEReadStatus(...args);
    if (first.data?.job?.state !== 'SUCCEEDED') return first;
    return completePPTable(
      first,
      (next) => PPPoEReadPage(args[0], args[1], args[2], next),
      'read',
    );
  },
  cancel: CancelPPPoERead,
  cancelOperation: CancelPPPoEReadOperation,
  page: async (...args: Parameters<typeof PPPoEReadPage>) =>
    completePPTable(
      await PPPoEReadPage(...args),
      (next) => PPPoEReadPage(args[0], args[1], args[2], next),
      'read',
    ),
};
export const pppoeReadAPI: {
  [K in keyof typeof readBindings]: (
    ...args: Parameters<(typeof readBindings)[K]>
  ) => Promise<PPResult>;
} = readBindings;
export const deviceMonitoringAPI = {
  current: async (
    ...args: Parameters<typeof GetDeviceMonitoring>
  ): Promise<PPResult> =>
    completePPTable(
      await GetDeviceMonitoring(...args),
      (next) => GetDeviceMonitoring(args[0], args[1], args[2], args[3], next),
      'read',
    ),
};
