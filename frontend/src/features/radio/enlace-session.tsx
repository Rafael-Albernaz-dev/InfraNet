import { useState } from 'react';
import type { ReactNode } from 'react';
import { DataBlock } from '../../shared/ui/DataBlock';
import type { DataState } from '../../shared/ui/DataBlock';
import { InterfaceAPDialog } from '../equipment/interface-aps';
import { equipmentMessage } from '../equipment/equipment-api';
import { PPPCatalogPanel } from '../pppoe/ppp-catalog-panel';
import type { RadioLinkPanelProps } from './radio-link-props';
import { useEnlaceSession } from './enlace-session-state';
import { StoredRadioLinkPanel } from './stored-radio-link-panel';
import { EnlaceSessionContent } from './enlace-session-content';
export function SessionEnlace(p: RadioLinkPanelProps) {
  const state = useEnlaceSession(p);
  const [modal, setModal] = useState(false);
  const [manual, setManual] = useState(false);
  const v = state.view;
  const content = (
    <>
      {v && (
        <EnlaceSessionContent
          state={state}
          panel={p}
          link={() => setModal(true)}
        />
      )}
      {Boolean(v?.failed?.length) && (
        <p role="status">
          APs que não responderam: {v?.failed?.join(', ')}. Os outros continuam.
        </p>
      )}
    </>
  );
  const local = v?.state === 'unlinked' || v?.state === 'offline';
  return (
    <section aria-label="Enlace rádio">
      <DataBlock
        title="Enlace"
        source={local ? 'local' : 'SSH'}
        data={enlaceData(state, content)}
      />
      <PPPCatalogPanel
        profile={p.profile}
        activation={p.activation}
        device={p.device.id}
        screen="servers"
        openEquipment={p.onOpenEquipment}
        concentratorName={p.device.name}
        highlightInterface={v?.interface}
        linkageOnly
      />
      <details
        className="det"
        onToggle={(e) => setManual(e.currentTarget.open)}
      >
        <summary>Vínculo manual (casos especiais)</summary>
        {manual && <StoredRadioLinkPanel {...p} manual session={false} />}
      </details>
      {modal && v && (
        <InterfaceAPDialog
          profile={p.profile}
          activation={p.activation}
          concentrator={p.device.id}
          name={p.device.name}
          iface={v.interface}
          close={() => setModal(false)}
          saved={() => {
            setModal(false);
            state.refresh();
          }}
        />
      )}
    </section>
  );
}
function enlaceData(
  state: ReturnType<typeof useEnlaceSession>,
  content: ReactNode,
): DataState {
  if (state.error)
    return {
      state: 'failed',
      reason: equipmentMessage(state.error),
      technical: state.error,
      retry: state.refresh,
    };
  if (!state.view) return { state: 'loading' };
  return { state: 'ready', content };
}
