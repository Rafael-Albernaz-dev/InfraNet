import { Dialog } from '../../shared/ui/Dialog';
import { equipmentMessage } from './equipment-api';
import type { AddEquipmentProps } from './add-equipment';
import { AddEquipmentFields } from './add-equipment-fields';
import { Button } from '../../shared/ui/Button';
import { Field } from '../../shared/ui/Field';
import type { AddEquipmentModel } from './use-add-equipment-page';
type AddBodyProps = AddEquipmentProps & { model: AddEquipmentModel };

function SingleEnrollmentNotices(p: AddBodyProps) {
  const { enrollment, shownRow, singleError } = p.model;
  return (
    <>
      {enrollment.notice && <p role="status">{enrollment.notice}</p>}
      {enrollment.contact && (
        <Dialog
          title={`Não foi possível entrar em ${enrollment.contact.address}`}
          close={enrollment.dismissFailure}
        >
          <p>{equipmentMessage(enrollment.contact.code)}</p>
          <p>Cadastrar mesmo assim?</p>
          <div className="dialog-actions">
            <Button onClick={enrollment.dismissFailure}>Não</Button>
            <Button
              kind="primary"
              onClick={() => void enrollment.confirmFailure()}
            >
              Sim, cadastrar
            </Button>
          </div>
        </Dialog>
      )}
      {enrollment.retry && (
        <p className="inventory-notice">
          Para tentar com o mesmo cadastro, confira os dados
          {shownRow.credential === '__inline__'
            ? ' e digite a senha novamente.'
            : '.'}
        </p>
      )}
      {enrollment.needsSecret && (
        <p role="status">Digite a senha novamente para tentar salvar.</p>
      )}
      {singleError && (
        <p className="field-error" role="alert">
          {singleError}
        </p>
      )}
    </>
  );
}
function SingleInlineFields(p: AddBodyProps) {
  const {
    shownRow,
    validationAttrs,
    validationMessage,
    inlineUser,
    inlineSecret,
    setInlineUser,
    setInlineSecret,
    setSingleDirty,
    enrollment,
  } = p.model;
  return (
    <>
      {shownRow.credential === '__inline__' && (
        <div className="add-form-grid">
          <Field
            layout="form"
            label="Usuário"
            aria-label="Usuário"
            {...validationAttrs('user')}
            value={inlineUser}
            disabled={p.locked || p.model.enrollment.checking}
            maxLength={80}
            onChange={(event) => {
              setInlineUser(event.target.value);
              setSingleDirty(true);
              enrollment.changed();
            }}
          >
            {validationMessage('user')}
          </Field>
          <Field
            layout="form"
            label="Senha"
            aria-label="Senha"
            type="password"
            {...validationAttrs('secret')}
            value={inlineSecret}
            maxLength={1024}
            autoComplete="new-password"
            disabled={p.locked || !p.s.vault}
            onChange={(event) => {
              setInlineSecret(event.target.value);
              setSingleDirty(true);
              enrollment.changed();
            }}
          >
            {validationMessage('secret')}
          </Field>
        </div>
      )}
    </>
  );
}
function SingleFormActions(p: AddBodyProps) {
  const { back, shownRow, save } = p.model;
  return (
    <div className="actions">
      <Button onClick={back}>Cancelar</Button>
      <Button
        disabled={
          p.locked ||
          p.model.enrollment.checking ||
          (shownRow.credential === '__inline__' && !p.s.vault)
        }
        onClick={() => save(true)}
      >
        Salvar e adicionar outro
      </Button>
      <Button
        kind="primary"
        type="submit"
        disabled={
          p.locked ||
          p.model.enrollment.checking ||
          (shownRow.credential === '__inline__' && !p.s.vault)
        }
      >
        Salvar
      </Button>
    </div>
  );
}
export function SingleEquipmentForm(p: AddBodyProps) {
  if (p.model.enrollment.completed)
    return (
      <section className="note">
        <p role="status">{p.model.enrollment.notice}</p>
        <div className="actions">
          <Button kind="primary" onClick={p.back}>
            Ver na lista
          </Button>
          <Button onClick={p.model.enrollment.another}>Adicionar outro</Button>
        </div>
      </section>
    );
  const {
    validationRef,
    save,
    shownRow,
    validationAttrs,
    validationMessage,
    change,
  } = p.model;
  return (
    <form
      noValidate
      ref={validationRef}
      onSubmit={(event) => {
        event.preventDefault();
        save(false);
      }}
    >
      <SingleEnrollmentNotices {...p} />
      <AddEquipmentFields
        row={shownRow}
        credentials={p.s.credentials}
        blocks={p.s.blocks}
        groups={p.s.groups}
        credentialById
        disabled={p.locked || p.model.enrollment.checking}
        addCredential={() => p.s.setCredentialForm('new')}
        validation={{ attrs: validationAttrs, message: validationMessage }}
        change={change}
      />
      <SingleInlineFields {...p} />
      <p className="muted">
        O nome vem da identidade configurada no equipamento. Ao salvar, o app
        entra nele, guarda a chave SSH e já traz os dados.
      </p>
      <SingleFormActions {...p} />
    </form>
  );
}
