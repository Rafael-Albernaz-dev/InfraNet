import {
  useCallback,
  useContext,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { NavigationGuard } from '../../shared/navigation-guard';
import { Button } from '../../shared/ui/Button';
import { Dialog } from '../../shared/ui/Dialog';

export function useAddNavigationGuard(
  dirty: boolean,
  blocked: boolean,
  enabled = true,
  running = false,
) {
  const register = useContext(NavigationGuard);
  const current = useRef({ dirty, blocked });
  useLayoutEffect(() => {
    current.current = { dirty, blocked };
  }, [dirty, blocked]);
  const [confirm, setConfirm] = useState(false);
  const pending = useRef<() => void>(() => {});
  const attempt = useCallback((action: () => void) => {
    if (current.current.blocked) return;
    if (!current.current.dirty) {
      action();
      return;
    }
    pending.current = action;
    setConfirm(true);
  }, []);
  useLayoutEffect(() => {
    if (!enabled) return;
    register?.(attempt);
    return () => register?.();
  });
  const dialog = confirm && (
    <Dialog
      portal
      title={running ? 'Sair da importação?' : 'Descartar alterações?'}
      close={() => setConfirm(false)}
      dismissOnBackdrop={false}
    >
      <p>
        {running
          ? 'A fila continua no serviço. Os equipamentos cadastrados aparecem na lista.'
          : 'As alterações deste formulário serão perdidas.'}
      </p>
      <div className="dialog-actions">
        <Button onClick={() => setConfirm(false)}>
          {running ? 'Ficar na importação' : 'Continuar editando'}
        </Button>
        <Button
          kind="primary"
          onClick={() => {
            setConfirm(false);
            pending.current();
          }}
        >
          {running ? 'Sair da tela' : 'Descartar alterações'}
        </Button>
      </div>
    </Dialog>
  );
  return { attempt, dialog };
}
