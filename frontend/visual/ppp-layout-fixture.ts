import { radioFixtureOperations } from './layout-radio-operations';
export function pppFixtureOperations(result: (data: object) => unknown) {
  return {
    ListPPP: async () =>
      result({
        entries: [],
        total: 0,
        catalog_dates: { 'CCR-CENTRO': new Date().toISOString() },
      }),
    GetPPPSummary: async () =>
      result({
        total: 0,
        summary: {
          online: 0,
          offline: 0,
          drops_today: 0,
          plans: 0,
          servers: 0,
          invalid_servers: 0,
          falling_servers: 0,
          unlinked_servers: 0,
        },
      }),
  };
}

export function radioAndPPPFixtureOperations(
  result: (data: object) => unknown,
  reading: (id: string) => unknown,
) {
  return {
    ...radioFixtureOperations(result, reading),
    ...pppFixtureOperations(result),
  };
}
