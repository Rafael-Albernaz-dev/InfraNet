import { ConfigOrigins } from './radio-contract';
import { Fragment } from 'react';
import { Button } from '../../shared/ui/Button';
import { DataBlock } from '../../shared/ui/DataBlock';
import type { DataState } from '../../shared/ui/DataBlock';
import { DataTable } from '../../shared/ui/DataTable';
import { Dialog } from '../../shared/ui/Dialog';
import { useConfigState, configSituation } from './config-state';
import type { ConfigVersion } from './config-types';
import { equipmentMessage } from './equipment-api';
import { formatDate } from './network-format';
import { readingAge } from './equipment-reading-format';
type State = NonNullable<ReturnType<typeof useConfigState>>;
function configData(s: State, content: React.ReactNode): DataState {
  if (s.loading) return { state: 'loading' };
  if (s.error)
    return {
      state: 'failed',
      reason: s.error,
      technical: s.error,
      retry: () => void s.load(),
      previous: content,
    };
  if (!s.view?.supported)
    return {
      state: 'empty',
      message: 'Este equipamento ainda não tem cópia de configuração',
    };
  return { state: 'ready', content };
}
function ConfigSave({ s, primary = false }: { s: State; primary?: boolean }) {
  return (
    <Button
      className={primary ? 'primary' : ''}
      disabled={s.blocked || s.working || s.view?.busy || !s.view?.supported}
      onClick={() => void s.save()}
    >
      Salvar configuração agora
    </Button>
  );
}
function ConfigFeedback({ s }: { s: State }) {
  const status = s.view?.status;
  let message = s.message;
  if (s.view?.busy) message = 'Lendo configuração…';
  else if (status?.code && status.code !== 'OK')
    message = equipmentMessage(status.code);
  else if (status?.unchanged)
    message = `Sem mudança desde ${formatDate(s.view!.versions[0].created_at)}. Nenhuma versão nova.`;
  return message ? (
    <p className="muted" role="status">
      {message}
    </p>
  ) : null;
}
export function ConfigSummary() {
  const s = useConfigState();
  if (!s) return null;
  const last = s.view?.versions[0];
  const content = (
    <>
      <dl className="kv">
        <dt>Última cópia</dt>
        <dd>
          {last
            ? `${formatDate(last.created_at)} · ${last.origin === ConfigOrigins.automatic ? 'automática' : 'manual'}`
            : 'Ainda sem cópia'}
        </dd>
        <dt>Situação</dt>
        <dd>{configSituation(s.view)}</dd>
        <dt>Versões guardadas</dt>
        <dd>{s.view?.versions.length || 0} de 30</dd>
      </dl>
      <ConfigFeedback s={s} />
      <div className="actions">
        <ConfigSave s={s} />
        <Button onClick={s.open}>Ver versões</Button>
      </div>
    </>
  );
  return (
    <DataBlock
      title="Configuração"
      source="cópia local"
      age={last ? readingAge(last.created_at) : ''}
      data={configData(s, content)}
    />
  );
}
function VersionActions({
  s,
  v,
  index,
}: {
  s: State;
  v: ConfigVersion;
  index: number;
}) {
  return (
    <>
      {index < (s.view?.versions.length || 0) - 1 && (
        <Button disabled={s.working} onClick={() => void s.compare(v.id)}>
          Comparar com a anterior
        </Button>
      )}{' '}
      <Button disabled={s.working} onClick={() => s.setDownload(v.id)}>
        Baixar
      </Button>
    </>
  );
}
function ConfigVersions({ s }: { s: State }) {
  const versions = s.view?.versions || [];
  return (
    <DataTable tableId="config-versions" aria-label="Versões da configuração">
      <thead>
        <tr>
          {['Data', 'Origem', 'Tamanho', 'Mudança', 'Ações'].map((x) => (
            <th key={x}>{x}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {versions.map((v, i) => (
          <tr key={v.id} data-row-id={v.id}>
            <td>{formatDate(v.created_at)}</td>
            <td>
              {v.origin === ConfigOrigins.automatic ? 'Automática' : 'Manual'}
            </td>
            <td>
              {(v.size / 1024).toLocaleString('pt-BR', {
                maximumFractionDigits: 1,
              })}{' '}
              KB
            </td>
            <td>
              {v.first ? 'Primeira cópia' : `+${v.added} −${v.removed} linhas`}
            </td>
            <td>
              <VersionActions s={s} v={v} index={i} />
            </td>
          </tr>
        ))}
      </tbody>
    </DataTable>
  );
}
function ConfigComparison({ s }: { s: State }) {
  if (!s.change) return null;
  return (
    <>
      <DataBlock
        title={comparisonTitle(s.change.from, s.change.to)}
        source="cópia local"
        age="agora"
        data={{
          state: 'ready',
          content: (
            <>
              <div className="diff" aria-label="Diferença entre versões">
                {s.change.lines.map((l, i) => (
                  <Fragment key={i}>
                    <span className={diffClass(l.kind)}>
                      {diffPrefix(l.kind)}
                      {l.text}
                    </span>
                    {'\n'}
                  </Fragment>
                ))}
              </div>
              {s.change.truncated && (
                <p>
                  {s.change.lines.length === 2000
                    ? 'Comparação cortada em 2.000 linhas'
                    : 'Comparação cortada no limite de tamanho.'}
                </p>
              )}
              <p className="muted">
                Vermelho saiu, verde entrou. Linhas com senha aparecem
                mascaradas na comparação.
              </p>
              <div className="actions">
                <Button onClick={() => s.setChange(undefined)}>
                  Fechar comparação
                </Button>
              </div>
            </>
          ),
        }}
      />
      <div className="sp10" />
    </>
  );
}
function ConfigDownload({ s }: { s: State }) {
  if (!s.download) return null;
  return (
    <Dialog title="Baixar configuração?" close={() => s.setDownload('')}>
      <p className="msg-erro">Este arquivo contém senhas do equipamento.</p>
      {s.error && <p role="alert">{s.error}</p>}
      <p>
        Guarde em lugar seguro. Para restaurar, importe pelo WinBox (
        <span className="mono">/import</span>). Restaurar pelo app chega na L13.
      </p>
      <div className="actions">
        <Button disabled={s.working} onClick={() => s.setDownload('')}>
          Cancelar
        </Button>
        <Button
          className="primary"
          disabled={s.working}
          onClick={() => void s.confirmDownload()}
        >
          Baixar arquivo
        </Button>
      </div>
    </Dialog>
  );
}
export function ConfigTab() {
  const s = useConfigState();
  if (!s) return null;
  const format =
    s.view?.format === 'cfg'
      ? 'system.cfg (airOS)'
      : '.rsc (export do RouterOS)';
  return (
    <>
      <ConfigComparison s={s} />
      <DataBlock
        title="Cópias da configuração"
        source="cópia local"
        age={
          s.view?.versions[0] ? readingAge(s.view.versions[0].created_at) : ''
        }
        data={configData(
          s,
          <>
            <div className="bar">
              <span className="muted">
                Formato {format}, com senhas, cifrado neste computador · cópia
                automática diária às 03:00, guardada só quando muda · até 30
                versões
              </span>
              <span className="grow" />
              <ConfigSave s={s} primary />
            </div>
            <ConfigFeedback s={s} />
            <ConfigVersions s={s} />
            <p className="muted">
              Restaurar pelo app chega na L13, com simulação antes. Até lá,
              baixe o arquivo e importe pelo WinBox (
              <span className="mono">/import</span>). O aviso de mudança pelo
              Telegram vem com a L09.
            </p>
          </>,
        )}
      />
      <ConfigDownload s={s} />
    </>
  );
}

function diffClass(kind: string) {
  if (kind === 'remove') return 'rem';
  if (kind === 'add') return 'add';
  return '';
}
function diffPrefix(kind: string) {
  if (kind === 'remove') return '- ';
  if (kind === 'add') return '+ ';
  return '  ';
}

function comparisonTitle(from?: string, to?: string) {
  const before = from ? formatDate(from) : 'anterior';
  const after = to ? formatDate(to) : 'atual';
  return `Mudanças de ${before} para ${after}`;
}
