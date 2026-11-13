import { id } from './equipment-reading-format';
import { useState } from 'react';
import { equipmentAPI, equipmentMessage } from './equipment-api';
import { groupContains } from './groups';
import type { useEquipmentState } from './equipment-state';
export type GroupsProps = {
  s: ReturnType<typeof useEquipmentState>;
  profile: string;
  activation: string;
  locked: boolean;
};
export type GroupEdit = {
  mode: 'create' | 'rename';
  id: string;
  parent: string;
  name: string;
};
export function useEquipmentGroups(p: GroupsProps) {
  const [selected, setSelected] = useState('*');
  const [checked, setChecked] = useState<string[]>([]);
  const [destination, setDestination] = useState('');
  const [edit, setEdit] = useState<GroupEdit>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const { s } = p;
  const selection = groupSelection(s, selected);
  const { devices } = selection;
  const blocked = p.locked || busy;
  const run = async (
    operation: () => ReturnType<typeof equipmentAPI.groups>,
  ) => {
    if (blocked) return;
    setBusy(true);
    setError('');
    try {
      const result = await operation();
      if (result.code !== 'OK') {
        setError(equipmentMessage(result.code));
        return;
      }
      if (result.data?.groups) s.setGroups(result.data.groups);
      s.reload((n) => n + 1);
      setEdit(undefined);
      setChecked([]);
    } catch {
      setError('Não foi possível concluir. Tente novamente.');
    } finally {
      setBusy(false);
    }
  };
  const create = (ancestor: string) =>
    setEdit({ mode: 'create', id: id(), parent: ancestor, name: '' });
  const save = () => {
    if (!edit) return;
    const op = id();
    void run(() =>
      edit.mode === 'create'
        ? equipmentAPI.createGroup(
            p.profile,
            p.activation,
            op,
            edit.id,
            edit.parent,
            edit.name,
          )
        : equipmentAPI.renameGroup(
            p.profile,
            p.activation,
            op,
            edit.id,
            edit.name,
          ),
    );
  };
  const move = () =>
    void run(() =>
      equipmentAPI.bulk(p.profile, p.activation, id(), {
        items: devices
          .filter((d) => checked.includes(d.id))
          .map(({ id, revision }) => ({ id, revision })),
        group_id: destination,
      }),
    );
  const remove = () =>
    void run(() =>
      equipmentAPI.deleteGroup(p.profile, p.activation, id(), selected),
    );
  return {
    s,
    selected,
    select: (id: string) => {
      setSelected(id);
      setChecked([]);
    },
    checked,
    setChecked,
    destination,
    setDestination,
    edit,
    setEdit,
    error,
    blocked,
    ...selection,
    create,
    save,
    move,
    remove,
  };
}
export type GroupsModel = ReturnType<typeof useEquipmentGroups>;

function groupSelection(s: GroupsProps['s'], selected: string) {
  const count = (id: string) =>
    s.devices.filter(
      (d) => id === '*' || groupContains(s.groups, id, d.group_id || ''),
    ).length;
  const devices = s.devices.filter(
    (d) =>
      selected === '*' || groupContains(s.groups, selected, d.group_id || ''),
  );
  const real = s.groups.find((g) => g.id === selected);
  return {
    count,
    devices,
    real,
    occupied:
      !!count(selected) || s.groups.some((g) => g.parent_id === selected),
  };
}
