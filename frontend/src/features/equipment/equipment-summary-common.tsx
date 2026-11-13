import {
  readingUpdateState,
  type ReadingUpdateStatus,
} from './reading-update-state';
import type { ReactNode } from 'react';
import type { Device } from './equipment-api';
import { formatDate } from './network-format';
import { DataBlock } from '../../shared/ui/DataBlock';
import { KeyValues } from '../../shared/ui/KeyValues';
import { Button } from '../../shared/ui/Button';
export function SummaryAccess({
  device,
  credential,
  edit,
}: {
  device: Device;
  credential?: string;
  edit?: () => void;
}) {
  const accepted = device.trust_acceptance?.accepted_at;
  return (
    <DataBlock
      title="Acesso"
      source="cadastro local"
      data={{
        state: 'ready',
        content: (
          <>
            <KeyValues
              rows={[
                ['Porta SSH', String(device.port)],
                [
                  'Credencial',
                  credential ||
                    (device.credential_id ? 'Acesso salvo' : 'Sem credencial'),
                ],
                [
                  'Chave SSH',
                  device.fingerprint
                    ? 'Guardada sozinha' +
                      (accepted ? ' · ' + formatDate(accepted) : '')
                    : 'Ainda sem chave guardada',
                ],
              ]}
            />
            {edit && (
              <div className="actions">
                <Button onClick={edit}>Editar acesso</Button>
              </div>
            )}
          </>
        ),
      }}
    />
  );
}
export function SummaryPending({
  title,
  slice,
}: {
  title: string;
  slice: string;
}) {
  return (
    <DataBlock
      title={title}
      source="serviço local"
      data={{ state: 'unbuilt', slice, label: 'Não feito' }}
    />
  );
}
export function SummaryValues({
  title,
  source = 'SSH',
  age,
  rows,
  action,
  status,
}: {
  title: string;
  source?: string;
  age?: string;
  rows: [string, ReactNode][];
  action?: ReactNode;
  status?: ReadingUpdateStatus;
}) {
  return (
    <DataBlock
      title={title}
      source={source}
      age={age}
      data={readingUpdateState(
        <>
          <KeyValues rows={rows} />
          {action && <div className="actions">{action}</div>}
        </>,
        status,
      )}
    />
  );
}
