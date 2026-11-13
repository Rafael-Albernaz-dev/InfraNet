import { useState } from 'react';
import { pppCatalogAPI } from '../../platform/wails/presentation';
import { Button } from '../../shared/ui/Button';
export function PPPExportActions(p: {
  profile: string;
  activation: string;
  counts?: Record<string, number>;
}) {
  const [busy, setBusy] = useState(false),
    [message, setMessage] = useState('');
  const save = async (format: 'csv' | 'json') => {
    setBusy(true);
    setMessage('');
    try {
      const code = await pppCatalogAPI.export(p.profile, p.activation, format);
      if (code === 'EXPORT_CANCELLED') return;
      setMessage(
        code === 'OK' ? 'Arquivo salvo.' : 'Não foi possível exportar: ' + code,
      );
    } catch {
      setMessage('Serviço indisponível. Tente novamente.');
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <div className="bar">
        {p.counts && (
          <b>
            <span className="st-on">{p.counts.ONLINE} ONLINE</span> ·{' '}
            <span className="st-off">{p.counts.OFFLINE} OFFLINE</span>
          </b>
        )}
        <span className="muted">
          Lista relida a cada 60 s. Download e Upload: média de 60 s para todos
          e, nas linhas desta página, o instante a cada 2 s.
        </span>
        <span className="grow" />
        <Button disabled={busy} onClick={() => void save('csv')}>
          Exportar CSV
        </Button>
        <Button disabled={busy} onClick={() => void save('json')}>
          Exportar JSON sanitizado
        </Button>
      </div>
      <p className="muted">
        O CSV contém dados identificáveis de assinantes. O JSON sanitizado não
        contém.
      </p>
      {message && <p role="status">{message}</p>}
    </>
  );
}
