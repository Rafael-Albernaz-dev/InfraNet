import { useState } from 'react';
import { useDiagnostic } from './pppoe-diag-state';
import type { Device } from '../equipment/equipment-api';
import { PingResultDetails } from './pppoe-ping-result';
import { Button } from '../../shared/ui/Button';

export function StoredSessionPing(p: {
  profile: string;
  activation: string;
  device: Device;
  observation: string;
  row: string;
}) {
  const [opened, setOpened] = useState(false);
  const view = useDiagnostic(p.profile, p.activation, p.device, 'ping', {
    observation: p.observation,
    row: p.row,
  });
  const result = view.rows[0]?.ping;
  return (
    <>
      <Button
        onClick={() => {
          setOpened(true);
          view.reopen();
        }}
      >
        Reabrir sondagem armazenada
      </Button>
      {opened && view.loading && (
        <p role="status">Carregando última sondagem armazenada…</p>
      )}
      {opened && view.error && <p role="alert">{view.error}</p>}
      {opened && result && view.meta && (
        <PingResultDetails
          result={result}
          metaDate={view.meta.provenance.ended_at}
        />
      )}
      {opened && !view.loading && !view.error && !view.meta && (
        <p>Nenhuma sondagem armazenada para esta sessão.</p>
      )}
    </>
  );
}
