import { describe, expect, it } from 'vitest';
import { groupTreeOptions } from './groups';

describe('groupTreeOptions', () => {
  it('lists groups in tree order with indented names instead of full paths', () => {
    const groups = [
      { id: 'p', parent_id: 't', name: 'Painel 04' },
      { id: 's', parent_id: '', name: 'Região Sul' },
      { id: 'n', parent_id: '', name: 'Região Norte' },
      { id: 't', parent_id: 'n', name: 'Torre Ribeirão' },
    ];
    expect(groupTreeOptions(groups)).toEqual([
      { value: 'n', label: 'Região Norte' },
      { value: 't', label: '  Torre Ribeirão' },
      { value: 'p', label: '    Painel 04' },
      { value: 's', label: 'Região Sul' },
    ]);
  });
});
