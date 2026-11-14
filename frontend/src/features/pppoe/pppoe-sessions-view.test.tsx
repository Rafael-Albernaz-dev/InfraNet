import { render, screen, within } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { PPPoESessionsTable } from './pppoe-sessions-view';
import { usePPPoEViewState } from './pppoe-view-state';
import { pppColumns } from './ppp-columns';
vi.mock('../../platform/wails/presentation', async () =>
  (await import('../../shared/testing/presentation-mock')).presentationMock(),
);
function SavedSessions() {
  const s = usePPPoEViewState('p', 'a', 'sessions');
  return (
    <PPPoESessionsTable
      rows={[]}
      onDetail={() => {}}
      control={{
        ...s,
        change: () => {},
      }}
    />
  );
}
it('shows Cliente after Usuário and the wireframe column order before any preference is saved', () => {
  const expected = [
    'Usuário',
    'Cliente',
    'IP',
    'Plano',
    'Download',
    'Upload',
    'Conectado há',
    'Estado',
    'VLAN',
    'Serviço',
    'MTU',
    'CPE',
  ];
  const direct = render(<PPPoESessionsTable rows={[]} onDetail={() => {}} />);
  const headers = () =>
    within(screen.getByRole('table', { name: 'Sessões PPPoE' }))
      .getAllByRole('columnheader')
      .map((h) => h.textContent);
  expect(headers()).toEqual(expected);
  direct.unmount();
  render(<SavedSessions />);
  expect(headers()).toEqual(expected);
  const menu = [...expected];
  menu.splice(8, 0, 'Concentrador');
  expect(
    pppColumns('sessions', true)
      .filter((c) => !c.optional)
      .map((c) => c.label),
  ).toEqual(menu);
});
