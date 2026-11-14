import { readingAge } from '../equipment/equipment-reading-format';
import { CSVNotice } from './pppoe-csv-notice';
import { PPPCatalogPanel } from './ppp-catalog-panel';
import { usePPPCatalogRefresh } from './use-ppp-catalog-refresh';
import { PPPoEHeader, PPPoEStatusBanners } from './pppoe-header';
import { observationReadingStatus } from './pppoe-observation-state';
import { PPPoETabContent } from './pppoe-observation-view';
import type { PPPoETab } from './pppoe-state';
import { readingUpdateState } from '../equipment/reading-update-state';
import { DataBlock } from '../../shared/ui/DataBlock';
const PPPOE_TABS: readonly [PPPoETab, string][] = [
  ['sessions', 'Sessões'],
  ['compare', 'Comparação'],
  ['references', 'Referências'],
  ['catalog', 'VLANs e serviços'],
  ['radius', 'RADIUS'],
  ['monitoring', 'Monitoramento'],
];
type WorkspaceProps = Parameters<typeof PPPoETabContent>[0] & {
  section: 'sessions' | 'services';
};
function ObservationBlock(p: WorkspaceProps) {
  const {
    s,
    refs,
    disabled,
    openEquipment,
    onOpenSession: openSession,
    onFocusOrigin: rememberFocus,
  } = p;
  if (s.tab === 'catalog' || s.tab === 'radius')
    return <PPPoETabContent {...p} />;
  return (
    <DataBlock
      title={
        s.tab === 'sessions'
          ? 'Sessões PPPoE'
          : PPPOE_TABS.find(([id]) => id === s.tab)?.[1] || 'Sessões'
      }
      source={s.tab === 'references' ? 'cadastro local' : 'SSH'}
      age={readingAge(
        s.tab === 'references'
          ? s.savedReferences[0]?.created_at
          : s.current?.provenance.ended_at,
      )}
      data={readingUpdateState(
        <PPPoETabContent
          s={s}
          refs={refs}
          disabled={disabled}
          openEquipment={openEquipment}
          onOpenSession={openSession}
          onFocusOrigin={rememberFocus}
        />,
        observationReadingStatus(s, disabled),
      )}
    />
  );
}
export function PPPoEWorkspace(p: WorkspaceProps) {
  const { s, refs, section } = p;
  const catalog = usePPPCatalogRefresh({
    profile: s.profile,
    activation: s.activation,
    devices: [s.device],
    available: !p.disabled && !!s.device.fingerprint,
  });
  const catalogScreen =
    s.tab === 'servers' ||
    s.tab === 'secrets' ||
    s.tab === 'profiles' ||
    s.tab === 'l2tp'
      ? s.tab
      : undefined;
  const { focusedControlRef, detailIntentRef, detailTriggerRef } = refs;
  return (
    <section
      className="pppoe-workspace"
      onFocusCapture={(event) => {
        focusedControlRef.current = event.target as HTMLElement;
      }}
      onKeyDown={(event) => {
        if (event.key === 'Escape' && (s.selected || s.pair)) {
          ++detailIntentRef.current;
          s.select(undefined);
          s.setPair(undefined);
          if (detailTriggerRef.current?.isConnected)
            detailTriggerRef.current.focus();
          event.preventDefault();
        }
      }}
      aria-label="Diagnóstico PPPoE"
      tabIndex={0}
    >
      <PPPoEHeader s={s} section={section} />
      <PPPoEStatusBanners s={s} />
      {catalogScreen ? (
        <PPPCatalogPanel
          key={catalogScreen}
          profile={s.profile}
          activation={s.activation}
          device={s.device.id}
          screen={catalogScreen}
          openEquipment={p.openEquipment}
          refresh={catalog.refresh}
        />
      ) : (
        <ObservationBlock {...p} />
      )}
      {catalogScreen && catalog.error && <p role="alert">{catalog.error}</p>}
      {s.tab === 'sessions' && !s.current && <CSVNotice />}
    </section>
  );
}
