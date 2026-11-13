import { readingAge } from '../equipment/equipment-reading-format';
import { CSVNotice } from './pppoe-csv-notice';
import { ContextCounts } from './pppoe-context';
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
      <ObservationBlock {...p} />
      {s.tab === 'sessions' && !s.current && <CSVNotice />}
      {/* Stays here until L-PPP moves it to Servidores PPPoE (TELAS §4.2). */}
      {s.tab === 'sessions' && s.current && !s.current.context_omitted && (
        <>
          <div className="sp10" />
          <ContextCounts
            key={s.current.observation_id}
            profile={s.profile}
            activation={s.activation}
            device={s.device.id}
            observation={s.current.observation_id}
            at={s.current.provenance.ended_at}
          />
        </>
      )}
    </section>
  );
}
