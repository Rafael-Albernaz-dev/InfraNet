import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { pppoeReadAPI } from '../../platform/wails/presentation';
import type { PPResult } from '../../platform/wails/presentation';
import type { Device } from '../equipment/equipment-api';
import { SessionConnectivityTab } from './pppoe-session-info';

vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock({
    pppoeReadAPI: {
      cancel: vi.fn(async () => ({ code: 'OK' })),
      cancelOperation: vi.fn(async () => ({ code: 'OK' })),
    },
  }),
);

const props = {
  profile: 'p',
  activation: 'a',
  observation: 'o',
  row: 'r',
  address: '198.51.100.1',
  uptime: { known: true, min: 1, max: 2 },
  disabled: false,
  isOffline: false,
  device: {
    id: 'd',
    name: 'CCR sintético',
    active: true,
    revision: 1,
    fingerprint: 'SHA256:synthetic',
  } as Device,
};
beforeEach(() => vi.clearAllMocks());
it('streams one ping, closes on Fechar and cancels a start that finishes after unmount', async () => {
  vi.mocked(pppoeReadAPI.start).mockResolvedValue({
    code: 'OK',
    data: { job: { id: 'j', state: 'RUNNING' } },
  } as PPResult);
  vi.mocked(pppoeReadAPI.status).mockResolvedValue({
    code: 'OK',
    data: {
      job: { id: 'j', state: 'RUNNING' },
      read: {
        scope: 'ping-window',
        rows: [{ kind: 'ping-window', value: '0 198.51.100.1 56 64 3ms' }],
        ping_summary: 'sent=1 received=1 packet-loss=0%',
      },
    },
  } as PPResult);
  const view = render(<SessionConnectivityTab {...props} />);
  fireEvent.click(screen.getByRole('button', { name: 'Ping' }));
  expect(
    await screen.findByRole('dialog', {
      name: 'Ping pelo CCR sintético para 198.51.100.1',
    }),
  ).toBeVisible();
  await waitFor(() =>
    expect(screen.getByText(/sent=1 received=1/)).toHaveAttribute(
      'aria-live',
      'polite',
    ),
  );
  expect(pppoeReadAPI.start).toHaveBeenCalledTimes(1);
  expect(pppoeReadAPI.start).toHaveBeenCalledWith(
    'p',
    'a',
    'd',
    expect.any(String),
    'ping-window',
    'o',
    ['r'],
    1,
  );
  view.rerender(<SessionConnectivityTab {...props} isOffline />);
  expect(screen.getByRole('dialog')).toBeVisible();
  expect(pppoeReadAPI.start).toHaveBeenCalledTimes(1);
  view.rerender(
    <SessionConnectivityTab
      {...props}
      device={{ ...props.device, revision: 2 }}
    />,
  );
  expect(pppoeReadAPI.start).toHaveBeenCalledTimes(1);
  view.rerender(<SessionConnectivityTab {...props} />);
  vi.mocked(pppoeReadAPI.cancel).mockResolvedValueOnce({
    code: 'SERVICE_UNAVAILABLE',
    profile_id: 'p',
    activation_id: 'a',
  });
  fireEvent.click(screen.getByRole('button', { name: 'Fechar' }));
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Não foi possível confirmar o cancelamento',
  );
  expect(screen.getByRole('dialog')).toBeVisible();
  expect(pppoeReadAPI.start).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole('button', { name: 'Fechar' }));
  await waitFor(() =>
    expect(pppoeReadAPI.cancel).toHaveBeenCalledWith('p', 'a', 'j'),
  );
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  );
  let finish!: (result: PPResult) => void;
  vi.mocked(pppoeReadAPI.start).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Ping' }));
  await waitFor(() => expect(pppoeReadAPI.start).toHaveBeenCalledTimes(2));
  view.unmount();
  await act(async () =>
    finish({
      code: 'OK',
      data: { job: { id: 'late', state: 'RUNNING' } },
    } as PPResult),
  );
  expect(pppoeReadAPI.cancel).toHaveBeenCalledWith('p', 'a', 'late');
});
it('shows OFFLINE without starting SSH or exposing a ping action', () => {
  render(<SessionConnectivityTab {...props} isOffline />);
  expect(screen.getByText('Sem leitura: sessão OFFLINE')).toBeVisible();
  expect(
    screen.queryByRole('button', { name: 'Ping' }),
  ).not.toBeInTheDocument();
  expect(pppoeReadAPI.start).not.toHaveBeenCalled();
});
