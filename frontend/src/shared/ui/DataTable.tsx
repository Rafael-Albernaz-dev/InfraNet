import { cloneElement, useEffect, useMemo, useState } from 'react';
import type { ComponentProps, ReactNode, SetStateAction } from 'react';
import { tableModel, localTableRows, rowCells, rowID } from './table-model';
import type { TablePreference, TableFilter, TableRow } from './table-model';
import { useTablePreference } from './table-preferences';
import { TableControls, TablePager } from './TableControls';
import { Button } from './Button';
import './table.css';
export type TableService = {
  total: number;
  page: number;
  busy?: boolean;
  change: (state: TablePreference) => void;
  onPage: (page: number) => void;
};
type Props = ComponentProps<'table'> & {
  tableId?: string;
  plain?: boolean;
  defaultColumns?: string[];
  filters?: TableFilter[];
  extra?: ReactNode;
  manual?: boolean;
  ready?: boolean;
  emptyMessage?: (state: TablePreference, hasRows: boolean) => ReactNode;
  service?: TableService;
  preference?: ReturnType<typeof useTablePreference>;
  selection?: { ids: string[]; change: (ids: string[]) => void };
};
export function DataTable(p: Props) {
  const { plain, ...props } = p;
  if (plain) return <table {...nativeTableProps(props)} />;
  return <InteractiveTable {...p} />;
}
function useTableController(p: Props) {
  const model = useMemo(() => tableModel(p.children), [p.children]);
  const defaults =
    p.defaultColumns || model.columns.filter((c) => !c.fixed).map((c) => c.id);
  const local = useTablePreference(p.preference ? '' : p.tableId || '', {
    columns: defaults,
    filters: {},
    search: '',
    sort: null,
    manual: [],
    page_size: 50,
  });
  const { state: stored, change, error, loaded } = p.preference || local;
  const state = useMemo(
    () => ({
      ...stored,
      columns: stored.columns.filter((id) =>
        model.columns.some((column) => column.id === id && !column.fixed),
      ),
    }),
    [stored, model.columns],
  );
  useManualOrder(p, model.rows, state, loaded, change);
  const [page, setPage] = useState(1);
  const filters = p.filters || [];
  const rows = p.service
    ? model.rows
    : localTableRows(model.rows, model.columns, state, filters);
  useEffect(() => {
    const selected = p.selection;
    if (!selected || p.ready === false) return;
    const ids = new Set(rows.map(rowID));
    const keep = selected.ids.filter((id) => ids.has(id));
    if (keep.length !== selected.ids.length) selected.change(keep);
  }, [p.selection, p.ready, rows]);
  const total = p.service?.total ?? rows.length;
  const pages = Math.max(1, Math.ceil(total / state.page_size));
  const current = Math.min(p.service?.page || page, pages);
  const visible = p.service
    ? rows
    : rows.slice((current - 1) * state.page_size, current * state.page_size);
  const columns = model.columns
    .filter((c) => (c.fixed && c.label !== '') || (c.fixed && c.index === 0))
    .concat(
      state.columns
        .map((id) => model.columns.find((c) => c.id === id && !c.fixed))
        .filter((c) => c !== undefined),
    );
  columns.push(
    ...model.columns.filter((c) => c.fixed && c.label === '' && c.index > 0),
  );
  const update = (next: TablePreference) => {
    setPage(1);
    change(next);
    p.service?.change(next);
  };
  const sort = (column: string) =>
    update({ ...state, sort: nextTableSort(state.sort, column) });
  return {
    model,
    defaults,
    state,
    change,
    error,
    rows,
    total,
    pages,
    current,
    visible,
    columns,
    update,
    sort,
    setPage,
  };
}
function useManualOrder(
  p: Props,
  rows: TableRow[],
  state: TablePreference,
  loaded: boolean,
  change: (p: SetStateAction<TablePreference>) => void,
) {
  const manualIDs = rows.map(rowID);
  const reconciled = state.manual
    .filter((id) => manualIDs.includes(id))
    .concat(manualIDs.filter((id) => !state.manual.includes(id)));
  const orderBody = JSON.stringify(reconciled);
  useEffect(() => {
    if (
      p.manual &&
      p.ready !== false &&
      loaded &&
      orderBody !== JSON.stringify(state.manual)
    )
      change((current) => ({
        ...current,
        manual: JSON.parse(orderBody) as string[],
      }));
  }, [p.manual, p.ready, loaded, orderBody, state, change]);
}
function nextTableSort(
  sort: TablePreference['sort'],
  column: string,
): TablePreference['sort'] {
  if (sort?.column !== column) return { column, desc: false };
  return sort.desc ? null : { column, desc: true };
}
type Controller = ReturnType<typeof useTableController>;
function InteractiveTable(p: Props) {
  const c = useTableController(p);
  const {
    model,
    defaults,
    state,
    change,
    error,
    total,
    pages,
    current,
    visible,
    columns,
    update,
    setPage,
  } = c;
  const display = nativeTableProps(p);
  return (
    <div
      role="group"
      aria-label={
        p['aria-label'] ||
        `Tabela de ${model.columns.find((column) => column.label)?.label || 'dados'}`
      }
    >
      <TableControls
        columns={model.columns}
        defaults={defaults}
        filters={p.filters || []}
        state={state}
        change={update}
        extra={p.extra}
      />
      <TableOrder p={p} c={c} />
      {error && <p role="status">{error}</p>}
      <div className="rolavel">
        <table {...display}>
          <TableHead p={p} c={c} />
          <tbody>
            {visible.map((r, i) =>
              renderRow({
                p,
                row: r,
                index: i,
                columns,
                state,
                change,
                all: model.rows,
              }),
            )}
            <EmptyTableRows p={p} c={c} />
          </tbody>
        </table>
      </div>
      <TablePager
        page={current}
        pages={pages}
        total={total}
        size={state.page_size}
        onPage={p.service?.onPage || setPage}
        disabled={p.service?.busy || p.ready === false}
        resize={(size) => update({ ...state, page_size: size })}
      />
    </div>
  );
}
function selectionHeader(p: Props, rows: TableRow[]) {
  if (!p.selection) return null;
  const ids = rows.map(rowID);
  return (
    <input
      type="checkbox"
      aria-label="Selecionar todos"
      checked={
        ids.length > 0 && ids.every((id) => p.selection?.ids.includes(id))
      }
      onChange={(e) => p.selection?.change(e.target.checked ? ids : [])}
    />
  );
}
type RowProps = {
  p: Props;
  row: TableRow;
  index: number;
  columns: ReturnType<typeof tableModel>['columns'];
  state: TablePreference;
  change: (p: TablePreference) => void;
  all: TableRow[];
};
function renderRow({ p, row, index, columns, state, change, all }: RowProps) {
  const id = rowID(row, index),
    cells = rowCells(row);
  const manual = p.manual && !state.sort;
  const move = (target: string) => {
    const ids = all.map(rowID);
    const order = state.manual
      .filter((i) => ids.includes(i))
      .concat(ids.filter((i) => !state.manual.includes(i)))
      .filter((i) => i !== id);
    const at = target === 'top' ? 0 : order.length;
    order.splice(at, 0, id);
    change({ ...state, manual: order });
  };
  return cloneElement(
    row,
    {
      key: id,
      draggable: manual,
      onDragStart: manual
        ? (e) => {
            e.dataTransfer.setData('text/plain', id);
          }
        : undefined,
      onDragOver: manual ? (e) => e.preventDefault() : undefined,
      onDrop: manual
        ? (e) => {
            e.preventDefault();
            const source = e.dataTransfer.getData('text/plain');
            const ids = all.map(rowID);
            if (!ids.includes(source) || source === id) return;
            const order = state.manual
              .filter((i) => ids.includes(i))
              .concat(ids.filter((i) => !state.manual.includes(i)))
              .filter((i) => i !== source);
            order.splice(order.indexOf(id), 0, source);
            change({ ...state, manual: order });
          }
        : undefined,
    },
    <>
      {manual && (
        <td className="drag" title="Arrastar para reordenar">
          ⠿
        </td>
      )}
      {columns.map((c) =>
        cells[c.index] ? (
          cloneElement(cells[c.index], { key: c.id })
        ) : (
          <td key={c.id} />
        ),
      )}
      {manual && (
        <td className="lin-menu" onClick={(e) => e.stopPropagation()}>
          <Button
            className="lin-btn"
            aria-label="Mover para o topo"
            onClick={() => move('top')}
          >
            ⤒
          </Button>
          <Button
            className="lin-btn"
            aria-label="Mover para o fim"
            onClick={() => move('end')}
          >
            ⤓
          </Button>
        </td>
      )}
    </>,
  );
}

function TableOrder({ p, c }: { p: Props; c: Controller }) {
  const { state, model, update } = c;
  if (!state.sort && !p.manual) return null;
  return (
    <div className="ord-linha">
      <span className="ord-info">
        {state.sort
          ? `Ordenado por ${model.columns.find((column) => column.id === state.sort?.column)?.label} ${state.sort.desc ? '▼' : '▲'}`
          : 'Ordem manual: arraste ⠿ ou use ⤒ ⤓. Clique no título de uma coluna para ordenar por ela.'}
      </span>
      {state.sort && (
        <Button onClick={() => update({ ...state, sort: null })}>
          {p.manual ? 'Voltar à ordem manual' : 'Tirar ordenação'}
        </Button>
      )}
    </div>
  );
}
function tableSortDirection(state: TablePreference, id: string) {
  if (state.sort?.column !== id) return 'none';
  return state.sort.desc ? 'descending' : 'ascending';
}
function TableHead({ p, c }: { p: Props; c: Controller }) {
  const { state, columns, sort, rows } = c;
  return (
    <thead>
      <tr>
        {p.manual && !state.sort && (
          <th className="drag" aria-label="Arrastar" />
        )}
        {columns.map((column) => (
          <th
            data-column={column.id}
            className={column.className}
            key={column.id}
            title={column.title}
            aria-sort={tableSortDirection(state, column.id)}
          >
            {column.label ? (
              <Button className="th-ord" onClick={() => sort(column.id)}>
                {column.label}
                <span className="ord" aria-hidden="true">
                  {state.sort?.column === column.id &&
                    (state.sort.desc ? '▼' : '▲')}
                </span>
              </Button>
            ) : (
              selectionHeader(p, rows)
            )}
          </th>
        ))}
        {p.manual && !state.sort && (
          <th className="lin-menu" aria-label="Mover" />
        )}
      </tr>
    </thead>
  );
}

const controlProps = new Set([
  'tableId',
  'plain',
  'defaultColumns',
  'filters',
  'extra',
  'manual',
  'service',
  'preference',
  'selection',
  'ready',
  'emptyMessage',
]);
function nativeTableProps(p: Props): ComponentProps<'table'> {
  return Object.fromEntries(
    Object.entries(p).filter(([key]) => !controlProps.has(key)),
  );
}

function EmptyTableRows({ p, c }: { p: Props; c: Controller }) {
  if (c.visible.length || p.service?.busy || p.ready === false) return null;
  let content: ReactNode = 'Nada encontrado';
  if (p.emptyMessage)
    content = p.emptyMessage(c.state, c.model.rows.length > 0);
  else if (c.model.empty) content = rowCells(c.model.empty)[0]?.props.children;
  return (
    <tr>
      <td colSpan={c.columns.length + 2}>{content}</td>
    </tr>
  );
}
