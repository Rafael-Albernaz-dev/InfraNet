import { useContext, useState } from 'react';
import {
  SessionRoute,
  type SessionRouteTarget,
} from '../features/equipment/session-navigation';
export function useDesktopSession() {
  const external = useContext(SessionRoute);
  const [target, setTarget] = useState<SessionRouteTarget>();
  return (
    external || {
      active: !!target,
      target,
      open: setTarget,
      back: () => setTarget(undefined),
      leave: () => setTarget(undefined),
    }
  );
}
