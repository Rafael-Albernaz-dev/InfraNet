import { sessionLinkFixture } from './session-link-fixture';
export function enlaceFixtureOperations(result: (data: object) => unknown) {
  const params = new URLSearchParams(location.search);
  const phase = params.get('enlace') || 'ready';
  const binding = {
    concentrator_id: 'CCR-CENTRO',
    interface: 'vlan1004',
    concentrator_name: 'CCR-CENTRO',
    service: 'pppoe-0001',
    ap_ids: ['AP-NORTE', 'AP-SUL'],
  };
  return {
    GetInterfaceAPs: async () =>
      result({ interface_aps: phase === 'unlinked' ? [] : [binding] }),
    SetInterfaceAPs: async () => result({ interface_aps: [binding] }),
    ConfirmEnlaceAP: async () => result({}),
    GetEnlace: async () =>
      result({
        enlace: {
          state: phase,
          stage: 3,
          interface: 'vlan1004',
          login: 'cliente-0001',
          address: '192.0.2.201',
          ap_ids: binding.ap_ids,
          link:
            phase === 'ready'
              ? sessionLinkFixture(Date.parse('2026-11-04T12:00:00Z'))
              : undefined,
          choices:
            phase === 'two'
              ? [
                  { ap_id: 'AP-NORTE', name: 'AP-NORTE', signal: '-61 dBm' },
                  { ap_id: 'AP-SUL', name: 'AP-SUL', signal: '-79 dBm' },
                ]
              : undefined,
        },
      }),
  };
}
