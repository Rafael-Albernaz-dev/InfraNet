import { useEffect } from 'react';
import type { PPQuery } from '../../platform/wails/pppoe-types';
import { useTablePreference } from '../../shared/ui/table-preferences';
import type { TablePreference, TableFilter } from '../../shared/ui/table-model';
import { useLatestRef } from '../../shared/hooks/use-latest-ref';
export type SessionTableControl = {
  tablePreference?: ReturnType<typeof useTablePreference>;
  query: PPQuery;
  change: (q: Partial<PPQuery>) => void;
  total: number;
  history: string[];
  cursor: string;
  next: string;
  setCursor: (c: string) => void;
  setHistory: (h: string[]) => void;
  options: Record<string, string[]>;
  pageState: string;
};
export const SESSION_DEFAULT_COLUMNS = [
  'client',
  'address',
  'plan',
  'download',
  'upload',
  'uptime',
  'state',
  'vlan',
  'service',
  'mtu',
  'linked_device',
];
function sessionQuery(p: TablePreference): Partial<PPQuery> {
  return {
    search: p.search,
    sort: p.sort?.column || '',
    desc: p.sort?.desc || false,
    page_size: p.page_size,
    state: (p.filters.state || '') as PPQuery['state'],
    plan: p.filters.plan || '',
    vlan: p.filters.vlan || '',
    service: p.filters.service || '',
    duplicity: p.filters.duplicity || '',
    duplicates_only: false,
  };
}
export function useSessionTable(control: SessionTableControl | undefined) {
  const fallback = useTablePreference(
    control?.tablePreference ? '' : 'sessions',
    {
      columns: SESSION_DEFAULT_COLUMNS,
      filters: {},
      search: '',
      sort: null,
      manual: [],
      page_size: 50,
    },
  );
  const preference = control?.tablePreference || fallback;
  const latest = useLatestRef(control);
  const body = JSON.stringify(sessionQuery(preference.state));
  useEffect(() => {
    const p = latest.current;
    if (!p || !preference.loaded) return;
    const query = JSON.parse(body) as Partial<PPQuery>;
    if (
      Object.entries(query).some(([k, v]) => p.query[k as keyof PPQuery] !== v)
    )
      p.change(query);
  }, [body, preference.loaded, latest]);
  const filters: TableFilter[] = [
    'state',
    'plan',
    'vlan',
    'service',
    'duplicity',
  ].map((id, i) => ({
    id,
    label: ['Estado', 'Plano', 'VLAN', 'Serviço', 'Duplicidade'][i],
    options: sessionFilterOptions(id, control?.options || {}),
  }));
  const service = control
    ? {
        total: control.total,
        page: control.history.length + 1,
        busy: control.pageState !== 'ready',
        change: () => {},
        onPage: (page: number) => {
          if (page === 1) {
            control.setCursor('');
            control.setHistory([]);
          } else if (page > control.history.length + 1 && control.next) {
            control.setHistory([...control.history, control.cursor]);
            control.setCursor(control.next);
          } else if (page < control.history.length + 1) {
            control.setCursor(control.history[page - 1] || '');
            control.setHistory(control.history.slice(0, page - 1));
          }
        },
      }
    : undefined;
  return { preference, filters, service };
}

function sessionFilterOptions(id: string, options: Record<string, string[]>) {
  if (id === 'state')
    return [
      { value: 'ONLINE', label: 'ONLINE' },
      { value: 'OFFLINE', label: 'OFFLINE' },
    ];
  if (id === 'duplicity')
    return [
      { value: 'with', label: 'Com' },
      { value: 'without', label: 'Sem' },
    ];
  return (options[id] || []).map((value) => ({ value, label: value }));
}
