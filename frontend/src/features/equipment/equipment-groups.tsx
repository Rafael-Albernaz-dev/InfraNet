import { groupOptions, groupPath } from './groups';
import { GroupSelect } from './group-select';
import {
  useEquipmentGroups,
  type GroupsProps,
  type GroupsModel,
} from './equipment-groups-state';
import { Button } from '../../shared/ui/Button';
import { Dialog } from '../../shared/ui/Dialog';
import { Field } from '../../shared/ui/Field';
import { DataBlock } from '../../shared/ui/DataBlock';
import { DataTable } from '../../shared/ui/DataTable';
import { Checkbox } from '../../shared/ui/Checkbox';
import { equipmentType } from './equipment-layout';
import { equipmentPulseState } from './equipment-reading-format';
import type { TableFilter } from '../../shared/ui/table-model';
import { roleGroups } from './equipment-role-options';
import './equipment-groups.css';
const pulseClasses: Record<string, string> = {
  ONLINE: 'st-on',
  OFFLINE: 'st-off',
};

export function EquipmentGroups(p: GroupsProps) {
  const m = useEquipmentGroups(p);
  const title =
    m.selected === '*'
      ? 'Todos os equipamentos'
      : groupPath(m.s.groups, m.selected) || 'Sem grupo';
  return (
    <>
      <h1>Grupos</h1>
      <div className="equipment-groups-grid">
        <GroupTree m={m} />
        <DataBlock
          title={title}
          source="cadastro local"
          age="agora"
          data={{ state: 'ready', content: <GroupDevices m={m} /> }}
        />
      </div>
      {m.error && <p role="alert">{m.error}</p>}
      {m.edit && <GroupEditor m={m} />}
    </>
  );
}
function GroupTree({ m }: { m: GroupsModel }) {
  const groups = [
    { value: '*', label: 'Todos os equipamentos' },
    ...groupOptions(m.s.groups),
    { value: '', label: 'Sem grupo' },
  ];
  return (
    <div className="card">
      <ul className="equipment-group-tree" aria-label="Grupos">
        {groups.map((g) => {
          const group = m.s.groups.find((item) => item.id === g.value);
          const depth = group
            ? groupPath(m.s.groups, g.value).split(' › ').length - 1
            : 0;
          return (
            <li key={g.value} className={`group-depth-${depth}`}>
              <Button
                className={m.selected === g.value ? 'on' : ''}
                onClick={() => m.select(g.value)}
                aria-current={m.selected === g.value}
              >
                <span>{group?.name || g.label}</span>
                <span className="muted">{m.count(g.value)}</span>
              </Button>
            </li>
          );
        })}
      </ul>
      <GroupActions m={m} />
      <p className="muted">
        Cada equipamento fica em um grupo. Escolher um grupo mostra também o que
        está nos subgrupos dele.
      </p>
    </div>
  );
}
function GroupActions({ m }: { m: GroupsModel }) {
  return (
    <div className="actions">
      <Button disabled={m.blocked} onClick={() => m.create('')}>
        Novo grupo
      </Button>
      <Button
        disabled={m.blocked || !m.real}
        onClick={() => m.create(m.selected)}
      >
        Novo subgrupo
      </Button>
      <Button
        disabled={m.blocked || !m.real}
        onClick={() =>
          m.real &&
          m.setEdit({
            mode: 'rename',
            id: m.real.id,
            parent: m.real.parent_id,
            name: m.real.name,
          })
        }
      >
        Renomear
      </Button>
      <Button
        disabled={m.blocked || !m.real || m.occupied}
        title="Só grupo vazio, sem equipamentos nem subgrupos, pode ser excluído"
        onClick={m.remove}
      >
        Excluir
      </Button>
    </div>
  );
}
function groupsFilters(): TableFilter[] {
  return [
    {
      id: 'role',
      label: 'Papel',
      options: roleGroups.flatMap(([, roles]) =>
        roles.map(([value, label]) => ({ value, label })),
      ),
      matches: (r, v) => r.props['data-role'] === v,
    },
    {
      id: 'state',
      label: 'Estado',
      options: ['ONLINE', 'OFFLINE'].map((value) => ({ value, label: value })),
      matches: (r, v) => r.props['data-state'] === v,
    },
  ];
}
function GroupDevices({ m }: { m: GroupsModel }) {
  return (
    <>
      {m.checked.length > 0 && (
        <div className="actions">
          <span>Mover para</span>
          <GroupSelect
            groups={m.s.groups}
            value={m.destination}
            change={m.setDestination}
          />
          <Button kind="primary" disabled={m.blocked} onClick={m.move}>
            Mover
          </Button>
        </div>
      )}
      <DataTable
        tableId="groups-equipment"
        filters={groupsFilters()}
        selection={{ ids: m.checked, change: m.setChecked }}
        emptyMessage={() => 'Nenhum equipamento neste grupo'}
        aria-label="Equipamentos do grupo"
      >
        <thead>
          <tr>
            <th></th>
            <th>Equipamento</th>
            <th>Estado</th>
            <th>Papel</th>
            <th>Grupo</th>
            <th>Endereço</th>
            <th>Modelo</th>
          </tr>
        </thead>
        <tbody>
          {m.devices.map((d) => {
            const state = equipmentPulseState(d);
            return (
              <tr
                key={d.id}
                data-row-id={d.id}
                data-role={d.role}
                data-state={state}
                className="click"
                tabIndex={0}
                onClick={() => m.s.openDevice(d)}
                onKeyDown={(e) => {
                  if (e.target !== e.currentTarget) return;
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    m.s.openDevice(d);
                  }
                }}
              >
                <td>
                  <Checkbox
                    aria-label={`Selecionar ${d.name}`}
                    checked={m.checked.includes(d.id)}
                    onChange={(e) =>
                      m.setChecked(
                        e.target.checked
                          ? [...m.checked, d.id]
                          : m.checked.filter((id) => id !== d.id),
                      )
                    }
                  />
                </td>
                <td>{d.name}</td>
                <td className={pulseClasses[state]}>{state || '—'}</td>
                <td>{equipmentType(d.role)}</td>
                <td>{d.group_path || '—'}</td>
                <td>{d.address}</td>
                <td>{d.result?.board || d.radio_summary?.model || '—'}</td>
              </tr>
            );
          })}
        </tbody>
      </DataTable>
    </>
  );
}
function GroupEditor({ m }: { m: GroupsModel }) {
  const edit = m.edit;
  if (!edit) return null;
  let title = 'Novo grupo';
  if (edit.parent) title = 'Novo subgrupo';
  if (edit.mode === 'rename') title = 'Renomear grupo';
  return (
    <Dialog
      title={title}
      close={() => m.setEdit(undefined)}
      inactive={m.blocked}
    >
      <Field
        layout="form"
        label="Nome do grupo"
        value={edit.name}
        maxLength={80}
        onChange={(e) => m.setEdit({ ...edit, name: e.target.value })}
      />
      <div className="dialog-actions">
        <Button onClick={() => m.setEdit(undefined)} disabled={m.blocked}>
          Cancelar
        </Button>
        <Button
          kind="primary"
          disabled={m.blocked || !edit.name}
          onClick={m.save}
        >
          Salvar
        </Button>
      </div>
    </Dialog>
  );
}
