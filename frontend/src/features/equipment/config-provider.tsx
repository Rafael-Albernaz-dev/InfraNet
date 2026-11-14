import type { ReactNode } from 'react';
import { ConfigContext, useConfig } from './config-state';
import type { Scope } from './config-state';
export function ConfigProvider({
  children,
  ...scope
}: Scope & { children: ReactNode }) {
  const state = useConfig(scope);
  return (
    <ConfigContext.Provider value={state}>{children}</ConfigContext.Provider>
  );
}
