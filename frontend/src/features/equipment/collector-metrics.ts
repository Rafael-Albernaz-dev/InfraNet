import type { Device } from './equipment-api';
import { formatMetric } from './network-format';

export function collectorMetric(
  device: Device,
  field: 'latency_ms' | 'loss_percent',
) {
  if (!device.active || device.cpe_login) return undefined;
  return device.pulse?.[field];
}
export function collectorMetricText(device: Device) {
  const latency = collectorMetric(device, 'latency_ms');
  const loss = collectorMetric(device, 'loss_percent');
  if (latency === undefined && loss === undefined) return '—';
  return `${formatMetric(latency, 'ms', '—')} / ${formatMetric(loss, '%', '—')}`;
}
