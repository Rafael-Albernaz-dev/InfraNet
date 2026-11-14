import { enlaceFixtureOperations } from './enlace-layout-fixture';
import { sessionLinkFixture } from './session-link-fixture';
// Deterministic replies for the same approved, synthetic snapshot used by GetRadio.
export function radioFixtureOperations(
  result: (data: object) => unknown,
  reading: (id: string) => unknown,
) {
  const jobs = new Map<string, { device: string; linked: boolean }>();
  return {
    ...enlaceFixtureOperations(result),
    StartRadio: async (
      ...args: [string, string, { device_id: string; row_id?: string }]
    ) => {
      const input = args[2];
      const id = `fixture-radio-${input.device_id}`;
      jobs.set(id, { device: input.device_id, linked: Boolean(input.row_id) });
      return result({ job: { id, state: 'RUNNING', code: 'OK' } });
    },
    RadioStatus: async (...args: [string, string, string]) => {
      const job = jobs.get(args[2]);
      return result({
        job: { id: args[2], state: 'SUCCEEDED', code: 'OK' },
        reading: reading(job?.linked ? 'AP-NORTE' : job?.device || ''),
        cpe_reading: job?.linked ? reading('CPE-0001') : undefined,
        link:
          job?.linked || job?.device === 'CPE-0001'
            ? sessionLinkFixture(Date.parse('2026-11-04T12:00:00Z'))
            : undefined,
      });
    },
    CancelRadioOperation: async () => result({}),
  };
}
