import type { useImportState } from './add-import-state';
import { ImportSteps } from './add-import-file';
import { Button } from '../../shared/ui/Button';
import { DataTable } from '../../shared/ui/DataTable';

const statusClasses: Record<string, string> = {
  registered: 'st-on',
  failed: 'st-off',
  connecting: 'spin',
};
const labels: Record<string, string> = {
  queued: 'Na fila',
  connecting: 'Conectando…',
  registered: 'Cadastrado',
  cancelled: 'Cancelada',
  ignored: 'Ignorada',
};
export function ImportProgress({
  state: s,
}: {
  state: ReturnType<typeof useImportState>;
}) {
  const finished = s.items.filter(
    (i) => !['queued', 'connecting'].includes(i.status),
  ).length;
  const percent = Math.floor((finished * 100) / Math.max(1, s.items.length));
  return (
    <>
      <ImportSteps step={3} />
      <div className="bar import-progress-heading">
        <h2>Carregando – {percent}%</h2>
        <span className="muted">
          {finished} de {s.items.length} · 2 por vez, para não pesar na rede
        </span>
        <span className="grow" />
        <Button onClick={() => void s.cancel()}>Cancelar o restante</Button>
      </div>
      <progress
        className="import-progress-bar"
        aria-label="Andamento da importação"
        max={100}
        value={percent}
      />
      <DataTable
        tableId="import-progress"
        filters={[
          {
            id: 'situation',
            label: 'Situação',
            options: Object.entries(labels)
              .map(([value, label]) => ({ value, label }))
              .concat([{ value: 'failed', label: 'Falhou' }]),
            matches: (row, value) => row.props['data-situation'] === value,
          },
        ]}
      >
        <thead>
          <tr>
            <th>Linha</th>
            <th>IP</th>
            <th>Papel</th>
            <th>Grupo</th>
            <th>Situação</th>
            <th>Identidade lida</th>
          </tr>
        </thead>
        <tbody>
          {s.items.map((i) => (
            <tr key={i.row.line} data-situation={i.status}>
              <td>{i.row.line}</td>
              <td>{i.row.address}</td>
              <td>{i.row.role}</td>
              <td>{i.row.group || '—'}</td>
              <td className={statusClasses[i.status]}>
                {i.status === 'failed'
                  ? `Falhou: ${i.reason}`
                  : labels[i.status] || i.status}
              </td>
              <td>{i.identity || '—'}</td>
            </tr>
          ))}
        </tbody>
      </DataTable>
    </>
  );
}

export function ImportDiscarded({
  items,
}: {
  items: import('../../platform/wails/presentation').ImportItem[];
}) {
  const discarded = items.filter((i) =>
    ['failed', 'cancelled'].includes(i.status),
  );
  return (
    <DataTable
      tableId="import-discarded"
      emptyMessage={() => 'Nenhuma linha descartada'}
      aria-label="Linhas descartadas"
    >
      <thead>
        <tr>
          <th>Linha</th>
          <th>IP</th>
          <th>Papel</th>
          <th>Motivo</th>
        </tr>
      </thead>
      <tbody>
        {discarded.map((i) => (
          <tr key={i.row.line}>
            <td>{i.row.line}</td>
            <td>{i.row.address}</td>
            <td>{i.row.role}</td>
            <td>{i.reason}</td>
          </tr>
        ))}
      </tbody>
    </DataTable>
  );
}
