import type { useImportState } from './add-import-state';
import type { Credential } from './equipment-api';
import { GroupPathSelect } from './group-select';
import type { Group } from './groups';
import { Select } from '../../shared/ui/Select';
type ImportFileProps = {
  state: ReturnType<typeof useImportState>;
  credentials: Credential[];
  groups: Group[];
};
export function ImportDefaults(p: ImportFileProps) {
  const s = p.state;
  return (
    <section className="import-card">
      <h2>Aplicar às linhas sem valor</h2>
      <GroupPathSelect groups={p.groups} value={s.block} change={s.setBlock} />
      <label className="field">
        Credencial
        <Select
          aria-label="Credencial"
          value={s.credential}
          onChange={(e) => s.setCredential(e.target.value)}
        >
          <option value="">Sem credencial</option>
          {p.credentials.map((c) => (
            <option key={c.id} value={c.alias}>
              {c.alias}
            </option>
          ))}
        </Select>
      </label>
    </section>
  );
}
