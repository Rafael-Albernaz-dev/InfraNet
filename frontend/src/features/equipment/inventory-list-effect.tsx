import type { Group } from './groups';
import { NetworkPulse } from './network-pulse-context';
import { useContext, useEffect } from 'react';
import type { Device } from './equipment-api';
import { equipmentAPI as api, equipmentMessage } from './equipment-api';
import { useLatestRef } from '../../shared/hooks/use-latest-ref';
type UseDeviceListEffectProps = {
  inventoryNode: HTMLElement | null;
  profile: string;
  activation: string;
  search: string;
  cursor: number;
  inactive: boolean;
  roleFilter: string;
  blockFilter: string;
  modelFilter: string;
  stateFilter: string;
  setPulseNetwork: (v: string) => void;
  setBlocks: (v: string[]) => void;
  setGroups: (v: Group[]) => void;
  setModels: (v: string[]) => void;
  connected: boolean;
  refresh: number;
  onTotal?: (context: string, total: number) => void;
  generationRef: React.MutableRefObject<number>;
  setListError: (v: string) => void;
  setDevices: (v: Device[]) => void;
  setSelected: React.Dispatch<React.SetStateAction<Device | undefined>>;
  setTotal: (v: number) => void;
  setNext: (v: number) => void;
  setVault: (v: boolean) => void;
  setReady: (v: boolean) => void;
};

async function readList(
  p: UseDeviceListEffectProps,
  current: () => boolean,
  publish: (pulse: string) => void,
) {
  try {
    const r = await api.list(p.profile, p.activation, '', 0, false, {
      all: true,
      role: '',
      block: '',
      block_empty: false,
      model: '',
    });
    if (!current()) return;
    if (r.code !== 'OK' || !r.data)
      return p.setListError(equipmentMessage(r.code));
    p.setListError('');
    const list = r.data.devices || [];
    p.setDevices(list);
    p.setSelected((previous) => selectedInventoryLabels(previous, list));
    p.setPulseNetwork(r.data.pulse_network || '');
    publish(r.data.pulse_network || '');
    p.setBlocks(r.data.blocks || []);
    p.setGroups(r.data.groups || []);
    p.setModels(r.data.models || []);
    const total = r.data.total_devices ?? list.length;
    p.setTotal(total);
    p.onTotal?.([p.profile, p.activation].join('|'), total);
    p.setNext(r.data.next_cursor);
    p.setVault(r.data.vault_available);
    p.setReady(true);
  } catch {
    if (current()) p.setListError(equipmentMessage('SERVICE_UNAVAILABLE'));
  }
}
export function useDeviceListEffect(p: UseDeviceListEffectProps) {
  const { publish, register } = useContext(NetworkPulse);
  useEffect(() => {
    if (p.inventoryNode && p.profile && p.connected) return register();
  }, [p.inventoryNode, p.profile, p.connected, register]);
  const latest = useLatestRef(p);
  const { generationRef } = p;
  useEffect(() => {
    const p = latest.current;
    const doc = p.inventoryNode?.ownerDocument;
    if (!doc || !p.profile || !p.connected) return;
    let stopped = false;
    let pending = false;
    const read = () => {
      if (doc.visibilityState === 'hidden' || pending || stopped) return;
      pending = true;
      const generation = ++generationRef.current;
      const current = () => !stopped && generation === generationRef.current;
      void readList(p, current, publish).finally(() => {
        if (generation === generationRef.current) pending = false;
      });
    };
    const visibility = () => {
      ++generationRef.current;
      if (doc.visibilityState !== 'hidden') {
        pending = false;
        read();
      }
    };
    read();
    const timer = setInterval(read, 5000);
    doc.addEventListener('visibilitychange', visibility);
    return () => {
      stopped = true;
      clearInterval(timer);
      doc.removeEventListener('visibilitychange', visibility);
    };
  }, [
    latest,
    publish,
    generationRef,
    p.inventoryNode,
    p.profile,
    p.activation,
    p.connected,
    p.refresh,
    p.onTotal,
  ]);
}

function selectedInventoryLabels(
  previous: Device | undefined,
  devices: Device[],
) {
  if (!previous) return previous;
  const latest = devices.find(
    (device) =>
      device.id === previous.id && device.revision === previous.revision,
  );
  if (!latest) return previous;
  if (
    previous.name === latest.name &&
    previous.identity === latest.identity &&
    previous.group_id === latest.group_id &&
    previous.group_path === latest.group_path
  )
    return previous;
  return {
    ...previous,
    name: latest.name,
    identity: latest.identity,
    group_id: latest.group_id,
    group_path: latest.group_path,
  };
}
