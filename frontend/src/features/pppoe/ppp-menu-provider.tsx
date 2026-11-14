import { useState, type ReactNode } from 'react';
import type { PPPScreen } from '../../platform/wails/ppp-catalog';
import { PPPMenuNavigation } from './ppp-menu-navigation';
export function PPPMenuProvider({
  children,
  navigate,
}: {
  children: ReactNode;
  navigate?: () => void;
}) {
  const [screen, setScreen] = useState<PPPScreen>('sessions');
  return (
    <PPPMenuNavigation.Provider
      value={{
        screen,
        setScreen,
        open: (next) => {
          setScreen(next);
          navigate?.();
        },
      }}
    >
      {children}
    </PPPMenuNavigation.Provider>
  );
}
