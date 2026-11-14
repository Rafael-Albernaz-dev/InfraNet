import { createContext } from 'react';
import type { PPPScreen } from '../../platform/wails/ppp-catalog';
export const PPPMenuNavigation = createContext<
  | {
      screen: PPPScreen;
      setScreen: (screen: PPPScreen) => void;
      open?: (screen: PPPScreen) => void;
    }
  | undefined
>(undefined);
