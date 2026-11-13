import { useEffect, useState } from 'react';
import {
  pppoeAPI,
  type PPSessionHistory,
} from '../../platform/wails/presentation';
import type { Device } from '../equipment/equipment-api';
export type HistoryProps = {
  profile: string;
  activation: string;
  device: Device;
  login: string;
};
type HistoryPage = { rows: PPSessionHistory[]; next: string; error?: string };
function useHistoryPage(
  p: HistoryProps,
  reason: string,
  days: number,
  cursor: string,
  retry: number,
) {
  const [result, setResult] = useState<HistoryPage>();
  useEffect(() => {
    let alive = true;
    void pppoeAPI
      .history(
        p.profile,
        p.activation,
        p.device.id,
        p.login,
        reason,
        days,
        cursor,
      )
      .then((r) => {
        if (!alive) return;
        if (
          r.code !== 'OK' ||
          r.profile_id !== p.profile ||
          r.activation_id !== p.activation
        )
          setResult({
            rows: [],
            next: '',
            error: 'Falhou a leitura do histórico.',
          });
        else
          setResult({
            rows: r.data?.history || [],
            next: r.data?.next_cursor || '',
          });
      })
      .catch(() => {
        if (alive)
          setResult({
            rows: [],
            next: '',
            error: 'Falhou a leitura do histórico.',
          });
      });
    return () => {
      alive = false;
    };
  }, [
    p.profile,
    p.activation,
    p.device.id,
    p.login,
    reason,
    days,
    cursor,
    retry,
  ]);
  return { result, setResult };
}
function exportNotice(code: string) {
  if (code === 'OK') return 'Arquivo CSV salvo.';
  if (code === 'EXPORT_CANCELLED') return '';
  return 'Falhou a exportação.';
}
export function useSessionHistory(p: HistoryProps) {
  const [days, setDays] = useState(30),
    [reason, setReason] = useState(''),
    [cursor, setCursor] = useState('');
  const [notice, setNotice] = useState(''),
    [retry, bump] = useState(0);
  const { result, setResult } = useHistoryPage(p, reason, days, cursor, retry);
  const change = () => {
    setCursor('');
    setResult(undefined);
  };
  const exportCSV = async () => {
    try {
      setNotice(
        exportNotice(
          await pppoeAPI.exportHistory(
            p.profile,
            p.activation,
            p.device.id,
            p.login,
            reason,
            days,
          ),
        ),
      );
    } catch {
      setNotice('Falhou a exportação.');
    }
  };
  return {
    days,
    setDays,
    reason,
    setReason,
    cursor,
    setCursor,
    result,
    setResult,
    notice,
    change,
    exportCSV,
    retry: () => {
      change();
      bump((n) => n + 1);
    },
  };
}
export type HistoryState = ReturnType<typeof useSessionHistory>;
