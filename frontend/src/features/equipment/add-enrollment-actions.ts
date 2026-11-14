import { registrationOutcome } from './registration-actions';
import type { EquipmentResult } from './equipment-api';
import { equipmentAPI } from './equipment-api';
import { id } from './equipment-reading-format';
import { pendingEnrollments } from '../access/equipment-receipts';
import { fieldForEquipmentCode, fieldMessages } from './equipment-validation';
import type { ImportRow } from '../../platform/wails/presentation';
import { RadioTransports } from './radio-contract';

import type { EnrollmentProps, EnrollmentState } from './add-enrollment';

type EnrollmentInput = {
  row: ImportRow;
  another: boolean;
  user: string;
  secret: string;
};
function enrollmentRequest(
  s: EnrollmentState,
  { row, user, secret }: EnrollmentInput,
) {
  return {
    id: s.ids.device,
    operation_id: s.ids.operation,
    revision: 0,
    name: '',
    group_id: row.group_id || '',
    address: row.address.trim(),
    port: Number(row.port),
    role: row.role,
    block: row.block,
    transport: RadioTransports.ssh,
    credential_id: row.credential === '__inline__' ? '' : row.credential,
    user: row.credential === '__inline__' ? user.trim() : '',
    secret: row.credential === '__inline__' ? secret : '',
  };
}
function acceptEnrollment(
  p: EnrollmentProps,
  s: EnrollmentState,
  result: EquipmentResult | undefined,
  input: EnrollmentInput,
) {
  const unknown =
    !result ||
    ['SERVICE_UNAVAILABLE', 'DEADLINE_EXCEEDED', 'REVISION_CONFLICT'].includes(
      result.code,
    );
  if (result && fieldForEquipmentCode[result.code]) {
    const field = fieldForEquipmentCode[result.code];
    p.setErrors({ [field]: fieldMessages[field] });
  }
  if (!unknown) pendingEnrollments.delete(s.key);
  if (!s.alive.current) return;
  s.setNeedsSecret(Boolean(input.secret) && result?.code !== 'OK' && !unknown);
  s.setUncertain(unknown);
  if (result?.code === 'OK') {
    p.reloadCredentials((n) => n + 1);
    s.setRetry(false);
    s.setIDs({ device: id(), operation: id() });
    p.saved(input.another, result.data?.device?.id || s.ids.device);
  }
}
export async function saveEnrollment(
  p: EnrollmentProps,
  s: EnrollmentState,
  input: EnrollmentInput,
) {
  if (s.lock.current || s.uncertain) return;
  s.lock.current = true;
  pendingEnrollments.set(s.key, {
    id: s.ids.device,
    operation_id: s.ids.operation,
  });
  const request = enrollmentRequest(s, input);
  s.contactAddress.current = request.address;
  s.setChecking(true);
  s.setNotice(`Conectando em ${request.address}…`);
  try {
    let result = await p.perform(
      () => equipmentAPI.save(p.profile, p.activation, request),
      undefined,
      true,
      true,
    );
    if (result?.code === 'OK') result = await registrationOutcome(p, s, result);
    if (result?.code !== 'CONTACT_FAILED') {
      acceptEnrollment(p, s, result, input);
      if (result?.code === 'OK') s.setCompleted(!input.another);
    } else {
      pendingEnrollments.delete(s.key);
      s.setNotice('');
    }
  } catch {
    acceptEnrollment(p, s, undefined, input);
  } finally {
    request.secret = '';
    s.lock.current = false;
    if (s.alive.current) s.setChecking(false);
  }
}

export async function reconcileEnrollment(
  p: EnrollmentProps,
  s: EnrollmentState,
) {
  if (s.lock.current) return;
  s.lock.current = true;
  s.setChecking(true);
  try {
    const result = await p.perform(
      () =>
        equipmentAPI.enrollmentStatus(
          p.profile,
          p.activation,
          s.ids.device,
          s.ids.operation,
        ),
      undefined,
      false,
    );
    if (result?.code === 'OK') await acceptReconciledEnrollment(p, s, result);
  } finally {
    if (s.alive.current) s.setChecking(false);
    s.lock.current = false;
  }
}
async function acceptReconciledEnrollment(
  p: EnrollmentProps,
  s: EnrollmentState,
  result: EquipmentResult,
) {
  if (result.data?.job?.kind === 'inventory.register') {
    await recoverRegistrationJob(p, s, result);
    return;
  }
  if (result.data?.operation_state === 'COMMITTED') {
    if (result.data.device)
      s.setNotice(`Cadastrado: ${result.data.device.name}`);
    completeReconciledEnrollment(p, s);
  } else if (result.data?.operation_state === 'NOT_FOUND' && s.alive.current) {
    s.setRetry(true);
    s.setUncertain(false);
  }
}
function completeReconciledEnrollment(p: EnrollmentProps, s: EnrollmentState) {
  pendingEnrollments.delete(s.key);
  if (!s.alive.current) return;
  s.setUncertain(false);
  s.setCompleted(true);
  p.reloadCredentials((n) => n + 1);
  p.saved(false, s.ids.device);
}

async function recoverRegistrationJob(
  p: EnrollmentProps,
  s: EnrollmentState,
  result: EquipmentResult,
) {
  try {
    const outcome = await registrationOutcome(p, s, result);
    if (outcome?.code === 'CONTACT_FAILED') s.setUncertain(false);
    if (outcome?.data?.job?.state === 'SUCCEEDED')
      completeReconciledEnrollment(p, s);
  } catch {
    s.setUncertain(true);
  }
}
