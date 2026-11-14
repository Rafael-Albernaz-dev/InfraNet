import { ConfigOrigins } from './radio-contract';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ConfigProvider } from './config-provider';
import { ConfigSummary, ConfigTab } from './config-backup';
import { configAPI } from '../../platform/wails/presentation';
import type { ConfigView } from './config-types';
vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock({}),
);
const history: ConfigView = {
  supported: true,
  format: 'rsc',
  busy: false,
  status: { at: '', code: 'OK', unchanged: false },
  versions: [
    {
      id: 'new',
      device_id: 'd',
      created_at: '2026-11-14T03:10:00Z',
      origin: ConfigOrigins.automatic,
      format: 'rsc',
      size: 2048,
      added: 1,
      removed: 1,
      first: false,
    },
    {
      id: 'old',
      device_id: 'd',
      created_at: '2026-11-13T03:10:00Z',
      origin: 'manual',
      format: 'rsc',
      size: 2048,
      added: -1,
      removed: 0,
      first: true,
    },
  ],
};
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(configAPI.list).mockResolvedValue({ code: 'OK', data: history });
});
function configScreen(source = '') {
  return (
    <ConfigProvider
      profile="p"
      activation="a"
      device="d"
      source={source}
      blocked={false}
      open={() => {}}
    >
      <ConfigTab />
      <ConfigSummary />
    </ConfigProvider>
  );
}
function open() {
  return render(configScreen());
}
it('compara versões após a primeira leitura sem perder a lista e permite fechar a comparação', async () => {
  vi.mocked(configAPI.list).mockResolvedValueOnce({
    code: 'OK',
    data: { ...history, supported: false, versions: [] },
  });
  vi.mocked(configAPI.compare).mockResolvedValue({
    code: 'OK',
    data: {
      ...history,
      change: {
        added: 1,
        removed: 1,
        truncated: false,
        lines: [
          { kind: 'remove', text: 'add name=ana password=********' },
          { kind: 'add', text: 'add name=ana password=********' },
        ],
      },
    },
  });
  const rendered = render(configScreen());
  await screen.findAllByText(
    'Este equipamento ainda não tem cópia de configuração',
  );
  rendered.rerender(configScreen('router-read-v6'));
  fireEvent.click(
    await screen.findByRole('button', { name: 'Comparar com a anterior' }),
  );
  const diff = await screen.findByLabelText('Diferença entre versões');
  expect(diff.querySelector('.rem')).toHaveTextContent(
    '- add name=ana password=********',
  );
  expect(diff.querySelector('.add')).toHaveTextContent(
    '+ add name=ana password=********',
  );
  expect(screen.getAllByRole('button', { name: 'Baixar' })).toHaveLength(2);
  expect(configAPI.compare).toHaveBeenCalledWith('p', 'a', 'd', 'new');
  fireEvent.click(screen.getByRole('button', { name: 'Fechar comparação' }));
  expect(
    screen.queryByLabelText('Diferença entre versões'),
  ).not.toBeInTheDocument();
});
it('só chama o diálogo nativo de baixar depois da confirmação e respeita cancelar', async () => {
  vi.mocked(configAPI.download).mockResolvedValue({ code: 'OK' });
  open();
  fireEvent.click(
    (await screen.findAllByRole('button', { name: 'Baixar' }))[0],
  );
  expect(
    await screen.findByRole('dialog', { name: 'Baixar configuração?' }),
  ).toBeVisible();
  expect(
    screen.getByText('Este arquivo contém senhas do equipamento.'),
  ).toBeVisible();
  expect(configAPI.download).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Cancelar' }));
  expect(configAPI.download).not.toHaveBeenCalled();
  fireEvent.click(screen.getAllByRole('button', { name: 'Baixar' })[0]);
  fireEvent.click(screen.getByRole('button', { name: 'Baixar arquivo' }));
  await waitFor(() =>
    expect(configAPI.download).toHaveBeenCalledWith('p', 'a', 'd', 'new'),
  );
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  );
  expect(screen.getAllByRole('button', { name: 'Baixar' })).toHaveLength(2);
});
