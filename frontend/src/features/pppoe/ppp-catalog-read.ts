import { equipmentAPI, type Device } from '../equipment/equipment-api';
import { pppoeReadAPI } from '../../platform/wails/presentation';
import { shouldRefreshOnOpen } from '../equipment/use-refresh-on-open';
import { id } from '../equipment/equipment-reading-format';
export type CatalogReadInput = {
  profile: string;
  activation: string;
  devices: Device[];
  available: boolean;
};
export async function catalogDevice(
  p: CatalogReadInput,
  listed: Device,
  observedAt: string | undefined,
  manual: boolean,
) {
  const r = await equipmentAPI.get(p.profile, p.activation, listed.id);
  if (
    r.code !== 'OK' ||
    r.profile_id !== p.profile ||
    r.activation_id !== p.activation ||
    !r.data?.device
  )
    return;
  const device = r.data.device;
  const eligible = shouldRefreshOnOpen({
    now: Date.now(),
    observedAt: observedAt || new Date(0).toISOString(),
    device,
    vaultAvailable: p.available,
    serviceAvailable: p.available,
    reading: false,
  });
  if (!manual && !eligible) return;
  return device;
}
export async function readCatalog(
  p: CatalogReadInput,
  device: Device,
  alive: () => boolean,
) {
  const operation = id();
  let job = '';
  try {
    const started = await pppoeReadAPI.start(
      p.profile,
      p.activation,
      device.id,
      operation,
      'ppp-catalog',
      '',
      [],
      device.revision,
    );
    job = started.data?.job?.id || '';
    if (started.code !== 'OK' || !job) throw new Error(started.code);
    await pollCatalog(p, job, alive);
  } catch (reason) {
    if (!job)
      await pppoeReadAPI.cancelOperation(
        p.profile,
        p.activation,
        device.id,
        operation,
      );
    throw reason;
  }
}

async function pollCatalog(
  p: CatalogReadInput,
  job: string,
  alive: () => boolean,
) {
  const deadline = Date.now() + 30_000;
  while (alive() && Date.now() < deadline) {
    const result = await pppoeReadAPI.status(p.profile, p.activation, job);
    if (result.code !== 'OK') throw new Error(result.code);
    if (result.data?.job?.state !== 'RUNNING') {
      if (result.data?.job?.code !== 'OK')
        throw new Error(result.data?.job?.code || 'Leitura interrompida');
      return;
    }
    await new Promise<void>((resolve) => setTimeout(resolve, 250));
  }
  await pppoeReadAPI.cancel(p.profile, p.activation, job);
  throw new Error('Leitura interrompida');
}
export async function readCatalogDevices(
  p: CatalogReadInput,
  dates: Record<string, string>,
  manual: boolean,
  alive: () => boolean,
  publish: () => void,
) {
  const failures: string[] = [];
  for (const listed of p.devices.filter((d) => d.role === 'concentrator')) {
    if (!alive()) break;
    try {
      const device = await catalogDevice(p, listed, dates[listed.id], manual);
      if (!device || !alive()) continue;
      await readCatalog(p, device, alive);
      if (alive()) publish();
    } catch (reason) {
      failures.push(
        listed.name +
          ': ' +
          (reason instanceof Error ? reason.message : 'Falhou'),
      );
    }
  }
  return failures.join('; ');
}
