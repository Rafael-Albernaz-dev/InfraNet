import type { RadioClient } from '../equipment/radio-api';
import { formatMetric } from '../equipment/network-format';
import { radioMetric, radioRatePair } from './radio-table-format';
export function RadioClientRow({
  client,
  name,
  open,
}: {
  client: RadioClient;
  name: string;
  open?: (id: string) => void;
}) {
  const linked = Boolean(client.cpe_id && open);
  const navigate = () => {
    if (client.cpe_id) open?.(client.cpe_id);
  };
  return (
    <tr
      className={linked ? 'click' : undefined}
      tabIndex={linked ? 0 : undefined}
      onClick={linked ? navigate : undefined}
      onKeyDown={(event) => {
        if (
          linked &&
          event.target === event.currentTarget &&
          (event.key === 'Enter' || event.key === ' ')
        ) {
          event.preventDefault();
          navigate();
        }
      }}
    >
      <td
        data-value={name}
        title={
          client.cpe_id
            ? 'Vínculo confirmado por MAC, interface e BSSID'
            : 'Nome observado; sem vínculo confirmado'
        }
      >
        {ClientName({ client, name, navigate })}
      </td>
      {renderClientMetric(client, 'signal')}
      {renderClientMetric(client, 'ccq', 'ccq_tx')}
      <td data-value={radioMetric(client.metrics, 'rate_tx')?.value ?? ''}>
        {radioRatePair(client.metrics)}
      </td>
      {renderClientMetric(client, 'occupancy')}
    </tr>
  );
}

function ClientName({
  client,
  name,
  navigate,
}: {
  client: RadioClient;
  name: string;
  navigate: () => void;
}) {
  return client.cpe_id ? (
    <a
      href={`#ficha/${client.cpe_id}`}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        navigate();
      }}
    >
      {name}
    </a>
  ) : (
    <>{name}</>
  );
}

function renderClientMetric(client: RadioClient, name: string, fallback = '') {
  const metric =
    radioMetric(client.metrics, name) || radioMetric(client.metrics, fallback);
  return (
    <td data-value={metric?.value ?? ''}>
      {metric
        ? formatMetric(metric.value, metric.unit).replace(' %', '%')
        : 'Não informado nesta leitura'}
    </td>
  );
}
