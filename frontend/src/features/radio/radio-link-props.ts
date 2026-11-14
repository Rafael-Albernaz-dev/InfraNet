import type { Device } from '../equipment/equipment-api';
export type RadioLinkPanelProps = {
  refreshOnOpen?: boolean;
  profile: string;
  activation: string;
  device: Device;
  observation: string;
  row: string;
  pppoeUptime: string;
  blocked: boolean;
  onOpenEquipment?: (deviceID: string) => void;
};
