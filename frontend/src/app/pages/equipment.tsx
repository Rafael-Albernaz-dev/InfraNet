import { PPPMenu } from '../../features/pppoe/ppp-menu';
import { EquipmentGroups } from '../../features/equipment/equipment-groups';
import {
  EquipmentBanners,
  EquipmentDialogs,
} from '../../features/equipment/equipment-surfaces';
import { useContext, useEffect, useLayoutEffect, useRef } from 'react';
import { DetailTitle } from '../../shared/ui/DetailTitle';
import { Button } from '../../shared/ui/Button';
import { equipmentAPI as api } from '../../features/equipment/equipment-api';
import { EquipmentDetailPane } from './equipment-detail';
import { EquipmentInventoryPane } from '../../features/inventory/equipment-inventory';
import { useEquipmentState } from '../../features/equipment/equipment-state';
import { EquipmentCredentials } from '../../features/equipment/equipment-credentials';
import { AddEquipment } from '../../features/equipment/add-equipment';
import { useState } from 'react';
import { SessionRoute } from '../../features/equipment/session-navigation';
import type { SessionRouteTarget } from '../../features/equipment/session-navigation';

function EquipmentCapabilityBody({
  setSurface,
  ...p
}: {
  s: ReturnType<typeof useEquipmentState>;
  profile: string;
  activation: string;
  connected: boolean;
  primaryInHeader: boolean;
  credentialsPage?: boolean;
  groupsPage?: boolean;
  pppPage?: boolean;
  addPage?: boolean;
  onAdd?: () => void;
  onAddBack?: () => void;
  locked: boolean;
  trustOpen: boolean;
  onBackReady?: (back?: () => void) => void;
  setSurface: (surface: HTMLDivElement | null) => void;
}) {
  const { s, profile, activation, connected, primaryInHeader, locked } = p;
  const adding = p.addPage;
  const session = useContext(SessionRoute);
  const ppp = p.pppPage || session?.target?.originPPP;
  if (ppp) return <EquipmentPPPBody {...p} setSurface={setSurface} />;
  return (
    <div className="equipment-capability" ref={setSurface}>
      <EquipmentBanners s={s} locked={locked} trustOpen={p.trustOpen} />
      <div>
        {adding ? (
          <AddEquipment
            key={[profile, activation].join('|')}
            s={s}
            profile={profile}
            activation={activation}
            locked={locked}
            back={() => p.onAddBack?.()}
          />
        ) : p.credentialsPage ? (
          <EquipmentCredentials
            s={s}
            profile={profile}
            activation={activation}
            locked={locked}
          />
        ) : (
          <>
            {p.groupsPage && !s.detailOpen && (
              <EquipmentGroups
                s={s}
                profile={profile}
                activation={activation}
                locked={locked}
              />
            )}
            <div hidden={p.groupsPage}>
              <EquipmentInventoryPane
                key={[profile, activation].join('|')}
                {...s}
                profile={profile}
                activation={activation}
                connected={connected}
                primaryInHeader={primaryInHeader}
                locked={locked}
                onAdd={p.onAdd}
              />
            </div>
            {s.selected && s.detailOpen && (
              <EquipmentDetailPane
                {...s}
                key={s.selected.id}
                selected={s.selected}
                connected={connected}
                locked={locked}
                profile={profile}
                activation={activation}
                onBackReady={p.onBackReady}
              />
            )}
          </>
        )}
      </div>
      {s.detailOpen && !s.selected && (
        <p role="status">Carregando equipamento…</p>
      )}
      <EquipmentJobProgress s={s} profile={profile} activation={activation} />
      <EquipmentDialogs {...p} />
    </div>
  );
}

function EquipmentJobProgress({
  s,
  profile,
  activation,
}: {
  s: ReturnType<typeof useEquipmentState>;
  profile: string;
  activation: string;
}) {
  return (
    <>
      {s.job?.state === 'RUNNING' && (
        <div className="note" role="status">
          {s.job.kind === 'trust.inspect'
            ? 'Conferindo chave SSH sem enviar senha…'
            : 'Leitura em curso…'}{' '}
          <Button
            onClick={() =>
              void s.perform(
                () => api.status(profile, activation, s.job!.id),
                (r) => s.setJob(r.data?.job),
                false,
              )
            }
          >
            Conferir andamento
          </Button>
        </div>
      )}
    </>
  );
}

export type EquipmentProps = {
  profile: string;
  activation: string;
  connected: boolean;
  primaryInHeader?: boolean;
  credentialsPage?: boolean;
  groupsPage?: boolean;
  pppPage?: boolean;
  addPage?: boolean;
  onAdd?: () => void;
  onAddBack?: () => void;
  onTotal?: (context: string, total: number) => void;
  targetID?: string;
  onSelect?: (id: string) => void;
  onDetailChange?: (open: boolean) => void;
  onBackReady?: (back?: () => void) => void;
  onBlockedChange?: (blocked: boolean) => void;
};

export function Equipment(props: EquipmentProps) {
  const external = useContext(SessionRoute);
  const [localSession, setLocalSession] = useState<SessionRouteTarget>();
  return (
    <SessionRoute.Provider
      value={
        external || {
          active: !!localSession,
          target: localSession,
          open: setLocalSession,
          back: () => setLocalSession(undefined),
          leave: () => setLocalSession(undefined),
        }
      }
    >
      <EquipmentContext
        key={props.credentialsPage ? 'credentials' : 'equipment'}
        {...props}
      />
    </SessionRoute.Provider>
  );
}

function EquipmentContext(props: EquipmentProps) {
  const {
    profile,
    activation,
    connected,
    primaryInHeader = false,
    onBlockedChange,
  } = props;
  const adding = !!props.addPage;
  const [surface, setSurface] = useState<HTMLDivElement | null>(null);
  const returnFocus = useRef(false);
  useLayoutEffect(() => {
    if (adding || props.credentialsPage || !returnFocus.current) return;
    const trigger =
      surface?.querySelector<HTMLButtonElement>('[data-add-trigger]');
    if (trigger && !trigger.closest('[hidden]') && !trigger.disabled) {
      trigger.focus({ preventScroll: true });
      returnFocus.current = false;
    }
  }, [adding, props.credentialsPage, surface]);
  const s = useEquipmentState({
    ...props,
    addVisible: adding,
    inventoryNode: surface,
  });
  useEffect(() => {
    onBlockedChange?.(s.busy || s.job?.state === 'RUNNING');
    return () => onBlockedChange?.(false);
  }, [s.busy, s.job?.state, onBlockedChange]);
  const setTitle = useContext(DetailTitle);
  useLayoutEffect(() => {
    setTitle(
      !props.credentialsPage && s.detailOpen ? s.selected?.name || '' : '',
    );
    return () => setTitle('');
  }, [setTitle, props.credentialsPage, s.detailOpen, s.selected?.name]);
  if (!profile)
    return (
      <div className="empty-state">
        <h2>Selecione uma empresa</h2>
        <p>O inventário pertence à empresa ativa.</p>
      </div>
    );
  const locked = s.busy || !connected;
  const trustOpen = false;
  return (
    <EquipmentCapabilityBody
      s={s}
      profile={profile}
      activation={activation}
      connected={connected}
      primaryInHeader={primaryInHeader}
      credentialsPage={props.credentialsPage}
      groupsPage={props.groupsPage}
      pppPage={props.pppPage}
      addPage={adding}
      onAdd={props.onAdd}
      onAddBack={() => {
        returnFocus.current = true;
        props.onAddBack?.();
      }}
      setSurface={setSurface}
      locked={locked}
      trustOpen={trustOpen}
      onBackReady={props.onBackReady}
    />
  );
}

function EquipmentPPPBody(p: Parameters<typeof EquipmentCapabilityBody>[0]) {
  const { s, profile, activation, connected, locked, setSurface } = p;
  return (
    <div className="equipment-capability" ref={setSurface}>
      <EquipmentBanners s={s} locked={locked} trustOpen={p.trustOpen} />
      <PPPMenu
        profile={profile}
        activation={activation}
        devices={s.devices}
        available={connected && s.vault && !locked}
        openEquipment={s.openById}
      />
      <EquipmentJobProgress s={s} profile={profile} activation={activation} />
      <EquipmentDialogs {...p} />
    </div>
  );
}
