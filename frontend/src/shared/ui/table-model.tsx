import { Children, Fragment, isValidElement } from 'react';
import type { ReactNode, ReactElement, ComponentProps } from 'react';
export type TablePreference = {
  columns: string[];
  filters: Record<string, string>;
  search: string;
  sort: { column: string; desc: boolean } | null;
  manual: string[];
  page_size: number;
};
export type TableCell = ReactElement<
  ComponentProps<'td'> & {
    'data-value'?: string | number;
    'data-number'?: string | number;
  }
>;
export type TableRow = ReactElement<
  ComponentProps<'tr'> & {
    [key: `data-${string}`]: string | number | boolean | undefined;
  }
>;
export type TableColumn = {
  id: string;
  label: string;
  index: number;
  fixed: boolean;
  title?: string;
  className?: string;
};
export type TableFilter = {
  id: string;
  label: string;
  options: { value: string; label: string }[];
  checkbox?: boolean;
  always?: boolean;
  matches?: (row: TableRow, value: string) => boolean;
};
export function elements(
  node: ReactNode,
): ReactElement<Record<string, unknown>>[] {
  return Children.toArray(node).flatMap((n) => {
    if (!isValidElement<Record<string, unknown>>(n)) return [];
    return n.type === Fragment ? elements(n.props.children as ReactNode) : [n];
  });
}
export function tableText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  return Children.toArray(node)
    .map((child) =>
      isValidElement<{ children?: ReactNode }>(child)
        ? tableText(child.props.children)
        : tableText(child),
    )
    .join(' ')
    .trim();
}
export function rowCells(row: TableRow): TableCell[] {
  return elements(row.props.children) as TableCell[];
}
export type TableValue = string | number | bigint;
export function cellValue(cell: TableCell | undefined): TableValue {
  const numeric = cell?.props['data-number'];
  if (numeric !== undefined && numeric !== '') return tableNumber(numeric);
  if (cell?.props['data-value'] !== undefined) return cell.props['data-value'];
  if (typeof cell?.props.children === 'number') return cell.props.children;
  return tableText(cell?.props.children);
}
export function rowID(row: TableRow, index: number): string {
  return String(row.props['data-row-id'] || row.key || index);
}
export function tableModel(children: ReactNode) {
  const sections = elements(children);
  const head = sections.find((n) => n.type === 'thead');
  const body = sections.find((n) => n.type === 'tbody');
  const headers = elements(
    elements(head?.props.children as ReactNode)[0]?.props.children as ReactNode,
  );
  const primary = headers.findIndex(
    (h) => tableText(h.props.children as ReactNode) !== '',
  );
  const columns = headers.map((h, index) => ({
    id: String(h.props['data-column'] || `column-${index}`),
    label: tableText(h.props.children as ReactNode),
    title: typeof h.props.title === 'string' ? h.props.title : undefined,
    className:
      typeof h.props.className === 'string' ? h.props.className : undefined,
    index,
    fixed: index === primary || !tableText(h.props.children as ReactNode),
  }));
  const all = elements(body?.props.children as ReactNode) as TableRow[];
  const rows = all.filter(
    (r) => !rowCells(r).some((c) => (c.props.colSpan || 1) > 1),
  );
  const empty = all.find((r) =>
    rowCells(r).some((c) => (c.props.colSpan || 1) > 1),
  );
  return { columns, rows, empty };
}
const portuguese = new Intl.Collator('pt-BR', { sensitivity: 'base' });
export function compareTableValues(
  a: TableValue,
  b: TableValue,
  desc: boolean,
): number {
  const missing = (v: TableValue) =>
    v === '' || v === '—' || v === 'Não informado nesta leitura';
  if (missing(a) || missing(b)) return Number(missing(a)) - Number(missing(b));
  const compared =
    typeof a !== 'string' && typeof b !== 'string'
      ? compareTableNumbers(a, b)
      : portuguese.compare(String(a), String(b));
  return compared * (desc ? -1 : 1);
}
export function localTableRows(
  rows: TableRow[],
  columns: TableColumn[],
  p: TablePreference,
  filters: TableFilter[],
) {
  const ids = rows.map(rowID);
  const manual = p.manual
    .filter((id) => ids.includes(id))
    .concat(ids.filter((id) => !p.manual.includes(id)));
  const filtered = rows.filter((r) => {
    if (
      !rowCells(r).some((c) =>
        tableText(c.props.children)
          .toLocaleLowerCase('pt-BR')
          .includes(p.search.toLocaleLowerCase('pt-BR')),
      )
    )
      return false;
    return filters.every(
      (f) =>
        (!f.always && !p.filters[f.id]) ||
        f.matches?.(r, p.filters[f.id] || '') ||
        (!f.matches &&
          String(
            cellValue(
              rowCells(r)[columns.find((c) => c.id === f.id)?.index ?? -1],
            ),
          ) === p.filters[f.id]),
    );
  });
  const column = columns.find((c) => c.id === p.sort?.column);
  return filtered.sort((a, b) =>
    column
      ? compareTableValues(
          cellValue(rowCells(a)[column.index]),
          cellValue(rowCells(b)[column.index]),
          !!p.sort?.desc,
        )
      : manual.indexOf(rowID(a, rows.indexOf(a))) -
        manual.indexOf(rowID(b, rows.indexOf(b))),
  );
}

function tableNumber(value: string | number): TableValue {
  if (typeof value === 'number') return Number.isFinite(value) ? value : '';
  if (/^-?\d+$/.test(value)) return BigInt(value);
  const number = Number(value);
  return Number.isFinite(number) ? number : '';
}
function compareTableNumbers(a: number | bigint, b: number | bigint) {
  if (a < b) return -1;
  if (a > b) return 1;
  return 0;
}
