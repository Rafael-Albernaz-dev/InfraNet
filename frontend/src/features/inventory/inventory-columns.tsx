import './inventory-pulse.css';
import type { Device, Credential } from '../equipment/equipment-api';
import { equipmentType } from '../equipment/equipment-layout';
import {
  latestSavedReading,
  liveReading,
} from '../equipment/equipment-reading-format';
import {
  formatDate,
  formatDuration,
  formatMetric,
} from '../equipment/network-format';

type Context = { device: Device; now: number; credentials: Credential[] };
type Column = {
  id: string;
  label: string;
  cell: (c: Context) => React.ReactNode;
};
const metric = (v: number | undefined, unit: string) =>
  v === undefined ? '—' : formatMetric(v, unit);
const latest = (d: Device) => {
  const saved = latestSavedReading({ ...d, live: undefined });
  return {
    saved,
    radio:
      saved.at && saved.at === d.radio_summary?.observed_at
        ? d.radio_summary
        : undefined,
    monitoring:
      saved.at && saved.at === d.monitoring_at ? d.monitoring : undefined,
    result:
      saved.at && saved.at === d.result?.observed_at ? d.result : undefined,
  };
};
function memory(d: Device) {
  const { radio, monitoring, result } = latest(d);
  if (radio?.memory !== undefined) return metric(radio.memory, '%');
  const free = monitoring?.free_memory ?? result?.free_memory_bytes;
  const total = monitoring?.total_memory ?? result?.total_memory_bytes;
  return total && free !== undefined && free <= total
    ? metric(((total - free) / total) * 100, '%')
    : '—';
}
export const columns: Column[] = [
  { id: 'status', label: 'Estado', cell: pulseCell },
  {
    id: 'role',
    label: 'Papel',
    cell: ({ device }) => equipmentType(device.role),
  },
  { id: 'block', label: 'Bloco', cell: ({ device }) => device.block || '—' },
  {
    id: 'model',
    label: 'Modelo',
    cell: ({ device }) =>
      device.result?.board || device.radio_summary?.model || '—',
  },
  { id: 'address', label: 'Endereço', cell: ({ device }) => device.address },
  {
    id: 'online',
    label: 'Clientes online',
    cell: ({ device }) =>
      liveReading(device)?.clients ??
      device.pppoe?.online ??
      device.radio_summary?.clients ??
      '—',
  },
  {
    id: 'offline',
    label: 'Clientes offline',
    cell: ({ device }) => device.pppoe?.offline ?? '—',
  },
  {
    id: 'uptime',
    label: 'Uptime',
    cell: ({ device }) => {
      const s = latestSavedReading(device);
      return s.at ? <span title={s.exactUptime}>{s.uptime}</span> : '—';
    },
  },
  {
    id: 'cpu',
    label: 'CPU',
    cell: ({ device }) => metric(latestSavedReading(device).cpu, '%'),
  },
  {
    id: 'last_reading',
    label: 'Última leitura',
    cell: ({ device }) => {
      const now = Date.now();
      const fault = device.live?.fault;
      if (fault === 'refused' || fault === 'key_changed')
        return (
          <span className="st-off">
            {fault === 'refused' ? 'SENHA INCORRETA' : 'CHAVE SSH ALTERADA'}
          </span>
        );
      const at = latestSavedReading(device, now).at;
      return at ? (
        <span title={formatDate(at)}>
          há {formatDuration(Math.max(0, now - Date.parse(at)) / 1000)}
        </span>
      ) : (
        'Ainda sem leitura'
      );
    },
  },
  { id: 'memory', label: 'Memória', cell: ({ device }) => memory(device) },
  {
    id: 'temperature',
    label: 'Temperatura',
    cell: ({ device }) => {
      const d = latest(device);
      return metric(d.radio?.temperature ?? d.monitoring?.temperature, '°C');
    },
  },
  {
    id: 'voltage',
    label: 'Tensão',
    cell: ({ device }) => metric(latest(device).radio?.voltage, 'V'),
  },
  {
    id: 'system',
    label: 'Sistema',
    cell: ({ device }) => latestSavedReading(device).version || '—',
  },
  {
    id: 'credential',
    label: 'Credencial',
    cell: ({ device, credentials }) =>
      credentials.find((c) => c.id === device.credential_id)?.alias || '—',
  },
  {
    id: 'query',
    label: 'Consulta',
    cell: ({ device }) => device.transport?.toUpperCase() || '—',
  },
];
export const defaultColumns = [
  'equipment',
  'status',
  'role',
  'block',
  'model',
  'address',
  'online',
  'offline',
  'uptime',
  'cpu',
  'last_reading',
];
export const columnById = (id: string) => columns.find((c) => c.id === id);

const pulseColors: Record<string, string> = {
  ONLINE: 'st-on',
  OFFLINE: 'st-off',
};
function pulseCell({ device }: Context) {
  const p = device.pulse;
  const age = p?.checked_at
    ? Math.max(0, (Date.now() - Date.parse(p.checked_at)) / 1000)
    : undefined;
  const state = device.active && age !== undefined && age <= 90 ? p?.state : '';
  const title =
    age === undefined ? 'Ainda sem ping' : `Ping há ${formatDuration(age)}`;
  return (
    <span className={pulseColors[state || '']} title={title}>
      {state || '—'}
    </span>
  );
}

// Values are typed per metric; IPs, versions and names always keep text ordering.
const sortMetrics: Record<string, (c: Context) => number | string> = {
  cpu: (c) => latestSavedReading(c.device, c.now).cpu ?? '',
  online: ({ device: d, now }) =>
    liveReading(d, now)?.clients ??
    d.pppoe?.online ??
    d.radio_summary?.clients ??
    '',
  offline: ({ device }) => device.pppoe?.offline ?? '',
  uptime: (c) => durationSeconds(latestSavedReading(c.device, c.now).uptime),
  temperature: ({ device }) =>
    latest(device).radio?.temperature ??
    latest(device).monitoring?.temperature ??
    '',
  voltage: ({ device }) => latest(device).radio?.voltage ?? '',
  memory: ({ device }) => {
    const text = memory(device);
    return text === '—'
      ? ''
      : Number(text.replace('%', '').replace(',', '.').trim());
  },
  last_reading: (c) => {
    const at = latestSavedReading(c.device, c.now).at;
    return at ? Date.parse(at) : '';
  },
};
export function inventorySortValue(
  id: string,
  c: Context,
): number | string | undefined {
  return sortMetrics[id]?.(c);
}
function durationSeconds(text: string): number | string {
  const units: Record<string, number> = {
    w: 604800,
    d: 86400,
    h: 3600,
    min: 60,
    m: 60,
    s: 1,
  };
  const values = [...text.matchAll(/(\d[\d.,]*)\s*(min|[wdhms])/g)];
  if (!values.length) return '';
  return values.reduce(
    (sum, m) =>
      sum + Number(m[1].replace(/\./g, '').replace(',', '.')) * units[m[2]],
    0,
  );
}
