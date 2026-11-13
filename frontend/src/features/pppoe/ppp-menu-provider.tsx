import { useState, type ReactNode } from 'react';
import type { PPPScreen } from '../../platform/wails/ppp-catalog';
import { PPPMenuNavigation } from './ppp-menu-navigation';
export function PPPMenuProvider({ children }: { children: ReactNode }) {
  const [screen, setScreen] = useState<PPPScreen>('sessions');
  return (
    <PPPMenuNavigation.Provider value={{ screen, setScreen }}>
      {children}
    </PPPMenuNavigation.Provider>
  );
}
