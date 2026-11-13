import { Fragment } from 'react';
import type { ReactNode } from 'react';
import type { Device } from './equipment-api';
import {
  formatMetric,
  formatDate,
  formatRouterDuration,
  exactVolume,
  formatVolumePair,
} from './network-format';
import { stamp } from './equipment-reading-format';
import { DataBlock } from '../../shared/ui/DataBlock';
import { readingUpdateState } from './reading-update-state';
import type { ReadingUpdateStatus } from './reading-update-state';
export function Summary({
  device,
  statistics,
  status,
}: {
  device: Device;
  statistics?: ReactNode;
  status?: ReadingUpdateStatus;
}) {
  const r = device.result!;
  const pairs = [
    ['Nome', device.identity || r.identity || device.name],
    ['IP', device.address],
    ['Tipo', device.role],
    ['Grupo', device.group_path || 'Sem grupo'],
    ['Estado', device.pulse?.state || 'Não informado nesta leitura'],
    ['Modelo', r.board],
    ['Versão', r.version],
    ['Ligado há', formatRouterDuration(r.uptime)],
    [
      'Último reinício',
      device.last_reboot ? formatDate(device.last_reboot) : '—',
    ],
    ['Visto online por último', lastSeen(device)],
  ];
  return (
    <div className="grid">
      <DataBlock
        title="Equipamento"
        source="SSH"
        age={stamp(device)}
        data={readingUpdateState(
          <>
            <dl className="kv">
              {pairs.map(([key, value]) => (
                <Fragment key={key}>
                  <dt>{key}</dt>
                  <dd>{value || 'Não informado nesta leitura'}</dd>
                </Fragment>
              ))}
            </dl>
          </>,
          status,
        )}
      />
      <DataBlock
        title="Recursos"
        source="SSH"
        age={stamp(device)}
        data={readingUpdateState(
          <dl className="kv">
            <dt>CPU</dt>
            <dd>{formatMetric(r.cpu_percent, '%')}</dd>
            <dt>Memória em uso</dt>
            <dd
              title={`${exactVolume(r.free_memory_bytes)} / ${exactVolume(r.total_memory_bytes)}`}
            >
              {formatVolumePair(
                Math.max(0, r.total_memory_bytes - r.free_memory_bytes),
                r.total_memory_bytes,
              )}
            </dd>
            <dt>Temperatura</dt>
            <dd>{formatMetric(device.monitoring?.temperature, '°C')}</dd>
            <dt>Tensão</dt>
            <dd>Não informado nesta leitura</dd>
            <dt>Latência / perda</dt>
            <dd>Ainda não existe (L-PULSO-5)</dd>
          </dl>,
          status && { ...status, error: undefined },
        )}
      />
      {statistics}
    </div>
  );
}

function lastSeen(device: Device) {
  if (device.pulse?.state === 'ONLINE') return 'agora';
  if (device.pulse?.last_reply) return formatDate(device.pulse.last_reply);
  return 'Nunca respondeu';
}
