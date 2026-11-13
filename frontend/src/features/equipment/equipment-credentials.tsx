import type { TableFilter } from '../../shared/ui/table-model';
import { useCredentialUsage } from './credential-usage';
import type { useEquipmentState } from './equipment-state';
import { DefaultCredentialDetails } from './equipment-credential-form';
import { Button } from '../../shared/ui/Button';
import { DataTable } from '../../shared/ui/DataTable';
import { RadioRoles } from './radio-contract';

export function EquipmentCredentials({
  s,
  profile,
  activation,
  locked,
}: {
  s: ReturnType<typeof useEquipmentState>;
  profile: string;
  activation: string;
  locked: boolean;
}) {
  const usage = useCredentialUsage(profile, activation);
  const rows = s.credentials;
  const filters: TableFilter[] = [
    {
      id: 'default',
      label: 'Padrão em',
      options: [RadioRoles.ap, RadioRoles.cpe, RadioRoles.ptp].map((value) => ({
        value,
        label: value.toUpperCase(),
      })),
      matches: (r, v) => s.credentialDefaults[v] === r.props['data-row-id'],
    },
    {
      id: 'use',
      label: 'Uso',
      options: [
        { value: 'used', label: 'Em uso' },
        { value: 'unused', label: 'Sem uso' },
      ],
      matches: (r, v) =>
        v === 'used'
          ? Number(r.props['data-links']) > 0
          : Number(r.props['data-links']) === 0,
    },
  ];
  return (
    <>
      <h1>Credenciais</h1>
      <div className="toolbar">
        <span className="grow" />
        <Button
          kind="primary"
          disabled={locked || !s.vault}
          onClick={() => s.setCredentialForm('new')}
        >
          Adicionar credencial
        </Button>
      </div>
      <DataTable tableId="credentials" filters={filters}>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Usuário</th>
            <th>Usada por</th>
            <th>Padrão em</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((credential) => {
            const names = usage?.[credential.id]?.join(', ') || '—';
            const defaults = Object.entries(s.credentialDefaults)
              .filter(([, value]) => value === credential.id)
              .map(([role]) => role.toUpperCase())
              .join(', ');
            return (
              <tr
                key={credential.id}
                data-row-id={credential.id}
                data-links={credential.links}
                className="click"
                tabIndex={0}
                onClick={() => s.setCredentialForm(credential)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    s.setCredentialForm(credential);
                  }
                }}
              >
                <td>{credential.alias}</td>
                <td>{credential.user}</td>
                <td>
                  {usage ? (
                    <span className="credential-usage" title={names}>
                      {names}
                    </span>
                  ) : (
                    'Não informado nesta leitura'
                  )}
                </td>
                <td>{defaults || '—'}</td>
              </tr>
            );
          })}
          {!rows.length && (
            <tr>
              <td colSpan={4}>Nenhuma credencial encontrada</td>
            </tr>
          )}
        </tbody>
      </DataTable>
      <div className="sp10" />
      <div className="settings-card">
        <DefaultCredentialDetails
          {...s}
          profile={profile}
          activation={activation}
          locked={locked}
        />
      </div>
    </>
  );
}
