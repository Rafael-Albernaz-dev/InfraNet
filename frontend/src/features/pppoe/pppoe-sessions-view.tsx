import { useEffect, useState } from 'react';
import type { PPReadResult } from '../../platform/wails/pppoe-types';
import type { Device } from '../equipment/equipment-api';
import { usePPReadState } from './pppoe-read-state';
import { cloneElement } from 'react';
import {
  useSessionTable,
  type SessionTableControl,
} from './session-table-control';
import { sessionPlan, planDiffers, appliedSpeed } from './pppoe-plan';
import type { PPRow } from '../../platform/wails/presentation';
import {
  exactRate,
  formatRate,
  formatDate,
  uptimeTitle,
} from '../equipment/network-format';
import { runPPExport } from './pppoe-export';
import { formatPPUptime } from '../equipment/pppoe-format';
import type { PPObservation } from '../../platform/wails/pppoe-types';
import { SessionCPE } from './session-cpe';
import { Button } from '../../shared/ui/Button';
import { DataTable } from '../../shared/ui/DataTable';
export function PPPoEObsActions(p: {
  profile: string;
  activation: string;
  deviceId: string;
  current: PPObservation;
  tab: string;
  base: string;
  disabled: boolean;
  busy: boolean;
  setBusy: (b: boolean) => void;
  setNotice: (s: string) => void;
  setError: (s: string) => void;
  lifeRef: React.MutableRefObject<{ generation: number }>;
  exportFocusRef: React.MutableRefObject<'sanitized' | 'operational' | ''>;
  exportControlRef: React.RefObject<HTMLButtonElement | null>;
  csvExportControlRef: React.RefObject<HTMLButtonElement | null>;
  save: () => void;
}) {
  const { current, disabled, busy, exportControlRef, csvExportControlRef } = p;
  const onExport = (kind: 'sanitized' | 'operational') =>
    runPPExport(kind, {
      profile: p.profile,
      activation: p.activation,
      deviceId: p.deviceId,
      observationId: current.observation_id,
      base: p.base,
      tab: p.tab,
      setBusy: p.setBusy,
      setNotice: p.setNotice,
      setError: p.setError,
      lifeRef: p.lifeRef,
      exportFocusRef: p.exportFocusRef,
    });
  const offline = current.offline || 0;
  const online = current.online || Math.max(0, current.count - offline);
  return (
    <>
      <div className="bar">
        <span className="muted">
          <span>{online} ONLINE</span> · <span>{offline} OFFLINE</span> ·
          <span className="st-on">Ao vivo</span> · leitura de{' '}
          {formatDate(current.provenance.ended_at)}
        </span>
        <span className="grow" />
        <Button
          type="submit"
          disabled={disabled}
          onClick={p.save}
          aria-label="Salvar referência desta leitura"
        >
          Salvar referência
        </Button>
        <Button
          type="submit"
          ref={csvExportControlRef}
          disabled={busy}
          onClick={() => onExport('operational')}
          aria-label="Exportar CSV operacional"
          title="O CSV contém dados identificáveis de assinantes."
        >
          Exportar CSV
        </Button>
        <Button
          type="submit"
          ref={exportControlRef}
          disabled={busy}
          onClick={() => onExport('sanitized')}
        >
          Exportar JSON sanitizado
        </Button>
      </div>
      <p className="muted">
        Lista relida a cada 60 s. Download e Upload: média de 60 s para todos e,
        nas linhas desta página, o instante a cada 2 s
      </p>
      {current.context_omitted && (
        <p role="status">
          Contexto não incluído por limite. As sessões e seus estados foram
          preservados.
        </p>
      )}
    </>
  );
}
function SessionPlanCell({
  row,
  fastTrack,
}: {
  row: PPRow;
  fastTrack?: number;
}) {
  const dl = row.context?.download_limit,
    ul = row.context?.upload_limit;
  return (
    <td title={`${exactRate(dl)} download / ${exactRate(ul)} upload`}>
      {sessionPlan(row.context)}
      <small
        className={`cell-sub${planDiffers(row.context) ? ' diverge' : ''}`}
      >
        {appliedSpeed(row.context)}
        {planDiffers(row.context) && ' · diferente do plano'}
        {fastTrack !== undefined &&
          fastTrack > 0 &&
          ' · limite lido, não garantido'}
      </small>
    </td>
  );
}
type SessionsTableProps = {
  rows: PPRow[];
  device?: Device;
  blocked?: boolean;
  control?: SessionTableControl;
  fastTrack?: number;
  contextOmitted?: boolean;
  profile?: string;
  activation?: string;
  deviceId?: string;
  observation?: string;
  openEquipment?: (id: string) => void;
  onDetail: (row: PPRow, trigger: HTMLElement) => void;
};
function renderPPPoESessionRow(
  p: Omit<SessionsTableProps, 'rows'> & { row: PPRow },
) {
  const { row } = p;
  return (
    <tr className="click" data-row-id={row.row_id}>
      <td>
        <Button
          className="row-trigger"
          aria-label={'Detalhar ' + row.login + ' ' + row.remote_id}
          onClick={(e) => p.onDetail(row, e.currentTarget)}
        >
          {row.login || 'Não informado nesta leitura'}
        </Button>
      </td>
      <td>{row.address || 'Não informado nesta leitura'}</td>
      {SessionPlanCell({ row, fastTrack: p.fastTrack })}
      <td>{row.state === 'OFFLINE' ? '—' : formatRate(row.download)}</td>
      <td>{row.state === 'OFFLINE' ? '—' : formatRate(row.upload)}</td>
      <td title={uptimeTitle(row.uptime)}>
        {row.state === 'OFFLINE' ? '—' : formatPPUptime(row.uptime, true)}
      </td>
      <td>
        <span className={row.state === 'OFFLINE' ? 'st-off' : 'st-on'}>
          {row.state || 'ONLINE'}
        </span>
      </td>
      <td>
        <SessionCPE
          profile={p.profile || ''}
          activation={p.activation || ''}
          deviceId={p.deviceId || ''}
          observation={p.observation || ''}
          row={row.row_id}
          open={p.openEquipment}
        />
      </td>
      {optionalSessionCells(row)}
    </tr>
  );
}
export function PPPoESessionsTable(p: SessionsTableProps) {
  if (p.device && p.observation)
    return <LiveSessionsTable {...p} device={p.device} />;
  return <SessionsTable {...p} />;
}
type PageSample = {
  key: string;
  result?: PPReadResult;
  age: number;
  error: string;
};
function SessionPageSampler(
  p: SessionsTableProps & {
    device: Device;
    sampleKey: string;
    publish: (sample: PageSample) => void;
  },
) {
  const { setRoot, result, sampleAge, error } = usePPReadState({
    profile: p.profile || '',
    activation: p.activation || '',
    device: p.device,
    observation: p.observation || '',
    rows: p.rows.filter((r) => r.state !== 'OFFLINE').map((r) => r.row_id),
    automaticLive: true,
    blocked: p.blocked,
  });
  const { publish, sampleKey } = p;
  useEffect(
    () => publish({ key: sampleKey, result, age: sampleAge, error }),
    [publish, sampleKey, result, sampleAge, error],
  );
  return <span hidden ref={setRoot} />;
}
function LiveSessionsTable(p: SessionsTableProps & { device: Device }) {
  const [sample, publish] = useState<PageSample>();
  const key = p.observation + '|' + p.rows.map((r) => r.row_id).join(',');
  const current = sample?.key === key ? sample : undefined;
  const samples = new Map(current?.result?.rows.map((r) => [r.row_id, r]));
  const rows = p.rows.map((r) => {
    const v = samples.get(r.row_id);
    return v && current && current.age < 6000 && !current.error
      ? { ...r, download: v.download, upload: v.upload }
      : r;
  });
  return (
    <>
      <SessionPageSampler key={key} {...p} sampleKey={key} publish={publish} />
      <SessionsTable {...p} rows={rows} />
      {current?.error && <p role="status">{current.error}</p>}
    </>
  );
}
function SessionsTable(p: SessionsTableProps) {
  const table = useSessionTable(p.control);
  return (
    <div
      className="table-region"
      tabIndex={0}
      aria-label="Tabela de sessões PPPoE"
    >
      <DataTable
        tableId="sessions"
        aria-label="Sessões PPPoE"
        preference={table.preference}
        service={table.service}
        filters={table.filters}
        defaultColumns={[
          'address',
          'plan',
          'download',
          'upload',
          'uptime',
          'state',
          'linked_device',
        ]}
      >
        <thead>
          <tr>
            {[
              ['login', 'Usuário'],
              ['address', 'IP'],
              ['plan', 'Plano'],
              ['download', 'Download'],
              ['upload', 'Upload'],
              ['uptime', 'Conectado há'],
              ['state', 'Estado'],
              ['linked_device', 'CPE'],
              ['caller_id', 'Caller ID'],
              ['vlan', 'VLAN'],
              ['service', 'Serviço'],
              ['interface', 'Interface'],
              ['session_id', 'Session ID'],
              ['duplicates', 'Duplicidade'],
            ].map(([id, label]) => (
              <th key={id} data-column={id}>
                {label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {p.rows.map((row) =>
            cloneElement(renderPPPoESessionRow({ ...p, row }), {
              key: row.row_id,
            }),
          )}
        </tbody>
      </DataTable>
    </div>
  );
}
function optionalSessionCells(row: PPRow) {
  return (
    <>
      <td>{row.caller_id || '—'}</td>
      <td>{row.context?.vlan ?? '—'}</td>
      <td>{row.context?.service || '—'}</td>
      <td>{row.context?.interface || '—'}</td>
      <td>{row.session_id || '—'}</td>
      <td>{row.duplicates > 1 ? 'Com' : 'Sem'}</td>
    </>
  );
}
