import {
  ListPPP,
  GetPPPSummary,
  ExportPPP,
} from '../../../wailsjs/go/presentation/App';
import type { equipment } from '../../../wailsjs/go/models';
import type { PPResult, PPRow } from './pppoe-types';
export type PPPScreen =
  'sessions' | 'servers' | 'secrets' | 'profiles' | 'l2tp';
export type PPPQuery = {
  screen: PPPScreen;
  search: string;
  filters: Record<string, string>;
  sort: string;
  desc: boolean;
  page_size: number;
};
export type PPPEntry = {
  ap_devices?: Record<string, string>;
  at?: string;
  aps?: string[];
  peak?: number;
  drop?: number;
  id: string;
  kind: string;
  name: string;
  device_id: string;
  concentrator: string;
  observation_id?: string;
  client?: string;
  service?: string;
  profile?: string;
  disabled?: boolean;
  invalid?: boolean;
  interface?: string;
  vlan?: number;
  mtu?: string;
  mru?: string;
  authentication?: string;
  rate?: string;
  local_address?: string;
  remote_address?: string;
  only_one?: string;
  caller_id?: string;
  last_logged_out?: string;
  last_caller_id?: string;
  last_disconnect_reason?: string;
  state?: string;
  online: number;
  registrations: number;
  session?: PPRow;
};
export type PPPSummary = {
  online: number;
  offline: number;
  drops_today: number;
  plans: number;
  servers: number;
  invalid_servers: number;
  falling_servers: number;
  unlinked_servers: number;
};
export const pppCatalogAPI = {
  export: (p: string, a: string, format: 'csv' | 'json'): Promise<string> =>
    ExportPPP(p, a, format),
  list: (
    p: string,
    a: string,
    device: string,
    query: PPPQuery,
    cursor: string,
  ): Promise<PPResult> =>
    ListPPP(p, a, device, query as equipment.PPPQuery, cursor),
  summary: (p: string, a: string, device: string): Promise<PPResult> =>
    GetPPPSummary(p, a, device),
};
