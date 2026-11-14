import type { Device } from '../equipment/equipment-api';
import type { EnlaceView } from '../equipment/radio-api';
import { formatDate } from '../equipment/network-format';
import { KeyValues } from '../../shared/ui/KeyValues';
import type {
  RadioLink,
  RadioMetric,
  RadioReading,
} from '../equipment/radio-api';

const missing = 'Não informado nesta leitura';
function metric(reading: RadioReading | undefined, name: string) {
  const value =
    reading?.compatible &&
    reading.metrics.find(
      (item) =>
        item.name === name &&
        ['complete', 'observed', 'initial'].includes(item.quality),
    );
  return value && Number.isFinite(value.value) ? value : undefined;
}
function metricText(value: RadioMetric | undefined) {
  if (!value) return missing;
  return `${value.value}${value.unit === '%' ? '' : ' '}${value.unit}`;
}
function rates(reading?: RadioReading) {
  const tx = metric(reading, 'rate_tx'),
    rx = metric(reading, 'rate_rx');
  if (!tx || !rx || tx.unit !== rx.unit) return missing;
  return `${tx.value}/${rx.value} ${tx.unit}`;
}
export function SessionLinkSummary({
  link,
  enlace,
  device: cpe,
  concentrator,
  names,
  reading,
  open,
}: {
  link: RadioLink;
  enlace?: EnlaceView;
  device?: Device;
  concentrator?: string;
  names: Record<string, string>;
  reading?: RadioReading;
  open?: (id: string) => void;
}) {
  const device = (id: string) =>
    names[id] && open ? (
      <a
        href={'#ficha/' + encodeURIComponent(id)}
        onClick={(event) => {
          event.preventDefault();
          open(id);
        }}
      >
        {names[id]}
      </a>
    ) : (
      names[id] || missing
    );
  return (
    <KeyValues
      rows={[
        [
          'Caminho',
          <>
            {concentrator || device(link.concentrator_id)} ›{' '}
            {enlace?.interface ||
              link.ppp_interface ||
              'Não informado nesta leitura'}{' '}
            › {device(link.ap_id)} › {device(link.cpe_id)}
          </>,
        ],
        ['CPE', <EnlaceCPEDescription cpe={cpe} reading={reading} />],
        [
          'IP da CPE',
          <>
            {enlace?.address || cpe?.address || missing} · é o IP desta sessão
          </>,
        ],
        ['Sinal', metricText(metric(reading, 'signal'))],
        ['SNR', metricText(metric(reading, 'snr'))],
        ['CCQ', metricText(metric(reading, 'ccq'))],
        ['Taxas (TX/RX)', rates(reading)],
      ]}
    />
  );
}

function EnlaceCPEDescription({
  cpe,
  reading,
}: {
  cpe?: Device;
  reading?: RadioReading;
}) {
  return (
    <>
      {reading?.model || cpe?.result?.board || missing} ·{' '}
      {reading?.version || cpe?.result?.version || missing}
      {cpe?.cpe_login?.origin === 'enlace' && (
        <> · criada pelo Enlace em {formatDate(cpe.cpe_login.created_at)}</>
      )}{' '}
      · segue o login
    </>
  );
}
