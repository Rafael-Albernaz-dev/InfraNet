import { screen, within } from '@testing-library/react';

// The session route is visible before its first detail response.
export function waitForSessionReady() {
  return screen.findByRole('region', {
    name: 'Detalhes da sessão',
    busy: false,
  });
}

export function sessionTable() {
  const queries = {
    getByRole: (
      role: Parameters<typeof screen.getByRole>[0],
      options?: Parameters<typeof screen.getByRole>[1],
    ) =>
      within(screen.getByRole('group', { name: 'Sessões PPPoE' })).getByRole(
        role,
        options,
      ),
    findByRole: async (
      role: Parameters<typeof screen.findByRole>[0],
      options?: Parameters<typeof screen.findByRole>[1],
      wait?: Parameters<typeof screen.findByRole>[2],
    ) =>
      within(
        await screen.findByRole('group', { name: 'Sessões PPPoE' }),
      ).findByRole(role, options, wait),
    queryByRole: (
      role: Parameters<typeof screen.queryByRole>[0],
      options?: Parameters<typeof screen.queryByRole>[1],
    ) => {
      const group = screen.queryByRole('group', { name: 'Sessões PPPoE' });
      return group ? within(group).queryByRole(role, options) : null;
    },
  };
  return queries;
}

export function sessionSearch() {
  return sessionTable().getByRole('textbox', {
    name: 'Buscar em todas as colunas',
  });
}
