import { useEffect, useRef, useState } from 'react';
import { Dialog } from '../../shared/ui/Dialog';
import { Button } from '../../shared/ui/Button';
import { usePingWindow } from './ping-window-state';
import type { PingWindowTarget } from './ping-window-state';

export function PingWindowDialog(
  p: PingWindowTarget & {
    name: string;
    address: string;
    close: () => void;
  },
) {
  const state = usePingWindow(p);
  const [cancelError, setCancelError] = useState('');
  const close = async () => {
    try {
      const result = await state.stop();
      if (result?.code === 'OK') {
        p.close();
        return;
      }
    } catch {
      /* The open window keeps the cancellation failure visible. */
    }
    setCancelError(
      'Não foi possível confirmar o cancelamento. Tente Fechar novamente.',
    );
  };
  const terminal = useRef<HTMLPreElement>(null);
  const lines = state.result?.rows.map((row) => row.value).join('\n') || '';
  const text = [
    '  SEQ HOST                                     SIZE TTL TIME       STATUS',
    lines,
    state.result?.ping_summary,
  ]
    .filter(Boolean)
    .join('\n');
  useEffect(() => {
    if (terminal.current)
      terminal.current.scrollTop = terminal.current.scrollHeight;
  }, [text]);
  return (
    <Dialog
      title={`Ping pelo ${p.name} para ${p.address}`}
      close={() => void close()}
      portal
      wide
      showClose={false}
    >
      <p className="muted">
        Roda no concentrador até você fechar. Se a sessão cair, aparece timeout,
        como no terminal.
      </p>
      <pre className="term" aria-live="polite" ref={terminal}>
        {text}
      </pre>
      {cancelError && <p role="alert">{cancelError}</p>}
      {state.error && <p role="alert">{state.error}</p>}
      {state.expired && <p role="status">Ping parado após 15 min</p>}
      <div className="actions">
        {state.expired && <Button onClick={state.resume}>Continuar</Button>}
        <Button kind="primary" onClick={() => void close()}>
          Fechar
        </Button>
      </div>
    </Dialog>
  );
}
