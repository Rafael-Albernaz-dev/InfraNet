import { DevicePageSampler } from './ppp-device-sampler';
import { useCallback, useState } from 'react';
import type { PPPEntry, PPReadRow } from '../../platform/wails/presentation';
import { type PageSample } from './pppoe-sessions-view';
export type Input = {
  profile: string;
  activation: string;
  entries: PPPEntry[];
  active: boolean;
};
export function usePPPPageSamples(p: Input) {
  const [samples, publish] = useState<Record<string, PageSample>>({});
  const receive = useCallback(
    (device: string, sample: PageSample) =>
      publish((old) => ({ ...old, [device]: sample })),
    [],
  );
  const groups = new Map<string, PPPEntry[]>();
  for (const e of p.entries)
    if (e.session) {
      const rows = groups.get(e.device_id) || [];
      rows.push(e);
      groups.set(e.device_id, rows);
    }
  const entries = p.entries.map((e) => withSample(e, samples[e.device_id]));
  const samplers = p.active
    ? Array.from(groups, ([device, entries]) => (
        <DevicePageSampler
          key={device}
          {...p}
          device={device}
          entries={entries}
          publish={receive}
        />
      ))
    : null;
  return { entries, samplers };
}
function withSample(e: PPPEntry, sample: PageSample | undefined) {
  if (
    !sample ||
    sample.error ||
    sample.age >= 6000 ||
    !sample.key.startsWith(e.observation_id + '|') ||
    !e.session
  )
    return e;
  const r = sample.result?.rows.find(
    (row: PPReadRow) => row.row_id === e.session?.row_id,
  );
  return r
    ? {
        ...e,
        session: { ...e.session, download: r.download, upload: r.upload },
      }
    : e;
}
