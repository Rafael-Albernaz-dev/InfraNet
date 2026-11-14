import { InterfaceAPAction } from '../equipment/interface-aps';
import type { PPPEntry } from '../../platform/wails/ppp-catalog';
export function PPPServerAPs(p: {
  entry: PPPEntry;
  context: {
    profile: string;
    activation: string;
    openEquipment?: (id: string) => void;
  };
  changed: () => void;
}) {
  const { entry, context } = p;
  return (
    <>
      {entry.ap_devices
        ? Object.entries(entry.ap_devices).map(([id, name], i) => (
            <span key={id}>
              {i > 0 ? ', ' : ''}
              <a
                href={'#ficha/' + id}
                onClick={(e) => {
                  e.preventDefault();
                  context.openEquipment?.(id);
                }}
              >
                {name}
              </a>
            </span>
          ))
        : entry.aps?.join(', ')}{' '}
      <InterfaceAPAction
        profile={context.profile}
        activation={context.activation}
        concentrator={entry.device_id}
        name={entry.concentrator}
        iface={entry.interface || ''}
        linked={Boolean(entry.aps?.length)}
        changed={p.changed}
      />
    </>
  );
}
