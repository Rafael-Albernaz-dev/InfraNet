import type { Group } from '../equipment/groups';
import { GroupSelect } from '../equipment/group-select';
import { useRef, useState } from 'react';
import type { Credential, Device } from '../equipment/equipment-api';
import { equipmentAPI, equipmentMessage } from '../equipment/equipment-api';
import { Button } from '../../shared/ui/Button';
import { Dialog } from '../../shared/ui/Dialog';
import { Select } from '../../shared/ui/Select';

export function InventoryBulkDialog({
  mode,
  profile,
  activation,
  selected,
  credentials,
  groups,
  close,
  applied,
}: {
  mode: 'block' | 'credential';
  profile: string;
  activation: string;
  selected: Device[];
  credentials: Credential[];
  groups: Group[];
  blocks?: string[];
  close: () => void;
  applied: () => void;
}) {
  const [value, setValue] = useState(''),
    [busy, setBusy] = useState(false),
    [error, setError] = useState('');
  const operation = useRef(crypto.randomUUID());
  async function apply() {
    if (busy || (mode === 'credential' && !value)) return;
    setBusy(true);
    setError('');
    try {
      const input = {
        items: selected.map(({ id, revision }) => ({ id, revision })),
        ...(mode === 'block' ? { group_id: value } : { credential_id: value }),
      };
      const result = await equipmentAPI.bulk(
        profile,
        activation,
        operation.current,
        input,
      );
      if (result.code !== 'OK') {
        setError(equipmentMessage(result.code));
        return;
      }
      applied();
    } catch {
      setError('Alteração incerta. Tente novamente para conferir o recibo.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      title={mode === 'block' ? 'Definir grupo' : 'Definir credencial'}
      close={close}
      inactive={busy}
    >
      {mode === 'block' ? (
        <GroupSelect groups={groups} value={value} change={setValue} />
      ) : (
        <label className="field">
          Credencial
          <Select
            aria-label="Credencial"
            value={value}
            onChange={(e) => setValue(e.target.value)}
          >
            <option value="">Selecione</option>
            {credentials.map((c) => (
              <option key={c.id} value={c.id}>
                {c.alias}
              </option>
            ))}
          </Select>
          <p className="muted">
            Vale para os equipamentos selecionados. Nenhuma consulta é feita
            agora.
          </p>
        </label>
      )}
      {error && <p role="alert">{error}</p>}
      <div className="dialog-actions">
        <Button disabled={busy} onClick={close}>
          Cancelar
        </Button>
        <Button
          kind="primary"
          disabled={busy || (mode === 'credential' && !value)}
          onClick={() => void apply()}
        >
          Aplicar
        </Button>
      </div>
    </Dialog>
  );
}
