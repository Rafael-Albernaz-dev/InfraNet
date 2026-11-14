import { useEffect, useState } from 'react';
import { radioAPI } from '../equipment/radio-api';
import type { EnlaceView } from '../equipment/radio-api';
import type { RadioLinkPanelProps } from './radio-link-props';
export function useEnlaceSession(p: RadioLinkPanelProps) {
  const [result, setResult] = useState<{
    key: string;
    view?: EnlaceView;
    error?: string;
  }>();
  const [revision, refresh] = useState(0);
  const { profile, activation, device, observation, row } = p;
  const key = [profile, activation, device.id, observation, row, revision].join(
    '|',
  );
  useEffect(() => {
    let alive = true;
    let timer: ReturnType<typeof setTimeout>;
    const load = async () => {
      try {
        const r = await radioAPI.enlace(profile, activation, {
          device_id: device.id,
          observation_id: observation,
          row_id: row,
        });
        if (!alive) return;
        if (
          r.code !== 'OK' ||
          r.profile_id !== profile ||
          r.activation_id !== activation
        )
          setResult({ key, error: r.code });
        else setResult({ key, view: r.data?.enlace });
      } catch {
        if (alive) setResult({ key, error: 'SERVICE_UNAVAILABLE' });
      }
      if (alive) timer = setTimeout(() => void load(), 2000);
    };
    void load();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [profile, activation, device.id, observation, row, key]);
  async function confirm(ap: string) {
    try {
      const r = await radioAPI.confirmEnlace(profile, activation, {
        device_id: device.id,
        observation_id: observation,
        row_id: row,
        confirm_ap: ap,
      });
      if (
        r.code === 'OK' &&
        r.profile_id === profile &&
        r.activation_id === activation
      )
        refresh((n) => n + 1);
      else setResult({ key, error: r.code });
    } catch {
      setResult({ key, error: 'SERVICE_UNAVAILABLE' });
    }
  }

  return {
    view: result?.key === key ? result.view : undefined,
    error: result?.key === key ? result.error : undefined,
    confirm,
    refresh: () => refresh((n) => n + 1),
  };
}
