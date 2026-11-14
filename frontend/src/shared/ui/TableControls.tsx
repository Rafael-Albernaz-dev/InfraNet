import { useRef } from 'react';
import type { TableColumn, TableFilter, TablePreference } from './table-model';
import { Button } from './Button';
import { Select } from './Select';
import { Checkbox } from './Checkbox';
import { Popover } from './Popover';
type Props = {
  columns: TableColumn[];
  defaults: string[];
  filters: TableFilter[];
  state: TablePreference;
  change: (p: TablePreference) => void;
  extra?: React.ReactNode;
};
function ColumnChoice({ p, column }: { p: Props; column: TableColumn }) {
  const ids = p.state.columns;
  const at = ids.indexOf(column.id);
  const move = (offset: number) => {
    const next = ids.slice();
    [next[at], next[at + offset]] = [next[at + offset], next[at]];
    p.change({ ...p.state, columns: next });
  };
  return (
    <div className="col-item">
      <label>
        <Checkbox
          checked={at >= 0}
          onChange={(e) =>
            p.change({
              ...p.state,
              columns: e.target.checked
                ? [...ids, column.id]
                : ids.filter((id) => id !== column.id),
            })
          }
        />
        {column.label}
      </label>
      {at >= 0 && (
        <>
          <Button
            aria-label={`Subir ${column.label}`}
            disabled={at === 0}
            onClick={() => move(-1)}
          >
            ↑
          </Button>
          <Button
            aria-label={`Descer ${column.label}`}
            disabled={at === ids.length - 1}
            onClick={() => move(1)}
          >
            ↓
          </Button>
        </>
      )}
    </div>
  );
}
function TableColumns(p: Props) {
  const optional = p.columns.filter((c) => !c.fixed);
  const ordered = p.state.columns
    .map((id) => optional.find((c) => c.id === id))
    .filter((c) => c !== undefined);
  return (
    <Popover label="Colunas">
      <div className="col-item muted">
        {p.columns
          .filter((c) => c.fixed && c.label)
          .map((c) => c.label)
          .join(', ')}{' '}
        sempre aparece
      </div>
      {ordered
        .concat(optional.filter((c) => !p.state.columns.includes(c.id)))
        .map((column) => (
          <ColumnChoice key={column.id} p={p} column={column} />
        ))}
      <Button onClick={() => p.change({ ...p.state, columns: p.defaults })}>
        Restaurar padrão
      </Button>
      <div className="col-item muted">
        Colunas, filtros, ordem e linhas por página ficam salvos para esta
        empresa.
      </div>
    </Popover>
  );
}
export function TableControls(p: Props) {
  const search = useRef<HTMLInputElement>(null);
  return (
    <div className="tb-tools">
      <input
        className="tb-busca"
        ref={search}
        aria-label="Buscar em todas as colunas"
        placeholder="Buscar em todas as colunas"
        value={p.state.search}
        maxLength={160}
        onChange={(e) => p.change({ ...p.state, search: e.target.value })}
      />
      {p.filters
        .filter((f) => !f.checkbox)
        .map((f) => (
          <Select
            key={f.id}
            aria-label={f.label}
            value={p.state.filters[f.id] || ''}
            onChange={(e) =>
              p.change({
                ...p.state,
                filters: { ...p.state.filters, [f.id]: e.target.value },
              })
            }
          >
            <option value="">{f.label}: todos</option>
            {f.options.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
        ))}
      {p.filters
        .filter((f) => f.checkbox)
        .map((f) => (
          <label key={f.id}>
            <Checkbox
              checked={!!p.state.filters[f.id]}
              onChange={(e) =>
                p.change({
                  ...p.state,
                  filters: {
                    ...p.state.filters,
                    [f.id]: e.target.checked ? f.options[0].value : '',
                  },
                })
              }
            />
            {f.label}
          </label>
        ))}
      {p.extra}
      {(p.state.search || Object.values(p.state.filters).some(Boolean)) && (
        <Button
          onClick={() => {
            p.change({ ...p.state, search: '', filters: {} });
            search.current?.focus();
          }}
        >
          Limpar filtros
        </Button>
      )}
      <span className="grow" />
      <TableColumns {...p} />
    </div>
  );
}
export function TablePager(p: {
  page: number;
  pages: number;
  total: number;
  size: number;
  onPage: (page: number) => void;
  resize: (size: number) => void;
  disabled?: boolean;
}) {
  return (
    <div className="pager">
      <Button disabled={p.disabled || p.page <= 1} onClick={() => p.onPage(1)}>
        Início
      </Button>
      <Button
        disabled={p.disabled || p.page <= 1}
        onClick={() => p.onPage(p.page - 1)}
      >
        Anterior
      </Button>
      <span>
        Página {p.page} de {p.pages}
      </span>
      <Button
        disabled={p.disabled || p.page >= p.pages}
        onClick={() => p.onPage(p.page + 1)}
      >
        Próxima
      </Button>
      {!p.disabled && (
        <span
          className="muted"
          role="status"
          aria-label={`${p.total} resultados`}
        >
          {p.total} {p.total === 1 ? 'linha' : 'linhas'}
        </span>
      )}
      <label className="tam">
        Linhas por página{' '}
        <Select
          aria-label="Linhas por página"
          value={p.size}
          onChange={(e) => p.resize(Number(e.target.value))}
        >
          {[25, 50, 100, 200].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </Select>
      </label>
    </div>
  );
}
