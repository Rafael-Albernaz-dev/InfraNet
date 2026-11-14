import { useState } from 'react';
import { Button } from '../../shared/ui/Button';
import { equipmentMessage } from '../equipment/equipment-api';
import type { EnlaceView } from '../equipment/radio-api';
import type { RadioLinkPanelProps } from './radio-link-props';
import type { useEnlaceSession } from './enlace-session-state';
import { StoredRadioLinkPanel } from './stored-radio-link-panel';
export function EnlaceSessionContent(p: {
  state: ReturnType<typeof useEnlaceSession>;
  panel: RadioLinkPanelProps;
  link: () => void;
}) {
  const v = p.state.view;
  if (!v) return null;
  switch (v.state) {
    case 'offline':
      return (
        <p>
          Sem leitura: sessão OFFLINE. O vínculo salvo continua; a leitura volta
          quando o cliente reconectar.
        </p>
      );
    case 'unlinked':
      return (
        <EnlaceUnlinked view={v} blocked={p.panel.blocked} link={p.link} />
      );
    case 'two':
      return (
        <EnlaceChoices
          view={v}
          confirm={p.state.confirm}
          blocked={p.panel.blocked}
        />
      );
    case 'ready':
      return (
        <>
          <StoredRadioLinkPanel {...p.panel} session enlace={v} />
          <p className="muted">
            A CPE é lida em ronda leve a cada 15–30 min e por completo ao abrir
            este cliente.
          </p>
        </>
      );
    case 'none':
      return (
        <p>MAC não encontrado nos APs ligados. Confira o vínculo manual.</p>
      );
    case 'unread':
      return <p>Sem leitura dos APs ligados.</p>;
    case 'failed':
      return (
        <p role="alert">
          {v.reason === 'LIMIT_EXCEEDED'
            ? 'Limite de 5.000 equipamentos'
            : equipmentMessage(v.reason || '')}
        </p>
      );
    default:
      return <EnlaceStages view={v} />;
  }
}
function EnlaceUnlinked(p: {
  view: EnlaceView;
  blocked: boolean;
  link: () => void;
}) {
  return (
    <>
      <p>
        A VLAN deste cliente,{' '}
        <b>{p.view.interface || 'Não informado nesta leitura'}</b>, ainda não
        está ligada a um AP.
      </p>
      <div className="actions">
        <Button
          kind="primary"
          disabled={p.blocked || !p.view.interface}
          onClick={p.link}
        >
          Ligar esta VLAN a um AP
        </Button>
      </div>
      <p className="muted">
        Depois disso o app entra no AP, acha este cliente pelo MAC e cria a CPE
        sozinho.
      </p>
    </>
  );
}
function EnlaceChoices(p: {
  view: EnlaceView;
  confirm: (ap: string) => Promise<void>;
  blocked: boolean;
}) {
  const [selected, setSelected] = useState('');
  return (
    <>
      <p>
        O MAC deste cliente apareceu em dois APs ligados a{' '}
        <b>{p.view.interface}</b>. Escolha o certo:
      </p>
      <fieldset>
        <legend>AP do cliente</legend>
        {p.view.choices?.map((c) => (
          <label className="campo" key={c.ap_id}>
            <input
              type="radio"
              name="enlace-ap"
              checked={selected === c.ap_id}
              onChange={() => setSelected(c.ap_id)}
            />
            {c.name}{' '}
            <span className="muted">
              · sinal {c.signal || 'Não informado nesta leitura'}
            </span>
          </label>
        ))}
      </fieldset>
      <div className="actions">
        <Button
          kind="primary"
          disabled={p.blocked || !selected}
          onClick={() => void p.confirm(selected)}
        >
          Confirmar AP
        </Button>
      </div>
      <p className="muted">
        Depois de confirmar, a CPE é criada e lida sozinha.
      </p>
    </>
  );
}
function EnlaceStages({ view: v }: { view: EnlaceView }) {
  const stages = [
    'Entrando no AP com a credencial dele',
    'Lendo os clientes do AP',
    'MAC achado no AP',
    `Criando a CPE e entrando nela pelo IP da sessão (${v.address}) com a credencial padrão de CPE`,
    'Lendo sinal, modelo e versão da CPE',
  ];
  return (
    <>
      <p>
        Montando o enlace deste cliente pela VLAN <b>{v.interface}</b>:
      </p>
      <ol className="fases">
        {stages.map((t, i) => (
          <li key={t} className={stageClass(i, v.stage)}>
            {i < v.stage ? '✓ ' : ''}
            {i === v.stage ? <span className="spin">{t}…</span> : t}
          </li>
        ))}
      </ol>
    </>
  );
}
function stageClass(index: number, current: number) {
  if (index < current) return 'feito';
  if (index === current) return 'agora';
  return undefined;
}
