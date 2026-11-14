import { equipmentMessage } from './equipment-api';
import type { Credential } from './equipment-api';
import { Button } from '../../shared/ui/Button';
import { Select } from '../../shared/ui/Select';
import type { useInterfaceAPEnrollment } from './interface-ap-enrollment';
export function InterfaceAPCreate(p: {
  state: ReturnType<typeof useInterfaceAPEnrollment>;
  credentials: Credential[];
  defaultCredential: string;
  unlinked: boolean;
}) {
  const { draft, setDraft, enrollment, errors } = p.state;
  return (
    <details className="det" open={p.unlinked || undefined}>
      <summary>Cadastrar um AP novo aqui</summary>
      <label className="campo">
        IP do AP
        <input
          value={draft.address}
          onChange={(e) => setDraft({ ...draft, address: e.target.value })}
        />
      </label>
      <label className="campo">
        Porta SSH
        <input
          type="number"
          value={draft.port}
          onChange={(e) => setDraft({ ...draft, port: e.target.value })}
        />
      </label>
      <label className="campo">
        Credencial
        <Select
          value={draft.credential || p.defaultCredential}
          onChange={(e) => setDraft({ ...draft, credential: e.target.value })}
        >
          <option value="">Selecione</option>
          {p.credentials.map((c) => (
            <option key={c.id} value={c.id}>
              {c.alias} · {c.user}
            </option>
          ))}
        </Select>
      </label>
      <p className="muted">
        O app entra no AP, guarda a chave, lê a identidade e os clientes.
      </p>
      {Object.values(errors).map((e) => (
        <p role="alert" key={e}>
          {e}
        </p>
      ))}
      {enrollment.notice && <p role="status">{enrollment.notice}</p>}
      {enrollment.contact && (
        <p role="alert">
          {equipmentMessage(enrollment.contact.code)}{' '}
          <Button onClick={() => void enrollment.confirmFailure()}>
            Cadastrar mesmo assim
          </Button>
        </p>
      )}
      {enrollment.uncertain && (
        <Button
          disabled={enrollment.checking}
          onClick={() => void enrollment.reconcile()}
        >
          Conferir cadastro
        </Button>
      )}
    </details>
  );
}
