import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { pppCatalogAPI } from '../../platform/wails/presentation';
import { useLatestRef } from '../../shared/hooks/use-latest-ref';
import { readCatalogDevices, type CatalogReadInput } from './ppp-catalog-read';
export function usePPPCatalogRefresh(p: CatalogReadInput) {
  const [refresh, bump] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const opening = useRef(false);
  const life = useRef(true);
  const running = useRef(false);
  const latest = useLatestRef(p);
  const read = async (manual: boolean) => {
    const p = latest.current;
    if (running.current || !p.available) return;
    running.current = true;
    setBusy(true);
    setError('');
    try {
      const saved = await pppCatalogAPI.list(
        p.profile,
        p.activation,
        '',
        {
          screen: 'servers',
          search: '',
          filters: {},
          sort: '',
          desc: false,
          page_size: 25,
        },
        '',
      );
      if (!life.current || saved.code !== 'OK')
        throw new Error('Falhou a leitura salva do catálogo PPP');
      const failure = await readCatalogDevices(
        p,
        saved.data?.catalog_dates || {},
        manual,
        () => life.current,
        () => bump((n) => n + 1),
      );
      if (life.current) setError(failure);
    } catch (reason) {
      if (life.current)
        setError(
          reason instanceof Error ? reason.message : 'Falhou a leitura PPP',
        );
    } finally {
      running.current = false;
      if (life.current) setBusy(false);
    }
  };
  useEffect(() => {
    life.current = true;
    return () => {
      life.current = false;
    };
  }, []);
  const autoRead = useEffectEvent(() => void read(false));
  useEffect(() => {
    if (!p.available || !p.devices.length || opening.current) return;
    opening.current = true;
    autoRead();
  }, [p.available, p.devices.length]);
  return { refresh, busy, error, update: () => void read(true) };
}
