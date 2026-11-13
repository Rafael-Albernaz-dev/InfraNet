import { useEffect, useState } from 'react';
import {
  pppCatalogAPI,
  type PPPQuery,
  type PPPScreen,
} from '../../platform/wails/presentation';
import type { PPResult } from '../../platform/wails/pppoe-types';
import { useTablePreference } from '../../shared/ui/table-preferences';
import { pppColumns } from './ppp-columns';
export type PPPTableInput = {
  profile: string;
  activation: string;
  screen: PPPScreen;
  device?: string;
  refresh?: number;
  active?: boolean;
};
export function usePPPTable(p: PPPTableInput) {
  const columns = pppColumns(p.screen, !p.device);
  const preference = useTablePreference(
    'ppp-' + p.screen + (p.device ? '-device' : '-all'),
    {
      columns: columns
        .slice(1)
        .filter((c) => !c.optional)
        .map((c) => c.id),
      search: '',
      filters: {},
      sort: null,
      manual: [],
      page_size: 50,
    },
  );
  const [page, setPage] = useState(1);
  const [reload, bump] = useState(0);
  const [response, setResponse] = useState<{ key: string; result: PPResult }>();
  useEffect(() => {
    if (p.screen !== 'sessions' || p.active === false) return;
    const timer = setInterval(() => bump((n) => n + 1), 5000);
    return () => clearInterval(timer);
  }, [p.screen, p.active]);
  const state = preference.state;
  const query: PPPQuery = {
    screen: p.screen,
    search: state.search,
    filters: state.filters,
    sort: state.sort?.column || '',
    desc: !!state.sort?.desc,
    page_size: state.page_size,
  };
  const body = JSON.stringify(query);
  const key = [
    p.profile,
    p.activation,
    p.device,
    body,
    page,
    reload,
    p.refresh,
  ].join('|');
  useEffect(() => {
    if (!preference.loaded) return;
    let alive = true;
    const load = async () => {
      let cursor = '';
      let result: PPResult | undefined;
      for (let i = 1; i <= page; i++) {
        result = await pppCatalogAPI.list(
          p.profile,
          p.activation,
          p.device || '',
          JSON.parse(body) as PPPQuery,
          cursor,
        );
        if (!alive) return;
        if (result.code !== 'OK' || !result.data?.next_cursor || i === page)
          break;
        cursor = result.data.next_cursor;
      }
      if (alive && result) setResponse({ key, result });
    };
    void load().catch(() => {
      if (alive)
        setResponse({
          key,
          result: {
            code: 'UNAVAILABLE',
            profile_id: p.profile,
            activation_id: p.activation,
          },
        });
    });
    return () => {
      alive = false;
    };
  }, [key, body, page, p.profile, p.activation, p.device, preference.loaded]);
  const result = response?.key === key ? response.result : undefined;
  const valid =
    result?.profile_id === p.profile && result.activation_id === p.activation;
  return {
    columns,
    preference,
    page,
    setPage,
    retry: () => bump((n) => n + 1),
    result: valid ? result : undefined,
    previous: response?.result,
    service: {
      total: tableTotal(valid ? result : response?.result),
      page,
      busy: !result,
      onPage: setPage,
      change: () => setPage(1),
    },
  };
}

function tableTotal(result: PPResult | undefined) {
  return result?.data?.total || 0;
}
