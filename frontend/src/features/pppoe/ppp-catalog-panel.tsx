import { PPPExportActions } from './ppp-export-actions';
import { usePPPPageSamples } from './ppp-page-samples';
import type { Device } from '../equipment/equipment-api';
import { PPPServerPolicy } from './ppp-server-policy';
import { useContext, useState } from 'react';
import { SessionRoute } from '../equipment/session-navigation';
import { readingAge } from '../equipment/equipment-reading-format';
import { Button } from '../../shared/ui/Button';
import type { DataState } from '../../shared/ui/DataBlock';
import { DataBlock } from '../../shared/ui/DataBlock';
import { DataTable } from '../../shared/ui/DataTable';
import type { PPPEntry } from '../../platform/wails/ppp-catalog';
import { PPP_SCREENS, pppValue } from './ppp-columns';
import { usePPPTable, type PPPTableInput } from './use-ppp-table';
export function PPPCatalogPanel(p: PPPTableInput & { devices?: Device[] }) {
  const table = usePPPTable(p);
  const navigation = useContext(SessionRoute);
  const data = table.result?.data || table.previous?.data;
  const entries = data?.entries || [];
  const sampled = usePPPPageSamples({
    profile: p.profile,
    activation: p.activation,
    entries,
    active: p.active !== false && p.screen === 'sessions' && !p.device,
  });
  const dates = pppReadingDates(p.screen, entries, data?.catalog_dates);
  const content = (
    <>
      {' '}
      {sampled.samplers}
      <PPPTable
        p={p}
        table={table}
        entries={sampled.entries}
        open={(e) => {
          if (!e.session || !e.observation_id) return;
          navigation?.open({
            device: e.device_id,
            observation: e.observation_id,
            row: e.session.row_id,
            login: e.name,
            originPPP: !p.device,
          });
        }}
      />
    </>
  );
  return (
    <DataBlock
      title={PPP_SCREENS.find(([id]) => id === p.screen)?.[1] || 'PPP'}
      source={p.device ? 'SSH' : 'SSH · todos os concentradores'}
      age={readingAge(dates[0])}
      data={tableData(table, content, entries.length > 0)}
    />
  );
}
function PPPTable({
  p,
  table,
  entries,
  open,
}: {
  p: PPPTableInput;
  table: ReturnType<typeof usePPPTable>;
  entries: PPPEntry[];
  open: (e: PPPEntry) => void;
}) {
  const [policy, setPolicy] = useState({ threshold: 50, minimum: 5 });
  const data = table.result?.data || table.previous?.data;
  const filters = pppFilters(p.screen);
  if (!p.device) filters.push('concentrator');
  return (
    <>
      {p.screen === 'sessions' && !p.device && (
        <PPPExportActions
          profile={p.profile}
          activation={p.activation}
          counts={data?.counts}
        />
      )}
      {p.screen === 'servers' && (
        <PPPServerPolicy
          onChange={(threshold, minimum) => setPolicy({ threshold, minimum })}
        />
      )}
      {p.screen === 'secrets' && (
        <p className="muted">
          Cópia do cadastro mantida pelo SGP no roteador; quem autentica é o
          RADIUS. A senha nunca é lida. O nome do cliente vem do comentário e
          fica só neste computador.
        </p>
      )}
      {p.screen === 'l2tp' && <p className="muted">A senha nunca é lida.</p>}
      {data?.catalog_limited && <p role="status">Lista limitada</p>}
      <DataTable
        tableId={'ppp-' + p.screen}
        aria-label={PPP_SCREENS.find(([id]) => id === p.screen)?.[1]}
        preference={table.preference}
        service={table.service}
        filters={filters.map((id) => ({
          id,
          label: table.columns.find((c) => c.id === id)?.label || id,
          options: (data?.options?.[id] || []).map((value) => ({
            value,
            label: value,
          })),
        }))}
      >
        <thead>
          <tr>
            {table.columns.map((c) => (
              <th key={c.id} data-column={c.id}>
                {c.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr key={e.device_id + e.id} data-row-id={e.device_id + e.id}>
              {table.columns.map((c, i) => (
                <td key={c.id} data-value={pppValue(e, c.id)}>
                  <PPPTableCell
                    entry={e}
                    column={c.id}
                    primary={i === 0}
                    policy={policy}
                    open={open}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </DataTable>
    </>
  );
}

function tableData(
  table: ReturnType<typeof usePPPTable>,
  content: React.ReactNode,
  present: boolean,
): DataState {
  if (!table.result)
    return { state: present ? 'refreshing' : 'loading', content };
  if (table.result.code !== 'OK')
    return {
      state: 'failed',
      reason: 'Falhou a leitura salva',
      technical: table.result.code,
      retry: table.retry,
      previous: content,
    };
  return { state: 'ready', content };
}
function pppFilters(screen: PPPTableInput['screen']) {
  if (screen === 'sessions')
    return ['state', 'profile', 'vlan', 'service', 'duplicates'];
  if (screen === 'servers') return ['state', 'aps'];
  if (screen === 'secrets')
    return ['profile', 'registration', 'state', 'last_disconnect_reason'];
  return [];
}
function PPPTableCell({
  entry,
  column,
  primary,
  policy,
  open,
}: {
  entry: PPPEntry;
  column: string;
  primary: boolean;
  policy: { threshold: number; minimum: number };
  open: (e: PPPEntry) => void;
}) {
  if (primary && entry.session)
    return (
      <Button className="row-trigger" onClick={() => open(entry)}>
        {entry.name}
      </Button>
    );
  const value = pppValue(entry, column);
  if (
    column === 'drop' &&
    entry.drop !== undefined &&
    entry.peak !== undefined &&
    entry.drop >= policy.threshold &&
    entry.peak >= policy.minimum
  )
    return <span className="st-off">{value}</span>;
  if (column === 'state')
    return <span className={pppStateClass(value)}>{value}</span>;
  return value;
}

function pppStateClass(value: string | number) {
  if (value === 'OFFLINE' || value === 'Inválido') return 'st-off';
  if (value === 'ONLINE' || value === 'Válido') return 'st-on';
  return undefined;
}

function pppReadingDates(
  screen: PPPTableInput['screen'],
  entries: PPPEntry[],
  catalog: Record<string, string> | undefined,
) {
  if (screen === 'sessions')
    return entries.flatMap((e) => (e.at ? [e.at] : [])).sort();
  return Object.values(catalog || {}).sort();
}
