import { useMemo } from 'react';
import type { ReactNode } from 'react';
import { TablePreferences } from '../shared/ui/table-preferences';
import { tablePreferenceAPI } from '../platform/wails/presentation';
export function TablePreferenceProvider(p: {
  profile: string;
  activation: string;
  children: ReactNode;
}) {
  const port = useMemo(
    () =>
      p.profile && p.activation
        ? {
            scope: `${p.profile}|${p.activation}`,
            load: (id: string) =>
              tablePreferenceAPI.get(p.profile, p.activation, id),
            save: (
              id: string,
              state: Parameters<typeof tablePreferenceAPI.set>[3],
            ) => tablePreferenceAPI.set(p.profile, p.activation, id, state),
          }
        : undefined,
    [p.profile, p.activation],
  );
  return (
    <TablePreferences.Provider value={port} key={port?.scope}>
      {p.children}
    </TablePreferences.Provider>
  );
}
