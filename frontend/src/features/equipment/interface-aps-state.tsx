import {
  pppCatalogAPI,
  type PPPEntry,
} from '../../platform/wails/presentation';
import { RadioRoles } from './radio-contract';
import { useEffect, useState } from 'react';
import { equipmentAPI, equipmentMessage } from './equipment-api';
import type { Credential, Device } from './equipment-api';
import { radioAPI } from './radio-api';
export type InterfaceAPInput = {
  profile: string;
  activation: string;
  concentrator: string;
  name: string;
  iface: string;
};
export function useInterfaceAPs(p: InterfaceAPInput) {
  const [loaded, setData] = useState<{
    key: string;
    devices: Device[];
    credentials: Credential[];
    defaults: Record<string, string>;
    selected: string[];
    interfaces: PPPEntry[];
  }>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const { profile, activation, concentrator, iface } = p;
  const key = [profile, activation, concentrator, iface].join('|');
  const data = loaded?.key === key ? loaded : undefined;
  useEffect(() => {
    let alive = true;
    void Promise.all([
      equipmentAPI.list(profile, activation, '', 0, false, {
        all: true,
        role: RadioRoles.ap,
        block: '',
        block_empty: false,
        model: '',
      }),
      readInterfaceCredentials(profile, activation),
      radioAPI.interfaceAPs(profile, activation),
      listLinkableInterfaces(profile, activation, concentrator),
    ])
      .then(([inventory, credentials, links, interfaces]) => {
        if (!alive) return;
        checkInterfaceScope(
          [inventory, credentials, links],
          profile,
          activation,
        );
        setData({
          key,
          interfaces,
          devices: inventory.data?.devices || [],
          credentials: credentials.data?.credentials || [],
          defaults: credentials.data?.credential_defaults || {},
          selected:
            links.data?.interface_aps?.find(
              (b) =>
                b.concentrator_id === concentrator && b.interface === iface,
            )?.ap_ids || [],
        });
      })
      .catch(() => {
        if (alive) setError('Falhou a leitura dos APs cadastrados.');
      });
    return () => {
      alive = false;
    };
  }, [profile, activation, concentrator, iface, key]);
  const toggle = (id: string) =>
    setData(
      (old) =>
        old && {
          ...old,
          selected: old.selected.includes(id)
            ? old.selected.filter((v) => v !== id)
            : [...old.selected, id],
        },
    );
  async function save(extra?: string) {
    if (!data || busy) return false;
    setBusy(true);
    setError('');
    try {
      const selected = extra ? [...data.selected, extra] : data.selected;
      const r = await radioAPI.setInterfaceAPs(profile, activation, {
        device_id: concentrator,
        interface: iface,
        ap_ids: selected,
      });
      if (
        r.code !== 'OK' ||
        r.profile_id !== profile ||
        r.activation_id !== activation
      ) {
        setError(equipmentMessage(r.code));
        return false;
      }
      return true;
    } catch {
      setError('Não foi possível gravar a ligação. Reabra para conferir.');
      return false;
    } finally {
      setBusy(false);
    }
  }
  return { data, error, busy, toggle, save };
}

async function listLinkableInterfaces(
  profile: string,
  activation: string,
  device: string,
) {
  const entries: PPPEntry[] = [];
  let cursor = '';
  const seen = new Set<string>();
  do {
    if (seen.has(cursor)) throw Error('CURSOR_INVALID');
    seen.add(cursor);
    const r = await pppCatalogAPI.list(
      profile,
      activation,
      device,
      {
        screen: 'servers',
        search: '',
        filters: {},
        sort: '',
        desc: false,
        page_size: 200,
      },
      cursor,
    );
    if (
      r.code !== 'OK' ||
      r.profile_id !== profile ||
      r.activation_id !== activation
    )
      throw Error(r.code);
    entries.push(
      ...(r.data?.entries || []).filter(
        (e) => e.invalid !== true && e.interface,
      ),
    );
    cursor = r.data?.next_cursor || '';
  } while (cursor);
  return entries;
}

function checkInterfaceScope(
  results: { code: string; profile_id: string; activation_id: string }[],
  profile: string,
  activation: string,
) {
  for (const r of results) {
    if (
      r.code !== 'OK' ||
      r.profile_id !== profile ||
      r.activation_id !== activation
    )
      throw Error(r.code);
  }
}

async function readInterfaceCredentials(profile: string, activation: string) {
  const first = await equipmentAPI.credentials(profile, activation, 0);
  if (first.code !== 'OK' || !first.data) return first;
  const credentials = [...(first.data.credentials || [])];
  let cursor = first.data.next_cursor;
  const seen = new Set<number>();
  while (cursor) {
    if (seen.has(cursor)) throw Error('CURSOR_INVALID');
    seen.add(cursor);
    const r = await equipmentAPI.credentials(profile, activation, cursor);
    checkInterfaceScope([r], profile, activation);
    credentials.push(...(r.data?.credentials || []));
    cursor = r.data?.next_cursor || 0;
  }
  return { ...first, data: { ...first.data, credentials, next_cursor: 0 } };
}
