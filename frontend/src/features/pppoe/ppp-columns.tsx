import type { PPPEntry, PPPScreen } from '../../platform/wails/ppp-catalog';
import { appliedSpeed, sessionPlan } from './pppoe-plan';
import { formatRate, formatDate } from '../equipment/network-format';
import { formatPPUptime } from '../equipment/pppoe-format';
export const PPP_SCREENS: [PPPScreen, string][] = [
  ['sessions', 'Sessões'],
  ['servers', 'Servidores PPPoE'],
  ['secrets', 'Cadastros'],
  ['profiles', 'Planos'],
  ['l2tp', 'L2TP Secrets'],
];
type Column = { id: string; label: string; optional?: boolean };
const column = (id: string, label: string, optional = false): Column => ({
  id,
  label,
  optional,
});
export function pppColumns(screen: PPPScreen, all: boolean): Column[] {
  const cols: Record<PPPScreen, Column[]> = {
    sessions: [
      column('name', 'Usuário'),
      column('client', 'Cliente', true),
      column('address', 'IP'),
      column('profile', 'Plano'),
      column('download', 'Download'),
      column('upload', 'Upload'),
      column('uptime', 'Conectado há'),
      column('state', 'Estado'),
      column('vlan', 'VLAN'),
      column('service', 'Serviço'),
      column('mtu', 'MTU'),
      column('linked_device', 'CPE'),
      column('absent_end', 'Desconectou em', true),
      column('interface', 'Interface', true),
      column('aps', 'AP', true),
      column('caller_id', 'Caller ID', true),
      column('session_id', 'Session ID', true),
      column('duplicates', 'Duplicidade', true),
    ],
    servers: [
      column('name', 'Serviço'),
      column('interface', 'Interface'),
      column('vlan', 'VLAN'),
      column('online', 'Sessões online'),
      column('peak', 'Pico 7 dias'),
      column('drop', 'Queda'),
      column('aps', 'APs ligados'),
      column('state', 'Estado'),
      column('mtu', 'MTU máx.'),
      column('mru', 'MRU máx.'),
      column('profile', 'Perfil padrão'),
      column('authentication', 'Autenticação', true),
    ],
    secrets: [
      column('name', 'Login'),
      column('client', 'Cliente', true),
      column('profile', 'Plano'),
      column('registration', 'Cadastro'),
      column('state', 'Sessão'),
      column('last_logged_out', 'Último logout'),
      column('last_disconnect_reason', 'Motivo da última queda'),
      column('last_caller_id', 'Último caller ID'),
      column('service', 'Serviço', true),
      column('caller_id', 'Caller ID fixo', true),
      column('remote_address', 'Endereço fixo', true),
    ],
    profiles: [
      column('name', 'Plano'),
      column('rate', 'Velocidade'),
      column('registrations', 'Cadastros'),
      column('online', 'Online'),
      column('local_address', 'Endereço local'),
      column('remote_address', 'Endereço remoto'),
      column('only_one', 'Uma sessão por login'),
    ],
    l2tp: [
      column('name', 'Nome'),
      column('profile', 'Perfil'),
      column('local_address', 'Endereço local'),
      column('remote_address', 'Endereço remoto'),
      column('last_logged_out', 'Último logout'),
    ],
  };
  if (!all) return cols[screen];
  const before: Record<PPPScreen, string> = {
    sessions: 'vlan',
    servers: 'state',
    secrets: 'last_caller_id',
    profiles: 'local_address',
    l2tp: '',
  };
  const at = cols[screen].findIndex((c) => c.id === before[screen]);
  const index = at < 0 ? cols[screen].length : at;
  return [
    ...cols[screen].slice(0, index),
    column('concentrator', 'Concentrador'),
    ...cols[screen].slice(index),
  ];
}
export function pppValue(e: PPPEntry, key: string): string | number {
  if (key === 'last_logged_out')
    return e.last_logged_out || 'Não informado nesta leitura';
  if (key === 'aps') return e.aps?.join(', ') || '—';
  if (key === 'drop')
    return e.drop === undefined ? '—' : e.drop.toFixed(0) + '%';
  if (key === 'registration') return registration(e.disabled);
  if (e.session) {
    const value = sessionValue(e, key);
    if (value !== undefined) return value;
  }
  if (key === 'state') return entryState(e);
  const value = e[key as keyof PPPEntry];
  if (typeof value === 'number') return value;
  if (typeof value === 'string') return value || 'Não informado nesta leitura';
  return 'Não informado nesta leitura';
}

function registration(disabled: boolean | undefined) {
  if (disabled === undefined) return 'Não informado nesta leitura';
  return disabled ? 'Desativado' : 'Ativo';
}
function sessionValue(e: PPPEntry, key: string): string | number | undefined {
  const r = e.session;
  if (!r) return undefined;
  const values: Record<string, () => string | number> = {
    profile: () => sessionPlan(r.context) + ' · ' + appliedSpeed(r.context),
    address: () => r.address,
    mtu: () => r.mtu ?? 'Não informado nesta leitura',
    uptime: () =>
      r.state === 'OFFLINE' ? '—' : formatPPUptime(r.uptime, true),
    download: () => (r.state === 'OFFLINE' ? '—' : formatRate(r.download)),
    upload: () => (r.state === 'OFFLINE' ? '—' : formatRate(r.upload)),
    session_id: () => r.session_id,
    duplicates: () =>
      r.duplicates > 1 ? 'Com duplicidade' : 'Sem duplicidade',
    caller_id: () => r.caller_id,
    linked_device: () => r.linked_device_name || 'Sem vínculo',
    absent_end: () => (r.absent_end ? formatDate(r.absent_end) : '—'),
  };
  return values[key]?.();
}

function entryState(e: PPPEntry) {
  if (e.kind === 'server') {
    if (e.invalid || e.disabled) return 'Inválido';
    if (e.invalid === undefined || e.disabled === undefined)
      return 'Não informado nesta leitura';
    return 'Válido';
  }
  return e.state || '—';
}
