import { useInterfaceAPEnrollment } from './interface-ap-enrollment';
import { RadioRoles } from './radio-contract';
import { useState } from 'react';
import { Button } from '../../shared/ui/Button';
import { Dialog } from '../../shared/ui/Dialog';
import { useInterfaceAPs } from './interface-aps-state';
import type { InterfaceAPInput } from './interface-aps-state';
import { InterfaceAPCreate } from './interface-ap-create';
import { Select } from '../../shared/ui/Select';
export function InterfaceAPAction(
  p: InterfaceAPInput & { linked?: boolean; changed?: () => void },
) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        className={p.linked ? 'lin-btn' : undefined}
        onClick={() => setOpen(true)}
      >
        {p.linked ? 'Alterar' : 'Ligar AP'}
      </Button>
      {open && (
        <InterfaceAPDialog
          {...p}
          close={() => setOpen(false)}
          saved={() => {
            setOpen(false);
            p.changed?.();
          }}
        />
      )}
    </>
  );
}
export function InterfaceAPDialog(
  p: InterfaceAPInput & { close: () => void; saved: () => void },
) {
  const [iface, setInterface] = useState(p.iface);
  const state = useInterfaceAPs({ ...p, iface });
  const created = (id: string) => {
    void state.save(id).then((ok) => {
      if (ok) p.saved();
    });
  };
  const registration = useInterfaceAPEnrollment({ ...p, created });
  const blocked = state.busy || registration.enrollment.checking;
  const link = () => {
    if (registration.draft.address)
      void registration.save(state.data?.defaults.ap || '');
    else
      void state.save().then((ok) => {
        if (ok) p.saved();
      });
  };
  return (
    <Dialog title={'Ligar VLAN a APs · ' + p.name} close={p.close} portal wide>
      <label className="campo">
        VLAN / interface PPPoE
        <Select value={iface} onChange={(e) => setInterface(e.target.value)}>
          {state.data?.interfaces.map((e) => (
            <option key={e.id} value={e.interface}>
              {e.interface} · VLAN {e.vlan ?? '—'} · {e.name}
            </option>
          ))}
        </Select>
      </label>
      <fieldset>
        <legend>APs já cadastrados atrás desta VLAN</legend>
        {state.data?.devices
          .filter((d) => d.role === RadioRoles.ap && d.active)
          .map((d) => (
            <label className="campo" key={d.id}>
              <input
                type="checkbox"
                checked={state.data!.selected.includes(d.id)}
                onChange={() => state.toggle(d.id)}
              />
              {d.identity || d.name}{' '}
              <span className="muted">
                · {d.address} · {d.group_path || 'Sem grupo'}
              </span>
            </label>
          ))}
      </fieldset>
      {state.data && (
        <InterfaceAPCreate
          state={registration}
          unlinked={!state.data.selected.length}
          credentials={state.data.credentials}
          defaultCredential={state.data.defaults.ap || ''}
        />
      )}
      <p className="muted">Switch ou PTP com vários APs atrás: marque todos.</p>
      {state.error && <p role="alert">{state.error}</p>}
      <div className="foot">
        <Button disabled={blocked} onClick={p.close}>
          Cancelar
        </Button>
        <Button
          kind="primary"
          disabled={!state.data || blocked || registration.enrollment.uncertain}
          onClick={link}
        >
          Ligar
        </Button>
      </div>
    </Dialog>
  );
}
