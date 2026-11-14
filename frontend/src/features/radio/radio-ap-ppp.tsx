import { PPPMenuNavigation } from '../pppoe/ppp-menu-navigation';
import { Button } from '../../shared/ui/Button';
import { useContext, useEffect, useState } from 'react';
import { radioAPI } from '../equipment/radio-api';
import type { InterfaceAPs } from '../equipment/radio-api';
import type { RadioEquipmentState } from './radio-equipment-state';
import { DataBlock } from '../../shared/ui/DataBlock';
import { KeyValues } from '../../shared/ui/KeyValues';
export function RadioAPPPP({ state: s }: { state: RadioEquipmentState }) {
  const menu = useContext(PPPMenuNavigation);
  const [refresh, retry] = useState(0);
  const [view, setView] = useState<{
    key: string;
    bindings?: InterfaceAPs[];
    error?: string;
  }>();
  const key = [s.profile, s.activation, s.device.id, refresh].join('|');
  const { profile, activation } = s;
  useEffect(() => {
    let alive = true;
    void Promise.resolve()
      .then(() => radioAPI.interfaceAPs(profile, activation))
      .then((r) => {
        if (!alive) return;
        if (
          r.code !== 'OK' ||
          r.profile_id !== profile ||
          r.activation_id !== activation
        )
          setView({ key, error: 'Falhou a leitura local' });
        else setView({ key, bindings: r.data?.interface_aps || [] });
      })
      .catch(() => {
        if (alive) setView({ key, error: 'Serviço indisponível' });
      });
    return () => {
      alive = false;
    };
  }, [profile, activation, key]);
  const current = view?.key === key ? view : undefined;
  const bindings = current?.bindings?.filter((b) =>
    b.ap_ids.includes(s.device.id),
  );
  const content = bindings?.length ? (
    bindings.map((b) => (
      <KeyValues
        key={b.concentrator_id + b.interface}
        rows={[
          [
            b.concentrator_name || b.concentrator_id,
            <>
              {b.service || 'Não informado nesta leitura'} · {b.interface}
            </>,
          ],
        ]}
      />
    ))
  ) : (
    <>
      <p>Nenhuma interface ligada a este AP.</p>
      {menu?.open && (
        <div className="actions">
          <Button onClick={() => menu.open?.('servers')}>
            Ligar em Servidores PPPoE
          </Button>
        </div>
      )}
    </>
  );
  let data: import('../../shared/ui/DataBlock').DataState = {
    state: 'ready',
    content: (
      <>
        {content}
        {Boolean(bindings?.length) && (
          <p className="muted">
            Clientes destas interfaces procuram a CPE neste AP.
          </p>
        )}
      </>
    ),
  };
  if (!current) data = { state: 'loading' };
  else if (current.error)
    data = {
      state: 'failed',
      reason: current.error,
      technical: current.error,
      retry: () => retry((n) => n + 1),
    };
  return (
    <DataBlock title="Ligado ao PPPoE" source="cadastro local" data={data} />
  );
}
