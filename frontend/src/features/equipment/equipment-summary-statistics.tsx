import { Button } from '../../shared/ui/Button';
import { useEffect, useState } from 'react';
import type { Device } from './equipment-api';
import type { PPPSummary } from '../../platform/wails/ppp-catalog';
import { pppCatalogAPI } from '../../platform/wails/presentation';
import { KeyValues } from '../../shared/ui/KeyValues';
import { DataBlock, type DataState } from '../../shared/ui/DataBlock';
import { EquipmentSummaryRestarts } from './equipment-summary-restarts';
type Input = { profile: string; activation: string; device: Device };
function usePPPSummary(p: Input) {
  const [refresh, bump] = useState(0);
  const [state, setState] = useState<{
    key: string;
    value?: PPPSummary;
    error?: string;
  }>();
  const { profile, activation, device } = p;
  const key = [profile, activation, device.id, device.revision, refresh].join(
    '|',
  );
  useEffect(() => {
    let alive = true;
    void Promise.resolve()
      .then(() => pppCatalogAPI.summary(profile, activation, device.id))
      .then((r) => {
        if (!alive) return;
        if (
          r.code !== 'OK' ||
          r.profile_id !== profile ||
          r.activation_id !== activation
        )
          setState({ key, error: 'Falhou a leitura salva' });
        else setState({ key, value: r.data?.summary });
      })
      .catch(() => {
        if (alive) setState({ key, error: 'Serviço indisponível' });
      });
    return () => {
      alive = false;
    };
  }, [profile, activation, device.id, key]);
  return {
    state: state?.key === key ? state : undefined,
    retry: () => bump((n) => n + 1),
  };
}
function summaryData(
  s: ReturnType<typeof usePPPSummary>,
  content: React.ReactNode,
): DataState {
  if (!s.state) return { state: 'loading' };
  if (s.state.error)
    return {
      state: 'failed',
      reason: s.state.error,
      technical: 'Projeção local indisponível',
      retry: s.retry,
    };
  if (!s.state.value) return { state: 'unread', action: s.retry };
  return { state: 'ready', content };
}
export function EquipmentSummaryStatistics(
  p: Input & { onSessions?: () => void; onServers?: () => void },
) {
  const s = usePPPSummary(p);
  const value = s.state?.value;
  const clients = (
    <KeyValues
      rows={[
        ['Online', value?.online ?? 'Não informado nesta leitura'],
        ['Offline (24 h)', value?.offline ?? 'Não informado nesta leitura'],
        ['Quedas hoje', value?.drops_today ?? 'Não informado nesta leitura'],
        ['Planos em uso', value?.plans ?? 'Não informado nesta leitura'],
      ]}
    />
  );
  const servers = serverValues(value);
  return (
    <>
      <DataBlock
        title="Clientes"
        source="SSH"
        data={summaryData(
          s,
          <>
            {clients}
            {p.onSessions && (
              <div className="actions">
                <Button onClick={p.onSessions}>Ver sessões</Button>
              </div>
            )}
          </>,
        )}
      />
      <DataBlock
        title="Servidores PPPoE"
        source="SSH"
        data={summaryData(
          s,
          <>
            {servers}
            {p.onServers && (
              <div className="actions">
                <Button onClick={p.onServers}>Ver servidores</Button>
              </div>
            )}
          </>,
        )}
      />
      <EquipmentSummaryRestarts {...p} />
    </>
  );
}

function serverValues(value: PPPSummary | undefined) {
  return (
    <KeyValues
      rows={[
        ['Serviços', value?.servers ?? 'Não informado nesta leitura'],
        ['Inválidos', value?.invalid_servers ?? 'Não informado nesta leitura'],
        [
          'Com queda agora',
          value?.falling_servers ?? 'Não informado nesta leitura',
        ],
        [
          'Sem AP ligado',
          value?.unlinked_servers ?? 'Não informado nesta leitura',
        ],
      ]}
    />
  );
}
