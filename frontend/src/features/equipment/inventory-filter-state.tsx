import { useState } from 'react';
import type { Group } from './groups';

export function useInventoryFilterState() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [stateFilter, setStateFilter] = useState('');
  const [pulseNetwork, setPulseNetwork] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [blockFilter, setBlockFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [blocks, setBlocks] = useState<string[]>([]);
  const [models, setModels] = useState<string[]>([]);
  return {
    groups,
    setGroups,
    stateFilter,
    setStateFilter,
    pulseNetwork,
    setPulseNetwork,
    roleFilter,
    setRoleFilter,
    blockFilter,
    setBlockFilter,
    modelFilter,
    setModelFilter,
    blocks,
    setBlocks,
    models,
    setModels,
  };
}
