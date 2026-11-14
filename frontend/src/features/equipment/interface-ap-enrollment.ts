import { useState } from 'react';
import { useAddEnrollment } from './add-enrollment';
import { equipmentMessage } from './equipment-api';
import { RadioRoles } from './radio-contract';
import type { InterfaceAPInput } from './interface-aps-state';
export function useInterfaceAPEnrollment(
  p: InterfaceAPInput & { created: (id: string) => void },
) {
  const [draft, setDraft] = useState({
    address: '',
    port: '22',
    credential: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [revision, bump] = useState(0);
  const enrollment = useAddEnrollment({
    profile: p.profile,
    activation: p.activation,
    reloadCredentials: bump,
    setErrors,
    saved: (_another, id) => {
      if (id) p.created(id);
    },
    perform: async (call) => {
      const r = await call();
      if (r.code !== 'OK') setErrors({ request: equipmentMessage(r.code) });
      return r;
    },
  });
  function save(defaultCredential: string) {
    return enrollment.save(
      {
        line: revision + 1,
        name: '',
        address: draft.address,
        port: draft.port,
        credential: draft.credential || defaultCredential,
        role: RadioRoles.ap,
        block: '',
      },
      false,
    );
  }
  return { draft, setDraft, errors, enrollment, save };
}
