import { collectorMetricText } from '../equipment/collector-metrics';
import { EquipmentStateLabel } from '../equipment/equipment-state-label';
import { RadioAPPPP } from './radio-ap-ppp';
import { RadioCPEClient } from './radio-cpe-client';
import { equipmentRadioError } from './radio-equipment-status';
import { RadioModes, RadioFamilies } from '../equipment/radio-contract';
import { RadioRoles } from '../equipment/radio-contract';
import type { RadioEquipmentState } from './radio-equipment-state';
import type { RadioReading } from '../equipment/radio-api';
import {
  SummaryAccess,
  SummaryPending,
  SummaryValues,
} from '../equipment/equipment-summary-common';
import { EquipmentSummaryRestarts } from '../equipment/equipment-summary-restarts';
import {
  formatEquipmentRole,
  formatDate,
  formatRouterDuration,
  formatMetric,
} from '../equipment/network-format';
import { readingAge } from '../equipment/equipment-reading-format';
import { radioRatePair } from './radio-table-format';
type Ready = RadioEquipmentState & { reading: RadioReading };
function metric(r: RadioReading, name: string) {
  const m = r.metrics.find(
    (m) =>
      (m.name === name || (name === 'ccq' && m.name === 'ccq_tx')) &&
      m.quality !== 'unavailable',
  );
  return m ? formatMetric(m.value, m.unit) : 'Não informado nesta leitura';
}
export function RadioSummary({ state }: { state: Ready }) {
  const d = state.device,
    r = state.reading;
  const age = readingAge(r.observed_at);

  return (
    <div className="grid">
      <SummaryValues
        title="Equipamento"
        status={radioSummaryStatus(state)}
        age={age}
        rows={[
          ['Nome', d.name],
          ['IP', d.address],
          ['Tipo', formatEquipmentRole(d.role)],
          ['Modelo', r.model || 'Não informado nesta leitura'],
          ['Versão', r.version || 'Não informado nesta leitura'],
          ['Estado', <EquipmentStateLabel value={d.pulse?.state} />],
          ['Ligado há', formatRouterDuration(r.device_uptime)],
          [
            'Último reinício',
            d.last_reboot
              ? formatDate(d.last_reboot)
              : 'Não informado nesta leitura',
          ],
          ['Visto online por último', radioLastSeen(d)],
          ['Grupo', d.group_path || 'Sem grupo'],
          ...cpeOriginRows(d),
        ]}
      />
      <SummaryValues
        title="Recursos"
        age={age}
        rows={[
          ['CPU', metric(r, 'cpu')],
          ['Memória em uso', metric(r, 'memory')],
          ['Temperatura', metric(r, 'temperature')],
          ['Tensão', metric(r, 'voltage')],
          ['Latência / perda', collectorMetricText(d)],
        ]}
      />
      <RadioSummaryWireless state={state} age={age} />
      {d.role === RadioRoles.ap && <RadioAPPPP state={state} />}
      {d.role === RadioRoles.cpe && <RadioCPEClient state={state} summary />}
      {d.role === RadioRoles.ptp && (
        <SummaryPending title="Lado B" slice="L07-DRV" />
      )}
      <EquipmentSummaryRestarts
        profile={state.profile}
        activation={state.activation}
        device={d}
      />
      <SummaryPending title="Configuração" slice="L-CFG" />
      <SummaryAccess device={d} edit={state.onAccess} />
    </div>
  );
}

function radioLastSeen(d: Ready['device']) {
  if (d.pulse?.state === 'ONLINE' && !d.cpe_login) return 'agora';
  if (d.pulse?.last_reply) return formatDate(d.pulse.last_reply);
  return 'Nunca respondeu';
}

function radioSummaryStatus(state: Ready) {
  return {
    refreshing: state.radio.busy,
    error: state.radio.error ? equipmentRadioError(state) : undefined,
    code: state.radio.view?.job?.code,
    retry: () =>
      void state.radio.start(RadioModes.once, RadioFamilies.standalone),
    disabled: state.blocked,
  };
}

function RadioSummaryWireless({ state, age }: { state: Ready; age: string }) {
  const d = state.device,
    r = state.reading;
  const ap = state.devices?.find((d) => d.id === state.radio.view?.link?.ap_id);
  return (
    <SummaryValues
      title="Rádio"
      age={age}
      rows={
        d.role === RadioRoles.ap
          ? [
              ['Clientes', metric(r, 'clients')],
              ['Frequência', metric(r, 'frequency')],
              ['SSID', metric(r, 'ssid')],
              ['Sinal médio', metric(r, 'signal')],
              ['CCQ médio', metric(r, 'ccq')],
              ['Ruído', metric(r, 'noise')],
            ]
          : [
              ['AP', ap?.name || 'Não informado nesta leitura'],
              ['Sinal', metric(r, 'signal')],
              ['SNR', metric(r, 'snr')],
              ['CCQ', metric(r, 'ccq')],
              ['Taxas (TX/RX)', radioRatePair(r.metrics)],
            ]
      }
    />
  );
}

function cpeOriginRows(d: Ready['device']): [string, React.ReactNode][] {
  if (!d.cpe_login) return [];
  const origin =
    d.cpe_login.origin === 'enlace'
      ? 'Criada pelo Enlace em ' +
        formatDate(d.cpe_login.created_at) +
        ' · segue o login'
      : 'Segue o login';
  return [
    ['Origem', origin],
    ['Leitura', 'Ronda leve a cada 15–30 min; completa ao abrir o cliente'],
  ];
}
