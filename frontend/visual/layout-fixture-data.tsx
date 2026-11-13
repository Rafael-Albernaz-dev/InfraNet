export const base5Scenes = new Set([
  'equipamentos',
  'equipamentos-colunas',
  'equipamentos-selecao',
  'credenciais',
  'codigo',
  'credencial-editar',
  'credencial-excluir',
  'ficha-ccr-centro-sesso-es',
  'ficha-sw-norte-resumo',
  'seguranca-erro',
  'seguranca-codigo',
]);
export const sessions = Array.from({ length: 25 }, (_, i) => ({
  row_id: `row-${i}`,
  remote_id: `*${i}`,
  session_id: String(i),
  login: `cliente-${String(i + 1).padStart(4, '0')}`,
  address: `192.0.2.${201 + i}`,
  caller_id: `02:00:5E:00:00:${String(i + 1).padStart(2, '0')}`,
  state: (i + 1) % 7 === 0 ? 'OFFLINE' : 'ONLINE',
  uptime: {
    min: [2 * 86400 + 3 * 3600, 5 * 3600 + 12 * 60, 11 * 86400][i % 3] * 1e9,
    max: [2 * 86400 + 3 * 3600, 5 * 3600 + 12 * 60, 11 * 86400][i % 3] * 1e9,
    known: true,
  },
  duplicates: 1,
  download: { min: 78e6, max: 78e6 },
  upload: { min: 8e6, max: 8e6 },
  context: {
    association: 'associated',
    profile: ['Plano 100M', 'Plano 50M', 'Plano 200M'][i % 3],
    secret_state: 'active',
    vlan: 100,
    service: 'pppoe-0001',
    interface: 'ether1',
    profile_download_limit: ['100000000', '50000000', '200000000'][i % 3],
    profile_upload_limit: ['20000000', '10000000', '40000000'][i % 3],
    download_limit:
      i === 4 ? '1000000' : ['100000000', '50000000', '200000000'][i % 3],
    upload_limit:
      i === 4 ? '1000000' : ['20000000', '10000000', '40000000'][i % 3],
  },
}));
export const curve = [
  72,
  74,
  73,
  76,
  79,
  81,
  84,
  83,
  86,
  88,
  null,
  null,
  89,
  91,
  90,
  70,
  78,
  85,
  90,
  92,
  94,
  95,
  97,
  99,
];

export function liveSessionRows() {
  return sessions
    .filter((r) => r.state !== 'OFFLINE')
    .map((r) => ({
      kind: 'session',
      row_id: r.row_id,
      login: r.login,
      download: r.download,
      upload: r.upload,
    }));
}

export const layoutWave = (i: number, base: number, amp: number) =>
  +(
    base +
    amp *
      (0.55 * Math.sin(i / 2.3) +
        0.3 * Math.sin(i / 1.1 + 1) +
        0.15 * Math.sin(i * 1.7))
  ).toFixed(2);
