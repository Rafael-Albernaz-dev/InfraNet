import { useState } from 'react';
import { ContextCountsPolicyInputs } from './pppoe-context-table';
export function PPPServerPolicy({
  onChange,
}: {
  onChange: (threshold: number, minimum: number) => void;
}) {
  const [dropThreshold, setDropThreshold] = useState('50');
  const [minimumPeak, setMinimumPeak] = useState('5');
  const threshold = Number(dropThreshold),
    minimum = Number(minimumPeak);
  const validPolicy =
    dropThreshold.trim() !== '' &&
    minimumPeak.trim() !== '' &&
    Number.isFinite(threshold) &&
    threshold > 0 &&
    threshold <= 100 &&
    Number.isInteger(minimum) &&
    minimum >= 1 &&
    minimum <= 2048;
  return (
    <>
      <div className="bar">
        <ContextCountsPolicyInputs
          dropThreshold={dropThreshold}
          setDropThreshold={(v) => {
            setDropThreshold(v);
            onChange(Number(v), minimum);
          }}
          minimumPeak={minimumPeak}
          setMinimumPeak={(v) => {
            setMinimumPeak(v);
            onChange(threshold, Number(v));
          }}
          validPolicy={validPolicy}
        />
        <span className="muted">Pico entre amostras salvas.</span>
      </div>
      <p className="muted">
        APs ligados mostram os vínculos já salvos. Ligar AP: ainda não existe
        (L-ENLACE).
      </p>
    </>
  );
}
