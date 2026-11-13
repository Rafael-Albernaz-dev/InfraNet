import {
  GetTablePreference,
  SetTablePreference,
} from '../../../wailsjs/go/presentation/App';
import type { equipment } from '../../../wailsjs/go/models';
import type { TablePreference } from '../../shared/ui/table-model';
export const tablePreferenceAPI = {
  get: async (p: string, a: string, id: string) => {
    const r = await GetTablePreference(p, a, id);
    if (r.code !== 'OK') throw new Error(r.code);
    return r.data?.table_preference as TablePreference | undefined;
  },
  set: async (p: string, a: string, id: string, state: TablePreference) => {
    const r = await SetTablePreference(
      p,
      a,
      id,
      state as equipment.TablePreference,
    );
    if (r.code !== 'OK') throw new Error(r.code);
  },
};
