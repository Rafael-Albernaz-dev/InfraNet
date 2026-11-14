import { useRadioLinkState } from './radio-link-state';
import type { RadioLinkPanelProps } from './radio-link-state';
import { RadioLinkContent } from './radio-link-content';
import { useSessionLinkDetails } from './session-link-details';
import { SessionLinkSummary } from './session-link-summary';
import type { EnlaceView } from '../equipment/radio-api';
export function StoredRadioLinkPanel(
  props: RadioLinkPanelProps & {
    session?: boolean;
    manual?: boolean;
    enlace?: EnlaceView;
  },
) {
  const { rootRef, ...state } = useRadioLinkState({
    ...props,
    refreshOnOpen: props.session,
  });
  const details = useSessionLinkDetails(
    state,
    Boolean(props.session || props.manual),
  );
  const content = state.loading ? (
    <p role="status">Lendo vínculo armazenado…</p>
  ) : (
    <>
      {(props.session || props.manual) && state.link && (
        <SessionLinkSummary
          link={state.link}
          enlace={props.enlace}
          device={details.cpe}
          concentrator={props.device.name}
          names={details.names}
          reading={details.reading}
          open={state.onOpenEquipment}
        />
      )}
      <RadioLinkContent state={state} session={props.session} />
    </>
  );
  return (
    <section
      className="radio-panel"
      ref={rootRef}
      aria-label={props.manual ? 'Vínculo manual' : 'Enlace rádio'}
    >
      {props.session ? (
        content
      ) : (
        <>
          <h3>Enlace rádio</h3>
          <p>Uptime PPPoE: {state.pppoeUptime}</p>
          {content}
        </>
      )}
    </section>
  );
}
