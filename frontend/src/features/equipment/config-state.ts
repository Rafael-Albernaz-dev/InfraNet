import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import { configAPI } from '../../platform/wails/presentation';
import { equipmentMessage } from './equipment-api';
import type { ConfigChange, ConfigResult, ConfigView } from './config-types';
import { formatDate } from './network-format';

export type Scope = {
  profile: string;
  activation: string;
  device: string;
  source?: string;
  blocked: boolean;
  open: () => void;
};
export function configSituation(view?: ConfigView) {
  const last = view?.versions[0];
  if (!last) return 'Ainda sem cópia';
  return `${view?.status.unchanged ? 'Igual desde' : 'Alterada em'} ${formatDate(last.created_at)}`;
}
export function useConfig(scope: Scope) {
  const { profile, activation, device } = scope;
  const [view, setView] = useState<ConfigView>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [change, setChange] = useState<ConfigChange>();
  const [download, setDownload] = useState('');
  const [message, setMessage] = useState('');
  const alive = useRef(true);
  const read = useRef(0);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const accept = useCallback((r: ConfigResult) => {
    if (!alive.current) return;
    if (r.code !== 'OK') {
      setError(equipmentMessage(r.code));
      return;
    }
    setError('');
    if (r.data) setView(r.data);
  }, []);
  const load = useCallback(async () => {
    const generation = ++read.current;
    try {
      const result = await configAPI.list(profile, activation, device);
      if (generation === read.current) accept(result);
    } catch {
      if (alive.current && generation === read.current)
        setError('Não foi possível abrir as cópias.');
    } finally {
      if (alive.current && generation === read.current) setLoading(false);
    }
  }, [profile, activation, device, accept]);
  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load, scope.source]);
  useEffect(() => {
    if (!view?.busy) return;
    const timer = setInterval(() => void load(), 1000);
    return () => clearInterval(timer);
  }, [view?.busy, load]);
  const run = async (action: () => Promise<ConfigResult>, kind: string) => {
    read.current++;
    setWorking(true);
    setMessage('');
    try {
      const r = await action();
      if (!alive.current) return;
      if (kind === 'save') accept(r);
      else if (r.code === 'OK' || r.code === 'EXPORT_CANCELLED') setError('');
      else setError(equipmentMessage(r.code));
      if (r.code === 'OK' && kind === 'compare') setChange(r.data?.change);
      if (r.code === 'OK' && kind === 'download') {
        setDownload('');
        setMessage('Arquivo guardado.');
      }
    } catch {
      if (alive.current) setError('Não foi possível concluir a operação.');
    } finally {
      if (alive.current) setWorking(false);
    }
  };
  return {
    ...scope,
    view,
    error,
    loading,
    working,
    change,
    setChange,
    download,
    setDownload,
    message,
    load,
    save: () => run(() => configAPI.save(profile, activation, device), 'save'),
    compare: (v: string) =>
      run(() => configAPI.compare(profile, activation, device, v), 'compare'),
    confirmDownload: () =>
      run(
        () => configAPI.download(profile, activation, device, download),
        'download',
      ),
  };
}
export const ConfigContext = createContext<
  ReturnType<typeof useConfig> | undefined
>(undefined);
export const useConfigState = () => useContext(ConfigContext);
