import { cloneElement } from 'react';
import type { useImportState } from './add-import-state';
import { roleGroups } from './equipment-role-options';
import type { ImportItem } from '../../platform/wails/presentation';
import { Button } from '../../shared/ui/Button';
import { Checkbox } from '../../shared/ui/Checkbox';
import { DataTable } from '../../shared/ui/DataTable';
type State = ReturnType<typeof useImportState>;
const labels: Record<ImportItem['status'], string> = {
  ready: 'Pronta',
  pending: 'Pendente',
  duplicate: 'Duplicada',
  conflict: 'Conflito',
  ignored: 'Ignorada',
};
type renderImportRowProps = {
  item: ImportItem;
  state: State;
  correct: () => void;
};

function renderImportRow(p: renderImportRowProps) {
  const { item, state: s } = p;
  const row = item.row;
  return (
    <tr data-situation={item.status}>
      <td className="import-check">
        <Checkbox
          aria-label={`Selecionar linha ${row.line}`}
          checked={s.selected.includes(row.line)}
          disabled={item.status === 'ignored'}
          onChange={(e) =>
            s.setSelected(
              e.target.checked
                ? [...s.selected, row.line]
                : s.selected.filter((n) => n !== row.line),
            )
          }
        />
      </td>
      <td>{row.line}</td>
      <td>{row.address}</td>
      <td>
        {roleGroups
          .flatMap(([, roles]) => roles)
          .find(([id]) => id === row.role)?.[1] || row.role}
      </td>
      <td>{row.group || '—'}</td>
      <td>{row.credential || 'Sem credencial'}</td>
      <td
        className={
          item.status === 'ready' || item.status === 'ignored'
            ? ''
            : 'import-status-error'
        }
      >
        {labels[item.status] || 'Situação desconhecida'}
      </td>
      <td className="import-reason">{item.reason || '—'}</td>
      <td>
        {item.status === 'ignored' ? (
          <Button onClick={() => s.update(row.line, { ignored: false })}>
            Restaurar
          </Button>
        ) : (
          <>
            <Button onClick={p.correct}>Corrigir</Button>{' '}
            <Button onClick={() => s.update(row.line, { ignored: true })}>
              Ignorar
            </Button>
          </>
        )}
      </td>
    </tr>
  );
}
export function ImportPreviewTable({
  s,
  visible,
  setEditing,
}: {
  s: State;
  visible: ImportItem[];
  setEditing: React.Dispatch<React.SetStateAction<ImportItem | undefined>>;
}) {
  return (
    <DataTable
      tableId="import-preview"
      filters={[
        {
          id: 'situation',
          label: 'Situação',
          options: Object.entries(labels).map(([value, label]) => ({
            value,
            label,
          })),
          matches: (r, v) => r.props['data-situation'] === v,
        },
      ]}
      aria-label="Prévia da importação"
    >
      <thead>
        <tr>
          <th className="import-check"></th>
          <th>Linha</th>
          <th>IP</th>
          <th>Papel</th>
          <th>Grupo</th>
          <th>Credencial</th>
          <th>Situação</th>
          <th>Motivo</th>
          <th>Ação</th>
        </tr>
      </thead>
      <tbody>
        {visible.map((item) =>
          cloneElement(
            renderImportRow({
              item: item,
              state: s,
              correct: () => setEditing(item),
            }),
            { key: item.row.line },
          ),
        )}
        {!visible.length && (
          <tr>
            <td colSpan={9}>Nenhuma linha nesta situação</td>
          </tr>
        )}
      </tbody>
    </DataTable>
  );
}
