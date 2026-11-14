export function EquipmentStateLabel({ value }: { value?: string }) {
  let color: string | undefined;
  if (value === 'ONLINE') color = 'st-on';
  if (value === 'OFFLINE') color = 'st-off';
  return (
    <span className={color}>{value || 'Não informado nesta leitura'}</span>
  );
}
