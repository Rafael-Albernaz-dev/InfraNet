export type ConfigVersion = {
  id: string;
  device_id: string;
  created_at: string;
  origin: string;
  format: string;
  size: number;
  added: number;
  removed: number;
  first: boolean;
};
export type ConfigChange = {
  from?: string;
  to?: string;
  lines: { kind: string; text: string }[];
  added: number;
  removed: number;
  truncated: boolean;
};
export type ConfigView = {
  versions: ConfigVersion[];
  supported: boolean;
  format: string;
  busy: boolean;
  status: { at: string; code: string; unchanged: boolean };
  change?: ConfigChange;
};
export type ConfigResult = { code: string; data?: ConfigView };
