import {
  useSessionHistory,
  type HistoryProps,
  type HistoryState,
} from './session-history-state';
import type { PPSessionHistory } from '../../platform/wails/presentation';
import type { Device } from '../equipment/equipment-api';
import { formatDate, formatDuration } from '../equipment/network-format';
import { DataBlock, type DataState } from '../../shared/ui/DataBlock';
import { DataTable } from '../../shared/ui/DataTable';
import { Button } from '../../shared/ui/Button';
const labels = [
  'Conectou',
  'Desconectou',
  'Duração',
  'Motivo',
  'IP',
  'IPv6 prefixo',
  'IPv6 PD',
  'MAC',
  'Serviço',
  'Interface',
  'Concentrador',
];
function historyCells(row: PPSessionHistory, device: Device) {
  const duration = row.disconnected_at
    ? formatDuration(
        Math.max(
          0,
          (Date.parse(row.disconnected_at) - Date.parse(row.connected_at)) /
            1000,
        ),
      )
    : '—';
  const c = row.row.context;
  const values = [
    formatDate(row.connected_at),
    row.disconnected_at ? formatDate(row.disconnected_at) : '—',
    duration,
    row.reason || '—',
    row.row.address || '—',
    '—',
    c?.delegated_prefix || '—',
    row.row.caller_id || '—',
    c?.service || '—',
    c?.interface || '—',
    device.name,
  ];
  if (row.kind === 'gap') return values.map((v, i) => (i < 4 ? v : '—'));
  return values;
}

function HistoryFilters({ s }: { s: HistoryState }) {
  return (
    <div className="bar">
      <label>
        Motivo{' '}
        <select
          aria-label="Motivo"
          value={s.reason}
          onChange={(e) => {
            s.change();
            s.setReason(e.target.value);
          }}
        >
          <option value="">Todos</option>
          <option>Não informado nesta leitura</option>
          <option>Lacuna de coleta</option>
        </select>
      </label>
      <select
        aria-label="Período"
        value={s.days}
        onChange={(e) => {
          s.change();
          s.setDays(Number(e.target.value));
        }}
      >
        <option value={30}>Últimos 30 dias</option>
        <option value={7}>Últimos 7 dias</option>
        <option value={90}>Últimos 90 dias</option>
        <option value={0}>Tudo</option>
      </select>
      <Button onClick={() => void s.exportCSV()}>Exportar CSV</Button>
    </div>
  );
}
function HistoryContent({ s, device }: { s: HistoryState; device: Device }) {
  return (
    <>
      <HistoryFilters s={s} />
      <DataTable plain aria-label="Histórico de conexões">
        <thead>
          <tr>
            {labels.map((label) => (
              <th key={label}>{label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {s.result?.rows.map((row) => (
            <tr key={row.id}>
              {historyCells(row, device).map((v, i) => (
                <td key={labels[i]}>{v}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </DataTable>
      {!s.result?.rows.length && <p>Nenhuma conexão neste período.</p>}
      <div className="actions">
        {s.cursor && <Button onClick={s.change}>Primeira página</Button>}
        {s.result?.next && (
          <Button
            onClick={() => {
              s.setResult(undefined);
              s.setCursor(s.result?.next || '');
            }}
          >
            Próxima página
          </Button>
        )}
      </div>
      <p className="muted">
        O CSV contém dados identificáveis de assinantes. Horários com margem do
        ciclo de coleta. O histórico começa quando a coleta foi ligada.
      </p>
      {s.notice && <p role="status">{s.notice}</p>}
    </>
  );
}
function historyData(s: HistoryState, device: Device): DataState {
  if (!s.result) return { state: 'loading' };
  if (s.result.error)
    return {
      state: 'failed',
      reason: s.result.error,
      technical: 'Projeção local indisponível',
      retry: s.retry,
    };
  return { state: 'ready', content: <HistoryContent s={s} device={device} /> };
}
export function SessionHistory(p: HistoryProps) {
  const s = useSessionHistory(p);
  return (
    <DataBlock
      title="Histórico de conexões"
      source="coleta local"
      data={historyData(s, p.device)}
    />
  );
}
