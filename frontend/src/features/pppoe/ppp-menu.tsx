import { PPPMenuNavigation } from './ppp-menu-navigation';
import { DetailTitle } from '../../shared/ui/DetailTitle';
import { useContext, useEffect, useLayoutEffect, useState } from 'react';
import { PPP_SCREENS } from './ppp-columns';
import { PPPCatalogPanel } from './ppp-catalog-panel';
import { usePPPCatalogRefresh } from './use-ppp-catalog-refresh';
import { SessionRoute } from '../equipment/session-navigation';
import { equipmentAPI, type Device } from '../equipment/equipment-api';
import { PPPoE } from './pppoe';
import { Button } from '../../shared/ui/Button';
type Props = {
  profile: string;
  activation: string;
  devices: Device[];
  available: boolean;
  openEquipment: (id: string) => void;
};
export function PPPMenu(p: Props) {
  const menu = useContext(PPPMenuNavigation);
  const screen = menu?.screen || 'sessions';
  const navigation = useContext(SessionRoute);
  const catalog = usePPPCatalogRefresh(p);
  return (
    <>
      <div hidden={navigation?.active}>
        <div className="head">
          <h1 className="grow">
            PPP · {PPP_SCREENS.find(([key]) => key === screen)?.[1]}
          </h1>
          <Button
            disabled={!p.available || catalog.busy}
            onClick={catalog.update}
          >
            Atualizar
          </Button>
        </div>
        <p className="muted">
          Todos os concentradores cadastrados. Para ver um só, use o filtro
          Concentrador ou a aba PPP da ficha dele.
        </p>
        {PPP_SCREENS.map(([key]) => (
          <div key={key} hidden={key !== screen}>
            <PPPCatalogPanel
              profile={p.profile}
              activation={p.activation}
              screen={key}
              active={key === screen && !navigation?.active}
              devices={p.devices}
              openEquipment={p.openEquipment}
              refresh={catalog.refresh}
            />
          </div>
        ))}
        {catalog.error && <p role="alert">{catalog.error}</p>}
      </div>
      {navigation?.active && <PPPMenuSession {...p} />}
    </>
  );
}
function PPPMenuSession(p: Props) {
  const navigation = useContext(SessionRoute);
  const target = navigation?.target;
  const setTitle = useContext(DetailTitle);
  const [loaded, setLoaded] = useState<{ device?: Device; error?: string }>();
  useEffect(() => {
    let alive = true;
    if (target)
      void equipmentAPI
        .get(p.profile, p.activation, target.device)
        .then((r) => {
          if (!alive) return;
          if (
            r.code !== 'OK' ||
            r.profile_id !== p.profile ||
            r.activation_id !== p.activation ||
            r.data?.device?.id !== target.device
          )
            setLoaded({
              error: 'Não foi possível abrir o concentrador da sessão',
            });
          else setLoaded({ device: r.data.device });
        })
        .catch(() => {
          if (alive) setLoaded({ error: 'Serviço indisponível' });
        });
    return () => {
      alive = false;
    };
  }, [p.profile, p.activation, target]);
  useLayoutEffect(() => {
    if (loaded?.device) setTitle(loaded.device.name);
    return () => setTitle('');
  }, [setTitle, loaded]);
  if (!loaded?.device)
    return (
      <p role={loaded?.error ? 'alert' : 'status'}>
        {loaded?.error || 'Carregando sessão…'}
      </p>
    );
  return (
    <PPPoE
      profile={p.profile}
      activation={p.activation}
      device={loaded.device}
      back={() => navigation?.back()}
      refreshAvailable={p.available}
      openEquipment={p.openEquipment}
    />
  );
}
