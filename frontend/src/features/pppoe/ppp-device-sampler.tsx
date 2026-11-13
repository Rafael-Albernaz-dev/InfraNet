import { useCallback, useEffect, useState } from 'react';
import { equipmentAPI, type Device } from '../equipment/equipment-api';
import { SessionPageSampler, type PageSample } from './pppoe-sessions-view';
import type { Input } from './ppp-page-samples';
export function DevicePageSampler(
  p: Input & {
    device: string;
    publish: (device: string, sample: PageSample) => void;
  },
) {
  const [device, setDevice] = useState<Device>();
  useEffect(() => {
    let alive = true;
    void equipmentAPI.get(p.profile, p.activation, p.device).then((r) => {
      if (
        alive &&
        r.code === 'OK' &&
        r.profile_id === p.profile &&
        r.activation_id === p.activation
      )
        setDevice(r.data?.device);
    });
    return () => {
      alive = false;
    };
  }, [p.profile, p.activation, p.device]);
  const { publish: receive, device: deviceID } = p;
  const publish = useCallback(
    (sample: PageSample) => receive(deviceID, sample),
    [receive, deviceID],
  );
  const rows = p.entries.flatMap((e) => (e.session ? [e.session] : []));
  const observation = p.entries[0]?.observation_id || '';
  const key = observation + '|' + rows.map((r) => r.row_id).join(',');
  if (!device) return null;
  return (
    <SessionPageSampler
      profile={p.profile}
      activation={p.activation}
      deviceId={p.device}
      device={device}
      observation={observation}
      rows={rows}
      sampleKey={key}
      publish={publish}
      onDetail={() => {}}
    />
  );
}
