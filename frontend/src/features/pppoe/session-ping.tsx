import { StoredSessionPing } from './stored-session-ping';
import { useEffect, useState } from 'react';
import type { Device } from '../equipment/equipment-api';
type SessionPingProps = {
  profile: string;
  activation: string;
  device: Device;
  observation: string;
  row: string;
  address: string;
  blocked: boolean;
  offline?: boolean;
  beforeOpen?: () => Promise<void>;
  afterClose?: () => void;
};
import { PingWindowDialog } from './ping-window-dialog';
import { Button } from '../../shared/ui/Button';

// A second session replaces the previous window; the daemon also enforces one
// live ping across all clients, including a start still crossing the binding.
let closeActiveWindow: (() => void) | undefined;
export function SessionPing(p: SessionPingProps) {
  const [target, setTarget] = useState<SessionPingProps>();
  const open = Boolean(target);
  useEffect(() => {
    if (!open) return;
    const close = () => setTarget(undefined);
    closeActiveWindow?.();
    closeActiveWindow = close;
    return () => {
      if (closeActiveWindow === close) closeActiveWindow = undefined;
    };
  }, [open]);
  const close = () => {
    setTarget(undefined);
    p.afterClose?.();
  };
  return (
    <>
      {!p.offline && (
        <>
          <p>
            Abre uma janela de terminal; o ping roda no {p.device.name} até você
            fechar.
          </p>
          <div className="actions">
            <Button
              kind="primary"
              disabled={p.blocked || !p.device.active || !p.device.fingerprint}
              onClick={async () => {
                await p.beforeOpen?.();
                setTarget(p);
              }}
            >
              Ping
            </Button>
            <StoredSessionPing {...p} />
          </div>
          <p className="muted">Não responder não significa estar offline.</p>
        </>
      )}
      {target && (
        <PingWindowDialog
          profile={target.profile}
          activation={target.activation}
          deviceId={target.device.id}
          revision={target.device.revision}
          observation={target.observation}
          row={target.row}
          name={target.device.name}
          address={target.address}
          close={close}
        />
      )}
    </>
  );
}
