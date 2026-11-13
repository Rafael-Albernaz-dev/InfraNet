import { useState } from 'react';
import type { Device } from '../equipment/equipment-api';
import type { useEquipmentState } from '../equipment/equipment-state';
import { InventoryBulkDialog } from './inventory-bulk-dialog';
import { InventoryTable } from './inventory-table';
import { Button } from '../../shared/ui/Button';
import { StateNotice } from '../../shared/ui/StateNotice';

type Props = ReturnType<typeof useEquipmentState> & {
  profile: string;
  activation: string;
  connected: boolean;
  primaryInHeader: boolean;
  locked: boolean;
  onAdd?: () => void;
};
export function EquipmentInventoryPane(p: Props) {
  const [bulk, setBulk] = useState<'block' | 'credential'>();
  const [selectedRows, setSelectedRows] = useState<Record<string, Device>>({});
  const state = { ...p, selectedRows, setSelectedRows };
  const selected = Object.values(selectedRows);
  return (
    <section hidden={p.detailOpen}>
      <div className="toolbar">
        <h1 className="grow">Equipamentos</h1>
        <span className="muted">Atualiza sozinha a cada 5 s</span>
        <Button
          kind="primary"
          data-add-trigger
          disabled={p.locked || !p.onAdd}
          onClick={p.onAdd}
        >
          Adicionar equipamento
        </Button>
      </div>
      {selected.length > 0 &&
        selectedActions(selected, p.locked, setBulk, () => setSelectedRows({}))}
      {!p.ready && !p.listError && (
        <p role="status">Carregando equipamentos…</p>
      )}
      {pulseNotice(p.pulseNetwork)}
      <InventoryTable p={state} />
      {bulk && selected.length > 0 && (
        <InventoryBulkDialog
          mode={bulk}
          profile={p.profile}
          activation={p.activation}
          selected={selected}
          credentials={p.credentials}
          groups={p.groups}
          close={() => setBulk(undefined)}
          applied={() => {
            setBulk(undefined);
            setSelectedRows({});
            p.reload((n) => n + 1);
          }}
        />
      )}
    </section>
  );
}

function selectedActions(
  selected: Device[],
  locked: boolean,
  setBulk: (v: 'block' | 'credential') => void,
  clear: () => void,
) {
  return (
    <div className="inventory-selection" role="status">
      <b>
        {selected.length} selecionado{selected.length === 1 ? '' : 's'}
      </b>
      <Button disabled={locked} onClick={() => setBulk('block')}>
        Definir grupo
      </Button>
      <Button disabled={locked} onClick={() => setBulk('credential')}>
        Definir credencial
      </Button>
      <span className="grow" />
      <Button onClick={clear}>Limpar seleção</Button>
    </div>
  );
}

function pulseNotice(network: string) {
  if (network === 'no_response')
    return (
      <StateNotice>
        Nenhum equipamento respondeu. Verifique a rede deste computador.
      </StateNotice>
    );
  if (network === 'probe_blocked')
    return (
      <StateNotice>
        Este computador não tem permissão para enviar ping. Estado indisponível.
      </StateNotice>
    );
  return null;
}
