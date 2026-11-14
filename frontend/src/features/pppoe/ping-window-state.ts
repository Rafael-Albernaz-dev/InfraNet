import { useEffect, useRef, useState } from 'react';
import { pppoeReadAPI } from '../../platform/wails/presentation';
import type { PPReadResult, PPResult } from '../../platform/wails/presentation';
import { diagnosticMessage } from './pppoe-diagnostic-message';

export type PingWindowTarget = {
  profile: string;
  activation: string;
  deviceId: string;
  revision: number;
  observation: string;
  row: string;
};
export function usePingWindow(p: PingWindowTarget) {
  const [generation, setGeneration] = useState(0);
  const stopRequest = useRef<(() => Promise<PPResult>) | undefined>(undefined);
  const [state, setState] = useState<{
    result?: PPReadResult;
    error?: string;
    expired?: boolean;
  }>({});
  const { profile, activation, deviceId, revision, observation, row } = p;
  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const operation = crypto.randomUUID();
    let job = '';
    const stop = () =>
      job
        ? pppoeReadAPI.cancel(profile, activation, job)
        : pppoeReadAPI.cancelOperation(
            profile,
            activation,
            deviceId,
            operation,
          );
    stopRequest.current = stop;
    async function poll() {
      try {
        const response = await pppoeReadAPI.status(profile, activation, job);
        if (cancelled) return;
        if (response.code !== 'OK')
          throw new Error(diagnosticMessage(response.code));
        const status = response.data?.job;
        setState({
          result: response.data?.read,
          expired: status?.code === 'DEADLINE_EXCEEDED',
          error:
            status?.state === 'FAILED' && status.code !== 'DEADLINE_EXCEEDED'
              ? diagnosticMessage(status.code)
              : undefined,
        });
        if (status?.state === 'RUNNING')
          timer = setTimeout(() => void poll(), 500);
      } catch (error) {
        if (!cancelled) setState({ error: String(error) });
        await stop().catch(() => undefined);
      }
    }
    async function start() {
      try {
        const response = await pppoeReadAPI.start(
          profile,
          activation,
          deviceId,
          operation,
          'ping-window',
          observation,
          [row],
          revision,
        );
        job = response.data?.job?.id || '';
        if (cancelled) {
          await stop().catch(() => undefined);
          return;
        }
        if (response.code !== 'OK' || !job)
          throw new Error(diagnosticMessage(response.code));
        await poll();
      } catch (error) {
        if (!cancelled) setState({ error: String(error) });
        await stop().catch(() => undefined);
      }
    }
    void start();
    return () => {
      cancelled = true;
      clearTimeout(timer);
      void stop().catch(() => undefined);
    };
  }, [profile, activation, deviceId, revision, observation, row, generation]);
  return {
    ...state,
    stop: () => stopRequest.current?.(),
    resume: () => {
      setState({});
      setGeneration((v) => v + 1);
    },
  };
}
