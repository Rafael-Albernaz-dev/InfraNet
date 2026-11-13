import { useState } from 'react';
import type { DataState } from '../../shared/ui/DataBlock';
import type { Device } from './equipment-api';
import { useEquipmentSnapshot } from './equipment-snapshot';
import { formatDate } from './network-format';
import { DataBlock } from '../../shared/ui/DataBlock';
import { KeyValues } from '../../shared/ui/KeyValues';
export function EquipmentSummaryRestarts(p: {
  profile: string;
  activation: string;
  device: Device;
}) {
  const [refresh, bump] = useState(0);
  const snapshot = useEquipmentSnapshot(
    p.profile,
    p.activation,
    p.device.id,
    p.device.revision,
    refresh,
  );
  const r = snapshot?.rows.find((row) => row.kind === 'restarts');
  return (
    <DataBlock
      title="Reinícios"
      source="coleta local"
      data={restartData(
        snapshot,
        <>
          {' '}
          <KeyValues
            rows={[
              [
                'Último',
                p.device.last_reboot
                  ? formatDate(p.device.last_reboot)
                  : 'Não informado nesta leitura',
              ],
              ['Últimos 7 dias', r?.used ?? 'Não informado nesta leitura'],
              ['Últimos 30 dias', r?.size ?? 'Não informado nesta leitura'],
            ]}
          />
          <p className="muted">Contado quando o tempo ligado volta a zero.</p>
        </>,
        () => bump((n) => n + 1),
      )}
    />
  );
}
function restartData(
  snapshot: ReturnType<typeof useEquipmentSnapshot>,
  content: React.ReactNode,
  retry: () => void,
): DataState {
  if (!snapshot) return { state: 'loading' };
  if (snapshot.failed)
    return {
      state: 'failed',
      reason: 'Falhou a leitura salva',
      technical: 'Projeção local indisponível',
      retry,
    };
  return { state: 'ready', content };
}
