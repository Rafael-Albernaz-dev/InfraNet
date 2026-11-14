import type { Device } from '../../features/equipment/equipment-api';
import type { PPResult, PPRow } from '../../platform/wails/presentation';
import { RadioTransports } from '../../features/equipment/radio-contract';
import type { Result } from '../../platform/wails/presentation';
export const sessionAccount = (company = 'company'): Result => ({
  code: 'OK',
  request_id: 'fixture',
  instance_id: 'one',
  activation_id: company === 'company-two' ? 'activation-two' : 'activation',
  profile_id: company,
  heartbeat: 0,
  setup: false,
  authenticated: true,
  name: 'fixture',
  theme: 'light',
  recovery: '',
  companies: [
    { id: 'company', name: 'Empresa um' },
    { id: 'company-two', name: 'Empresa dois' },
  ],
});

export const sessionDevice = {
  id: 'ccr-fixture',
  name: 'CCR fixture',
  role: 'concentrator',
  address: '192.0.2.1',
  port: 22,
  transport: RadioTransports.ssh,
  revision: 1,
  active: true,
  fingerprint: 'pinned',
} as Device;
export const sessionRows: PPRow[] = ['primeiro', 'segundo'].map((login, i) => ({
  row_id: `row-${i}`,
  remote_id: `*${i}`,
  session_id: `${i}`,
  login,
  address: `192.0.2.${i + 20}`,
  caller_id: '02:00:00:00:00:01',
  uptime: { known: true, min: 1e9, max: 1e9 },
  duplicates: 0,
}));
export const sessionPP = (data: PPResult['data'] = { total: 0 }): PPResult => ({
  code: 'OK',
  profile_id: 'company',
  activation_id: 'activation',
  data,
});
export const sessionEquipment = {
  code: 'OK',
  profile_id: 'company',
  activation_id: 'activation',
  data: {
    devices: [sessionDevice],
    device: sessionDevice,
    credentials: [],
    next_cursor: 0,
    total_devices: 1,
    vault_available: true,
  },
};

export const sessionEnlace = {
  code: 'OK',
  profile_id: 'company',
  activation_id: 'activation',
  data: {
    total: 0,
    more: false,
    next_offset: 0,
    enlace: {
      state: 'unlinked',
      stage: 0,
      interface: 'vlan1004',
      login: 'primeiro',
      address: '192.0.2.1',
      ap_ids: [],
    },
  },
};
