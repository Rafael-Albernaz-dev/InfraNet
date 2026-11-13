import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import type { SetStateAction } from 'react';
import type { TablePreference } from './table-model';
export type TablePreferencePort = {
  scope: string;
  load: (id: string) => Promise<TablePreference | undefined>;
  save: (id: string, p: TablePreference) => Promise<void>;
};
export const TablePreferences = createContext<TablePreferencePort | undefined>(
  undefined,
);
export function useTablePreference(id: string, initial: TablePreference) {
  const context = useContext(TablePreferences);
  const port = id ? context : undefined;
  const [state, setState] = useState(initial);
  const [error, setError] = useState('');
  const [loaded, setLoaded] = useState(!port);
  const revision = useRef(0);
  const saved = useRef(JSON.stringify(initial));
  const initialRef = useRef(initial);
  useEffect(() => {
    let alive = true;
    const at = revision.current;
    if (port)
      void port
        .load(id)
        .then((p) => {
          if (alive && revision.current === at) {
            const value = p
              ? {
                  ...initialRef.current,
                  ...p,
                  columns: p.columns || initialRef.current.columns,
                  filters: p.filters || {},
                  manual: p.manual || [],
                  sort: p.sort || null,
                }
              : initialRef.current;
            saved.current = JSON.stringify(value);
            setState(value);
          }
        })
        .catch(() => {
          if (alive) setError('Não foi possível carregar a arrumação');
        })
        .finally(() => {
          if (alive) setLoaded(true);
        });
    return () => {
      alive = false;
    };
  }, [port, id]);
  useEffect(() => {
    const body = JSON.stringify(state);
    if (!loaded || !port || body === saved.current) return;
    let alive = true;
    const timer = setTimeout(() => {
      void port
        .save(id, state)
        .then(() => {
          if (alive) {
            saved.current = body;
            setError('');
          }
        })
        .catch(() => {
          if (alive) setError('Não foi possível salvar a arrumação');
        });
    }, 500);
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [state, port, id, loaded]);
  const change = useCallback((next: SetStateAction<TablePreference>) => {
    revision.current++;
    setState(next);
  }, []);
  return { state, change, error, loaded };
}
