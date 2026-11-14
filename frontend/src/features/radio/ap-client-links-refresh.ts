import { useEffect } from 'react';
import type { RefObject } from 'react';
import { radioAPI } from '../equipment/radio-api';
import type { RadioEquipmentState } from './radio-equipment-state';
import { useLatestRef } from '../../shared/hooks/use-latest-ref';
// Refresh the local projection while background enrollment completes; this
// never starts an SSH read and never replaces a foreground operation's result.
export function useAPClientLinksRefresh(
  state: RadioEquipmentState,
  enabled: boolean,
  root: RefObject<HTMLElement | null>,
) {
  const latest = useLatestRef(state);
  const { profile, activation, device } = state;
  useEffect(() => {
    if (!enabled) return;
    let alive = true;
    let pending = false;
    const read = async () => {
      const current = latest.current;
      if (
        pending ||
        current.radio.busy ||
        root.current?.ownerDocument.visibilityState === 'hidden'
      )
        return;
      pending = true;
      const owner = current.radio.owner();
      try {
        const r = await radioAPI.current(profile, activation, device.id);
        if (
          !alive ||
          latest.current.radio.busy ||
          latest.current.radio.owner() !== owner
        )
          return;
        if (
          r.code !== 'OK' ||
          r.profile_id !== profile ||
          r.activation_id !== activation ||
          !r.data?.reading
        )
          return;
        const reading = r.data.reading;
        latest.current.radio.setView((old) =>
          old ? { ...old, reading } : r.data,
        );
      } catch {
        // The last reading stays visible; the existing explicit refresh reports failures.
      } finally {
        pending = false;
      }
    };
    const timer = setInterval(() => void read(), 5000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [profile, activation, device.id, enabled, latest, root]);
}
