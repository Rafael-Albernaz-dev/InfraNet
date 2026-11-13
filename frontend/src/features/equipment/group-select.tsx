import { Select } from '../../shared/ui/Select';
import { groupOptions, type Group } from './groups';
export function GroupSelect(p: {
  groups: Group[];
  value: string;
  change: (id: string) => void;
  disabled?: boolean;
}) {
  return (
    <label className="field">
      Grupo
      <Select
        aria-label="Grupo"
        value={p.value}
        disabled={p.disabled}
        onChange={(e) => p.change(e.target.value)}
      >
        <option value="">Sem grupo</option>
        {groupOptions(p.groups).map((g) => (
          <option key={g.value} value={g.value}>
            {g.label}
          </option>
        ))}
      </Select>
    </label>
  );
}

export function GroupPathSelect(p: {
  groups: Group[];
  value: string;
  change: (path: string) => void;
}) {
  const options = groupOptions(p.groups).map((g) => g.label);
  if (p.value && !options.includes(p.value)) options.push(p.value);
  return (
    <label className="field">
      Grupo
      <Select
        aria-label="Grupo"
        value={p.value}
        onChange={(e) => p.change(e.target.value)}
      >
        <option value="">Sem grupo</option>
        {options.map((path) => (
          <option key={path} value={path}>
            {path}
          </option>
        ))}
      </Select>
    </label>
  );
}
