import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { EquipmentSummaryStatistics } from './equipment-summary-statistics';
import {
  deviceMonitoringAPI,
  pppCatalogAPI,
} from '../../platform/wails/presentation';
import type { Device } from './equipment-api';
vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock({}),
);
const device = { id: 'synthetic', revision: 1 } as Device;
it('reabre projeção local após erro e não converte ausência de reinícios em zero', async () => {
  vi.mocked(pppCatalogAPI.summary)
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValue({
      code: 'OK',
      profile_id: 'p',
      activation_id: 'a',
      data: {
        total: 0,
        summary: {
          online: 2,
          offline: 1,
          drops_today: 1,
          plans: 1,
          servers: 1,
          invalid_servers: 0,
          falling_servers: 0,
          unlinked_servers: 1,
        },
      },
    });
  vi.mocked(deviceMonitoringAPI.current).mockResolvedValue({
    code: 'OK',
    profile_id: 'p',
    activation_id: 'a',
    data: { total: 0 },
  });
  const view = render(
    <EquipmentSummaryStatistics profile="p" activation="a" device={device} />,
  );
  const clients = screen.getByText('Clientes').closest('.block') as HTMLElement;
  await within(clients).findByRole('alert');
  expect(screen.queryByText('Nenhum reinício encontrado')).toBeNull();
  fireEvent.click(
    within(clients).getByRole('button', { name: 'Tentar novamente' }),
  );
  await screen.findByText('2');
  expect(screen.queryByRole('alert')).toBeNull();
  expect(
    screen.getAllByText('Não informado nesta leitura').length,
  ).toBeGreaterThan(0);
  view.rerender(
    <EquipmentSummaryStatistics
      profile="other"
      activation="b"
      device={device}
    />,
  );
  await waitFor(() => expect(screen.queryByText('2')).toBeNull());
  expect(await screen.findAllByRole('alert')).toHaveLength(3);
});
