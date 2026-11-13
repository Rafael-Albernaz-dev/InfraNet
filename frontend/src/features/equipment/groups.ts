export type Group = { id: string; parent_id: string; name: string };

export function groupPath(groups: Group[], id: string): string {
  const parts: string[] = [];
  for (let depth = 0; id && depth < 4; depth++) {
    const group = groups.find((g) => g.id === id);
    if (!group) return '';
    parts.unshift(group.name);
    id = group.parent_id;
  }
  return parts.join(' › ');
}
export function groupContains(
  groups: Group[],
  ancestor: string,
  id: string,
): boolean {
  if (!ancestor) return !id;
  for (let depth = 0; depth < 4; depth++) {
    if (ancestor === id) return true;
    const group = groups.find((g) => g.id === id);
    if (!group) return false;
    id = group.parent_id;
  }
  return false;
}
export function groupOptions(groups: Group[]) {
  return groups
    .map((g) => ({ value: g.id, label: groupPath(groups, g.id) }))
    .sort((a, b) => a.label.localeCompare(b.label));
}
