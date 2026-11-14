import { ConfigOrigins } from '../src/features/equipment/radio-contract';
const times = [
  '2026-11-12T03:10:00Z',
  '2026-11-09T03:10:00Z',
  '2026-11-04T16:42:00Z',
  '2026-11-01T03:10:00Z',
];
const versions = times.map((created_at, i) => ({
  id: 'config-' + i,
  device_id: 'CCR-CENTRO',
  created_at,
  origin: i === 2 ? 'manual' : ConfigOrigins.automatic,
  format: 'rsc',
  size: [49356, 49152, 48025, 48025][i],
  added: [3, 12, 1, -1][i],
  removed: [1, 0, 1, 0][i],
  first: i === 3,
}));
const view = {
  versions,
  format: 'rsc',
  supported: true,
  busy: false,
  status: { at: times[0], code: 'OK', unchanged: false },
};
export const configFixtureOperations = {
  ListConfig: async () => ({ code: 'OK', data: view }),
  SaveConfig: async () => ({
    code: 'OK',
    data: { ...view, status: { ...view.status, unchanged: true } },
  }),
  CompareConfig: async () => ({
    code: 'OK',
    data: {
      ...view,
      change: {
        from: times[1],
        to: times[0],
        added: 3,
        removed: 1,
        truncated: false,
        lines: [
          { kind: 'same', text: '/ip address' },
          { kind: 'remove', text: 'add address=192.0.2.1/24 interface=ether1' },
          {
            kind: 'add',
            text: 'add address=192.0.2.1/24 interface=ether1 comment=uplink',
          },
          { kind: 'same', text: '/ppp profile' },
          {
            kind: 'add',
            text: 'add name="Plano 300M" rate-limit=60M/300M local-address=192.0.2.1 remote-address=pool-cgnat',
          },
          { kind: 'same', text: '/system ntp client' },
          { kind: 'add', text: 'set enabled=yes primary-ntp=192.0.2.123' },
        ],
      },
    },
  }),
  DownloadConfig: async () => ({ code: 'EXPORT_CANCELLED' }),
};
