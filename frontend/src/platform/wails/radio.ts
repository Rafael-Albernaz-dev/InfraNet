import { completeRadioIndex, completeRadioTable } from './stored-table-pages';
import {
  CancelRadio,
  CancelRadioOperation,
  ConfirmRadioFamily,
  ConfirmRadioLink,
  GetRadio,
  GetRadioIndexPage,
  GetRadioLink,
  RadioStatus,
  StartRadio,
} from '../../../wailsjs/go/presentation/App';
import type { equipment } from '../../../wailsjs/go/models';
import type {
  RadioResult,
  RadioInput,
  RadioLinkChoice,
} from '../../features/equipment/radio-api';
// Aridade fixada pelo método Go do binding; a tupla rotulada documenta cada posição.
type LinkChoiceArgs = [
  profile: string,
  activation: string,
  device: string,
  observation: string,
  row: string,
  operation: string,
  choice: RadioLinkChoice,
];
export const radioAPI = {
  indexPage: async (
    ...args: Parameters<typeof GetRadioIndexPage>
  ): Promise<RadioResult> =>
    completeRadioIndex(await GetRadioIndexPage(...args), (offset) =>
      GetRadioIndexPage(args[0], args[1], args[2], args[3], args[4], offset),
    ),
  current: (
    p: string,
    a: string,
    d: string,
    r = '',
    offset = 0,
  ): Promise<RadioResult> =>
    GetRadio(p, a, d, r, offset).then((first) =>
      completeRadioTable(first, (id, next) => GetRadio(p, a, d, id, next)).then(
        (full) =>
          completeRadioIndex(full, (next) => GetRadio(p, a, d, r, next)),
      ),
    ),
  start: (p: string, a: string, input: RadioInput): Promise<RadioResult> =>
    StartRadio(p, a, input as equipment.RadioInput),
  status: (
    p: string,
    a: string,
    j: string,
    r = '',
    offset = 0,
  ): Promise<RadioResult> =>
    RadioStatus(p, a, j, r, offset).then((first) =>
      completeRadioTable(first, (id, next) =>
        RadioStatus(p, a, j, id, next),
      ).then((full) =>
        completeRadioIndex(full, (next) => RadioStatus(p, a, j, r, next)),
      ),
    ),
  cancel: (p: string, a: string, j: string): Promise<RadioResult> =>
    CancelRadio(p, a, j),
  cancelOperation: (
    p: string,
    a: string,
    d: string,
    o: string,
  ): Promise<RadioResult> => CancelRadioOperation(p, a, d, o),
  confirmFamily: ConfirmRadioFamily as (
    ...args: Parameters<typeof ConfirmRadioFamily>
  ) => Promise<RadioResult>,
  link: (
    p: string,
    a: string,
    d: string,
    o: string,
    row: string,
  ): Promise<RadioResult> =>
    GetRadioLink(p, a, d, o, row).then((first) =>
      completeRadioIndex(first, (offset) =>
        GetRadioIndexPage(p, a, d, o, row, offset),
      ),
    ),
  confirmLink: (
    ...[p, a, d, o, row, operation, choice]: LinkChoiceArgs
  ): Promise<RadioResult> =>
    ConfirmRadioLink(
      p,
      a,
      d,
      o,
      row,
      operation,
      choice as equipment.RadioLinkChoice,
    ),
};
