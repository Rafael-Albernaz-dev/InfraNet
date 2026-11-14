import { SessionEnlace } from './enlace-session';
import { StoredRadioLinkPanel } from './stored-radio-link-panel';
import type { RadioLinkPanelProps } from './radio-link-props';
export function RadioLinkPanel({
  session,
  ...props
}: RadioLinkPanelProps & { session?: boolean }) {
  return session ? (
    <SessionEnlace {...props} />
  ) : (
    <StoredRadioLinkPanel {...props} />
  );
}
