import { GroupPathSelect } from './group-select';
import type { Group } from './groups';
import { useState } from 'react';
import { AddEquipmentFields } from './add-equipment-fields';
import { ImportSteps } from './add-import-file';
import type { useImportState } from './add-import-state';
import { ImportPreviewTable } from './add-import-table';
import type { Credential } from './equipment-api';
import { roleGroups } from './equipment-role-options';
import type { ImportItem } from '../../platform/wails/presentation';
import { Button } from '../../shared/ui/Button';
import { Dialog } from '../../shared/ui/Dialog';
import { Select } from '../../shared/ui/Select';

type State = ReturnType<typeof useImportState>;
type CorrectionProps = {
  item: ImportItem;
  state: State;
  credentials: Credential[];
  groups: Group[];
  close: () => void;
};
function Correction(p: CorrectionProps) {
  const [row, setRow] = useState(() => ({
    ...p.item.row,
    role:
      roleGroups
        .flatMap(([, roles]) => roles)
        .find(
          ([id, label]) => id === p.item.row.role || label === p.item.row.role,
        )?.[0] || '',
  }));
  return (
    <Dialog
      portal
      title={`Corrigir linha ${row.line}`}
      close={p.close}
      showClose={false}
    >
      {p.item.reason && (
        <p className="import-status-error" role="alert">
          {p.item.reason}
        </p>
      )}
      <AddEquipmentFields
        row={row}
        correction
        groups={p.groups}
        credentials={p.credentials}
        change={(patch) => setRow({ ...row, ...patch })}
      />
      <div className="dialog-actions">
        <Button onClick={p.close}>Cancelar</Button>
        <Button
          kind="primary"
          onClick={() => {
            p.state.update(row.line, row);
            p.close();
          }}
        >
          Salvar linha
        </Button>
      </div>
    </Dialog>
  );
}
type SelectionDialogProps = {
  mode: 'group' | 'credential';
  state: State;
  credentials: Credential[];
  groups: Group[];
  close: () => void;
};
function SelectionDialog(p: SelectionDialogProps) {
  const [value, setValue] = useState('');
  return (
    <Dialog
      portal
      title={p.mode === 'group' ? 'Definir grupo' : 'Definir credencial'}
      close={p.close}
    >
      {p.mode === 'group' ? (
        <GroupPathSelect groups={p.groups} value={value} change={setValue} />
      ) : (
        <label className="field">
          Credencial
          <Select
            aria-label="Credencial"
            value={value}
            onChange={(e) => setValue(e.target.value)}
          >
            <option value="">Sem credencial</option>
            {p.credentials.map((c) => (
              <option key={c.id} value={c.alias}>
                {c.alias}
              </option>
            ))}
          </Select>
        </label>
      )}
      <div className="dialog-actions">
        <Button onClick={p.close}>Cancelar</Button>
        <Button
          kind="primary"
          onClick={() => {
            void p.state.review(
              p.state.items.map((item) =>
                p.state.selected.includes(item.row.line)
                  ? { ...item.row, [p.mode]: value }
                  : item.row,
              ),
            );
            p.close();
          }}
        >
          Aplicar
        </Button>
      </div>
    </Dialog>
  );
}
type ImportPreviewProps = {
  state: State;
  credentials: Credential[];
  groups: Group[];
  cancel: () => void;
  replace: () => void;
};
export function ImportPreview(p: ImportPreviewProps) {
  const s = p.state;
  const [editing, setEditing] = useState<ImportItem>();
  const [bulk, setBulk] = useState<'group' | 'credential'>();
  const open = s.items.filter(
    (item) => !['ready', 'ignored'].includes(item.status),
  ).length;
  const ready = s.items.filter((item) => item.status === 'ready').length;
  return (
    <>
      <ImportSteps step={2} />
      <PreviewHeader {...p} />
      <PreviewSelection s={s} setBulk={setBulk} />
      <ImportPreviewTable s={s} visible={s.items} setEditing={setEditing} />
      <PreviewFooter {...p} open={open} ready={ready} />
      {s.error && (
        <Button onClick={() => void s.review(s.items.map((item) => item.row))}>
          Atualizar prévia
        </Button>
      )}
      {editing && (
        <Correction
          item={editing}
          state={s}
          credentials={p.credentials}
          groups={p.groups}
          close={() => setEditing(undefined)}
        />
      )}
      {bulk && (
        <SelectionDialog
          mode={bulk}
          state={s}
          credentials={p.credentials}
          groups={p.groups}
          close={() => setBulk(undefined)}
        />
      )}
    </>
  );
}

function PreviewHeader(p: ImportPreviewProps) {
  const s = p.state;
  return (
    <>
      <div className="bar">
        <span className="muted">
          {s.filename} · {s.items.length} linhas · prévia: nada foi gravado nem
          consultado
        </span>
        <span className="grow" />
        <Button disabled={s.busy} onClick={p.replace}>
          Trocar arquivo
        </Button>
      </div>
    </>
  );
}
function PreviewSelection({
  s,
  setBulk,
}: {
  s: State;
  setBulk: (m: 'group' | 'credential') => void;
}) {
  return (
    <>
      {s.selected.length > 0 && (
        <div className="inventory-selection" role="status">
          <b>{s.selected.length} selecionada(s)</b>
          <Button onClick={() => setBulk('group')}>Definir grupo</Button>
          <Button onClick={() => setBulk('credential')}>
            Definir credencial
          </Button>
          <Button
            onClick={() =>
              void s.review(
                s.items.map((item) =>
                  s.selected.includes(item.row.line)
                    ? { ...item.row, ignored: true }
                    : item.row,
                ),
              )
            }
          >
            Ignorar
          </Button>
        </div>
      )}
    </>
  );
}
function PreviewFooter(
  p: ImportPreviewProps & { open: number; ready: number },
) {
  const s = p.state,
    { open, ready } = p;
  return (
    <div className="bar import-footer">
      <span className="muted">
        {open
          ? `Resolva ${open} linha${open === 1 ? '' : 's'} para continuar.`
          : 'Tudo pronto para cadastrar e conectar os equipamentos.'}
      </span>
      <span className="grow" />
      <Button onClick={p.cancel}>Cancelar</Button>
      <Button
        kind="primary"
        disabled={s.busy || !!open || !ready}
        onClick={() => void s.confirm()}
      >
        Cadastrar e conectar {ready} equipamentos
      </Button>
    </div>
  );
}
