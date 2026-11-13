import type { PPResult } from './pppoe-types';
import type { RadioResult } from '../../features/equipment/radio-api';
type Part = 'diagnostic_rows' | 'deltas' | 'read';
function pageIdentity(r: PPResult) {
  return [
    r.profile_id,
    r.activation_id,
    r.data?.diagnostic?.diagnostic_id,
    r.data?.read?.ended_at,
    r.data?.current?.observation_id,
  ].join('|');
}
function appendPPPage(
  data: NonNullable<PPResult['data']>,
  page: NonNullable<PPResult['data']>,
  part: Part,
) {
  if (part === 'read' && data.read && page.read)
    data.read.rows.push(...page.read.rows);
  else if (part === 'diagnostic_rows')
    data.diagnostic_rows = [
      ...(data.diagnostic_rows || []),
      ...(page.diagnostic_rows || []),
    ];
  else if (part === 'deltas')
    data.deltas = [...(data.deltas || []), ...(page.deltas || [])];
}
export async function completePPTable(
  first: PPResult,
  read: (cursor: string) => Promise<PPResult>,
  part: Part,
): Promise<PPResult> {
  if (first.code !== 'OK' || !first.data) return first;
  const data = { ...first.data };
  if (data.read) data.read = { ...data.read, rows: [...data.read.rows] };
  if (data.diagnostic_rows) data.diagnostic_rows = [...data.diagnostic_rows];
  if (data.deltas) data.deltas = [...data.deltas];
  let cursor = data.next_cursor || '';
  const seen = new Set<string>();
  while (cursor) {
    if (seen.has(cursor) || seen.size >= 256)
      throw new Error('invalid stored table cursor');
    seen.add(cursor);
    const page = await read(cursor);
    if (
      page.code !== 'OK' ||
      !page.data ||
      pageIdentity(page) !== pageIdentity(first)
    )
      throw new Error('stored table changed');
    appendPPPage(data, page.data, part);
    cursor = page.data.next_cursor || '';
  }
  return { ...first, data: { ...data, next_cursor: '' } };
}
export async function completeRadioTable(
  first: RadioResult,
  read: (id: string, offset: number) => Promise<RadioResult>,
): Promise<RadioResult> {
  if (
    first.code !== 'OK' ||
    !first.data?.reading ||
    first.data.job?.state === 'RUNNING'
  )
    return first;
  const data = {
    ...first.data,
    reading: {
      ...first.data.reading,
      clients: [...first.data.reading.clients],
    },
  };
  let page = first;
  const seen = new Set<number>();
  while (page.data?.more) {
    const offset = page.data.next_offset || 0;
    if (seen.has(offset) || seen.size >= 256)
      throw new Error('invalid radio table cursor');
    seen.add(offset);
    page = await read(data.reading.id, offset);
    if (!sameRadioTable(first, page, data.reading.id))
      throw new Error('radio table changed');
    data.reading.clients.push(...page.data.reading.clients);
  }
  return { ...first, data: { ...data, more: false, next_offset: 0 } };
}

function sameRadioTable(
  first: RadioResult,
  page: RadioResult,
  id: string,
): page is RadioResult & {
  data: NonNullable<RadioResult['data']> & {
    reading: NonNullable<NonNullable<RadioResult['data']>['reading']>;
  };
} {
  return (
    page.code === 'OK' &&
    page.profile_id === first.profile_id &&
    page.activation_id === first.activation_id &&
    page.data?.reading?.id === id
  );
}

export async function completeRadioIndex(
  first: RadioResult,
  read: (offset: number) => Promise<RadioResult>,
): Promise<RadioResult> {
  if (first.code !== 'OK' || !first.data?.index) return first;
  const index = {
    ...first.data.index,
    targets: { ...first.data.index.targets },
    labels: { ...first.data.index.labels },
  };
  let page = first;
  const seen = new Set<number>();
  while (page.data?.index?.more) {
    const offset = page.data.index.next_offset || 0;
    if (seen.has(offset) || seen.size >= 256)
      throw new Error('invalid index cursor');
    seen.add(offset);
    page = await read(offset);
    if (!sameRadioIndex(first, page)) throw new Error('index changed');
    Object.assign(index.targets, page.data!.index!.targets);
    Object.assign(index.labels, page.data!.index!.labels);
  }
  return {
    ...first,
    data: {
      ...first.data,
      index: { ...index, offset: 0, next_offset: 0, more: false },
    },
  };
}
function sameRadioIndex(first: RadioResult, page: RadioResult) {
  return (
    page.code === 'OK' &&
    page.profile_id === first.profile_id &&
    page.activation_id === first.activation_id &&
    !!page.data?.index &&
    page.data.index.observed_at === first.data?.index?.observed_at &&
    page.data.index.total === first.data?.index?.total
  );
}
