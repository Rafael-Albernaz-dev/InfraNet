import { RadioModes } from '../equipment/radio-contract';
import { usePPReadState, type ReadState } from './pppoe-read-state';
import { ReadPanelBody } from './pppoe-read-panel-body';
import { SessionHistory } from './session-history';
import type { PPRow } from '../../platform/wails/presentation';
import type { Device } from '../equipment/equipment-api';
import { PPPoEReadPanel } from './pppoe-diagnostics';
import { formatPPUptime } from '../equipment/pppoe-format';
import { SessionHeader } from './pppoe-session-header';
import { SessionAuthInfo, SessionConnectivityTab } from './pppoe-session-info';
import type { PPPoESessionDetailProps } from './pppoe-session-props';
import type { PPObservation } from '../../platform/wails/pppoe-types';
import { SessionLinkPanel } from './session-link-slot';
import { SessionChain } from './session-chain';
import { DataBlock } from '../../shared/ui/DataBlock';
import { Tabs } from '../../shared/ui/Tabs';

type SessionInformationTabProps = {
  selected: PPRow;
  current: PPObservation;
  profile: string;
  activation: string;
  device: Device;
  disabled: boolean;
  fastTrack?: number;
  openEquipment?: (id: string) => void;
  beforePing?: () => Promise<void>;
  afterPing?: () => void;
};
function SessionInformationTab(p: SessionInformationTabProps) {
  const { selected, current, profile, activation, device, disabled } = p;
  const isOffline = selected.state === 'OFFLINE';
  return (
    <>
      <div className="session-grid">
        <SessionAuthInfo
          selected={selected}
          current={current}
          profile={profile}
          activation={activation}
          device={device}
          openEquipment={p.openEquipment}
          fastTrack={p.fastTrack}
        />
        <SessionConnectivityTab
          profile={profile}
          activation={activation}
          device={device}
          observation={current.observation_id}
          row={selected.row_id}
          address={selected.address}
          uptime={selected.uptime}
          disabled={disabled}
          isOffline={isOffline}
          beforePing={p.beforePing}
          afterPing={p.afterPing}
          openEquipment={p.openEquipment}
        />
      </div>
      <div className="session-grid">
        {(['log', 'secret'] as const).map((evidence) => (
          <PPPoEReadPanel
            key={evidence}
            profile={profile}
            activation={activation}
            device={device}
            observation={current.observation_id}
            rows={[]}
            logRow={selected.row_id}
            evidence={evidence}
            panel
            blocked={disabled || isOffline}
          />
        ))}
      </div>
    </>
  );
}
function SessionDetailNav({
  detailTab,
  setDetailTab,
}: {
  detailTab: 'traffic' | 'information' | 'link' | 'history';
  setDetailTab: (t: 'traffic' | 'information' | 'link' | 'history') => void;
}) {
  return (
    <Tabs
      label="Detalhes da sessão"
      items={[
        ['information', 'Resumo'],
        ['traffic', 'Tráfego'],
        ['link', 'Enlace'],
        ['history', 'Histórico'],
      ]}
      value={detailTab}
      onChange={setDetailTab}
    />
  );
}
function SessionTrafficTab(p: PPPoESessionDetailProps & { live: ReadState }) {
  if (p.selected.state === 'OFFLINE')
    return (
      <DataBlock
        title="Tráfego"
        source={`SSH (${p.device.name})`}
        data={{ state: 'ready', content: <p>Sem leitura: sessão OFFLINE</p> }}
      />
    );
  const props = {
    profile: p.profile,
    activation: p.activation,
    device: p.device,
    observation: p.current.observation_id,
    rows: [p.selected.row_id],
    traffic: true,
    panel: true,
    blocked: p.disabled,
  };
  return (
    <DataBlock
      title="Tráfego"
      source={`SSH (${p.device.name})`}
      data={{
        state: 'ready',
        content: <ReadPanelBody s={p.live} props={props} />,
      }}
    />
  );
}
function SessionDetailContent(
  p: PPPoESessionDetailProps & { live: ReadState },
) {
  const {
    selected,
    current,
    profile,
    activation,
    device,
    disabled,
    fastTrack,
    openEquipment,
  } = p;
  if (p.detailTab === 'history')
    return (
      <SessionHistory
        profile={profile}
        activation={activation}
        device={device}
        login={selected.login}
      />
    );
  if (p.detailTab === 'traffic') return <SessionTrafficTab {...p} />;
  if (p.detailTab === 'link')
    return (
      <SessionLinkPanel
        profile={profile}
        activation={activation}
        device={device}
        observation={current.observation_id}
        row={selected.row_id}
        pppoeUptime={formatPPUptime(selected.uptime)}
        blocked={disabled || selected.state === 'OFFLINE'}
        onOpenEquipment={openEquipment}
      />
    );
  return (
    <SessionInformationTab
      selected={selected}
      current={current}
      profile={profile}
      activation={activation}
      device={device}
      disabled={disabled}
      beforePing={p.live.cancel}
      afterPing={() => {
        if (!disabled && selected.state !== 'OFFLINE')
          void p.live.start(RadioModes.live);
      }}
      fastTrack={fastTrack}
      openEquipment={openEquipment}
    />
  );
}
export function PPPoESessionDetail(p: PPPoESessionDetailProps) {
  return (
    <SessionDetailLive
      key={[p.profile, p.activation, p.selected.row_id, p.detailTab].join('|')}
      {...p}
    />
  );
}
function SessionDetailLive(p: PPPoESessionDetailProps) {
  const {
    current,
    profile,
    activation,
    device,
    selected,
    openEquipment,
    detailTab,
    setDetailTab,
  } = p;
  const { setRoot, ...live } = usePPReadState({
    profile,
    activation,
    device,
    observation: current.observation_id,
    rows: [selected.row_id],
    traffic: detailTab === 'traffic',
    automaticLive: detailTab !== 'traffic',
    blocked: p.disabled || selected.state === 'OFFLINE',
  });
  return (
    <section
      ref={setRoot}
      className="session-screen"
      aria-label="Detalhes da sessão"
    >
      <SessionHeader
        {...p}
        sample={
          live.sampleAge < 6000 && !live.error
            ? live.result?.rows[0]
            : undefined
        }
      />
      <SessionChain
        profile={profile}
        activation={activation}
        device={device}
        observation={current.observation_id}
        row={selected.row_id}
        open={openEquipment}
      />
      <SessionDetailNav detailTab={detailTab} setDetailTab={setDetailTab} />
      <SessionDetailContent {...p} live={live} />
    </section>
  );
}
