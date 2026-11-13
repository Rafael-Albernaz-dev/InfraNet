import { useEffect } from 'react';
import { useLatestRef } from '../../shared/hooks/use-latest-ref';
import {
  pppoeAPI,
  type PPObservation,
} from '../../platform/wails/presentation';
import type { SharedCtx } from './pppoe-state';

type ReadScope = { current: () => SharedCtx; accept: () => boolean };
async function refreshSelected(
  c: SharedCtx,
  current: PPObservation,
  scope: ReadScope,
) {
  const selected = c.selected;
  if (!selected) return;
  const row = await pppoeAPI.detail(
    c.profile,
    c.activation,
    c.device.id,
    current.observation_id,
    selected.row_id,
  );
  if (!scope.accept() || scope.current().selected?.row_id !== selected.row_id)
    return;
  if (row.code === 'OK' && row.data?.rows?.[0]) c.select(row.data.rows[0]);
  else if (row.code === 'OBSERVATION_UNAVAILABLE')
    c.select({
      ...selected,
      state: 'OFFLINE',
      download: undefined,
      upload: undefined,
    });
  else return;
  c.setCurrent(current);
}
async function refreshPage(
  c: SharedCtx,
  current: PPObservation,
  scope: ReadScope,
) {
  const queryKey = JSON.stringify(c.query),
    pageCount = c.history.length;
  const history: string[] = [];
  let cursor = '';
  for (let page = 0; page < pageCount; page++) {
    const r = await pppoeAPI.list(
      c.profile,
      c.activation,
      c.device.id,
      current.observation_id,
      c.query,
      cursor,
    );
    if (!scope.accept() || r.code !== 'OK') return;
    if (!r.data?.next_cursor) break;
    history.push(cursor);
    cursor = r.data.next_cursor;
  }
  const latest = scope.current();
  if (
    !scope.accept() ||
    JSON.stringify(latest.query) !== queryKey ||
    latest.history.length !== pageCount
  )
    return;
  c.setHistory(history);
  c.setCursor(cursor);
  c.setCurrent(current);
}
async function refreshSaved(c: SharedCtx, scope: ReadScope) {
  if (c.busy || c.job) return;
  const generation = c.lifeRef.current.generation;
  const result = await pppoeAPI.current(c.profile, c.activation, c.device.id);
  const current = result.data?.current;
  if (
    !scope.accept() ||
    !c.valid(generation, result) ||
    result.code !== 'OK' ||
    !current
  )
    return;
  if (current.observation_id === c.current?.observation_id) return;
  if (c.selected) await refreshSelected(c, current, scope);
  else await refreshPage(c, current, scope);
}
// Only local projections are refreshed here; the daemon owns SSH scheduling.
export function useSavedSessions(p: SharedCtx) {
  const latest = useLatestRef(p);
  const enabled = p.tab === 'sessions';
  useEffect(() => {
    if (!enabled) return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const scope = { current: () => latest.current, accept: () => !stopped };
    const refresh = async () => {
      try {
        await refreshSaved(latest.current, scope);
      } catch {
        if (!stopped)
          latest.current.setError('Falhou a atualização da lista salva.');
      } finally {
        if (!stopped) timer = setTimeout(() => void refresh(), 5000);
      }
    };
    timer = setTimeout(() => void refresh(), 5000);
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [latest, enabled, p.profile, p.activation, p.device.id]);
}
