import {
  ListConfig,
  SaveConfig,
  CompareConfig,
  DownloadConfig,
} from '../../../wailsjs/go/presentation/App';
import type { ConfigResult } from '../../features/equipment/config-types';
export const configAPI = {
  list: (p: string, a: string, d: string): Promise<ConfigResult> =>
    ListConfig(p, a, d),
  save: (p: string, a: string, d: string): Promise<ConfigResult> =>
    SaveConfig(p, a, d),
  compare: (
    p: string,
    a: string,
    d: string,
    v: string,
  ): Promise<ConfigResult> => CompareConfig(p, a, d, v),
  download: (
    p: string,
    a: string,
    d: string,
    v: string,
  ): Promise<ConfigResult> => DownloadConfig(p, a, d, v, true),
};
