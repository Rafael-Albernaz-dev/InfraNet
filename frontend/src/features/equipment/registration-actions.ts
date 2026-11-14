import { equipmentAPI } from './equipment-api';
import type { Device, EquipmentResult } from './equipment-api';
import type { EnrollmentProps, EnrollmentState } from './add-enrollment';

function registrationNotice(device: Device) {
  const model = device.result?.board || 'Modelo não informado';
  const version = device.result?.version || 'Versão não informada';
  return `Cadastrado: ${device.name} · ${model} · ${version}`;
}
export async function registrationOutcome(
  p: EnrollmentProps,
  s: EnrollmentState,
  initial: EquipmentResult,
) {
  const job = initial.data?.job;
  if (!job || job.kind !== 'inventory.register') return initial;
  let result = initial;
  while (result.data?.job?.state === 'RUNNING') {
    await new Promise((resolve) => setTimeout(resolve, 250));
    if (!s.alive.current) return undefined;
    result = await equipmentAPI.registrationStatus(
      p.profile,
      p.activation,
      job.id,
    );
    if (result.code !== 'OK') return result;
  }
  if (result.data?.job?.state === 'FAILED') {
    s.setContact({
      job: job.id,
      address: s.contactAddress.current,
      code: result.data.job.code,
    });
    return { ...result, code: 'CONTACT_FAILED' };
  }
  if (result.data?.device) s.setNotice(registrationNotice(result.data.device));
  return result;
}
export async function confirmFailedRegistration(
  p: EnrollmentProps,
  s: EnrollmentState,
) {
  if (!s.contact || s.lock.current) return;
  s.lock.current = true;
  try {
    const result = await p.perform(() =>
      equipmentAPI.confirmRegistration(p.profile, p.activation, s.contact!.job),
    );
    if (result?.code !== 'OK' || !s.alive.current) return;
    s.setContact(undefined);
    s.setNotice(
      `Cadastrado com aviso: ${s.contact.code}. O nome fica ${s.contact.address} até a primeira leitura.`,
    );
    s.setCompleted(true);
    p.saved(false, s.ids.device);
  } finally {
    s.lock.current = false;
  }
}
