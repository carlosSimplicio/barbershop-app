"use client";

import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type RefObject,
} from "react";
import {
  Button,
  Feedback,
  Modal,
  ModalActions,
  PasswordField,
} from "@/components/ui";
import { validatePassword } from "@/lib/auth/password-validation/password-validation";
import { TEMPORARY_PASSWORD_HINT } from "@/lib/professionals/professional-form-model";
import {
  reactivateProfessional,
  type Professional,
} from "@/lib/professionals/professionals-api";
import { classifyProfessionalOperation } from "@/lib/professionals/professional-operation-result";
import formStyles from "../create-professional-form/create-professional-form.module.css";

type Props = {
  returnFocusRef?: RefObject<HTMLElement | null>;
  fallbackFocusRef?: RefObject<HTMLElement | null>;
  record: Professional;
  canWrite: boolean;
  onClose: () => void;
  onRecordConfirmed: (record: Professional) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
};
type Notice = {
  variant: "success" | "error" | "warning";
  title: string;
  message: string;
};

export function ProfessionalReactivationPanel({
  record,
  returnFocusRef,
  fallbackFocusRef,
  canWrite,
  onClose,
  onRecordConfirmed,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
}: Props) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string>();
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  const [finished, setFinished] = useState(false);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const field = useRef<HTMLInputElement>(null);
  const focusError = useRef(false);
  useEffect(() => {
    if (!submitting && focusError.current) {
      focusError.current = false;
      field.current?.focus();
    }
  }, [submitting, error]);
  function showError() {
    setNotice({
      variant: "error",
      title: "Não foi possível reativar o profissional",
      message: "Tente novamente mais tarde.",
    });
  }
  async function run() {
    if (isPending() || !canWrite || finished || missing || record.active)
      return;
    const validation = validatePassword(
      password,
      "Informe uma senha temporária.",
    );
    setError(validation);
    if (validation) {
      field.current?.focus();
      return;
    }
    const request = begin();
    if (!request) return;
    try {
      const result = await reactivateProfessional(
        record.id,
        { temporary_password: password },
        request.signal,
      );
      if (request.signal.aborted) return;
      const outcome = classifyProfessionalOperation(result, 200);
      if (outcome.state === "unauthorized") {
        setPassword("");
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setMissing(true);
        setPassword("");
        setNotice({
          variant: "warning",
          title: "Profissional não encontrado",
          message: "O cadastro não está disponível. Atualize a lista.",
        });
        return;
      }
      if (outcome.state === "already_active") {
        setPassword("");
        setNotice({
          variant: "error",
          title: "Profissional já está ativo",
          message: "Esta tentativa não alterou a senha nem as sessões.",
        });
        return;
      }
      if (outcome.state === "read_only") {
        onReadOnly();
        setNotice({
          variant: "warning",
          title: "Reativação indisponível",
          message: "O estabelecimento está em modo somente leitura.",
        });
        return;
      }
      if (
        outcome.state === "confirmed" &&
        outcome.record.id === record.id &&
        outcome.record.active
      ) {
        onRecordConfirmed(outcome.record);
        setPassword("");
        setError(undefined);
        setFinished(outcome.record.active);
        setNotice({
          variant: "success",
          title: "Profissional reativado",
          message:
            "A troca de senha será obrigatória no próximo acesso. A senha temporária não poderá ser recuperada; compartilhe-a por um canal seguro.",
        });
        return;
      }
      if (outcome.state === "validation" && outcome.fields.temporary_password) {
        setError(outcome.fields.temporary_password);
        focusError.current = true;
        return;
      }
      showError();
    } catch {
      if (!request.signal.aborted) showError();
    } finally {
      request.finish();
    }
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void run();
  }
  const confirmationActions = (
    <ModalActions>
      {!finished && !missing ? (
        <Button
          type="submit"
          disabled={!canWrite || submitting || record.active}
        >
          {submitting ? "Aguarde…" : "Confirmar"}
        </Button>
      ) : null}
      <Button
        variant="secondary"
        disabled={submitting}
        onClick={() => {
          setPassword("");
          onClose();
        }}
      >
        {finished || missing ? "Fechar" : "Cancelar"}
      </Button>
    </ModalActions>
  );
  return (
    <Modal
      open
      title="Reativar profissional"
      onClose={onClose}
      returnFocusRef={returnFocusRef}
      fallbackFocusRef={fallbackFocusRef}
      closeDisabled={submitting}
    >
      <section
        aria-label="Reativar profissional"
        aria-busy={submitting}
        className={formStyles.panel}
      >
        <p>Reativar “{record.name}”?</p>
        <p>
          Defina uma senha temporária válida, inclusive igual à anterior. O
          profissional precisará trocá-la após entrar.
        </p>
        {notice ? (
          <Feedback variant={notice.variant} title={notice.title}>
            {notice.message}
          </Feedback>
        ) : null}
        {!finished && !missing ? (
          <form noValidate onSubmit={submit} className={formStyles.form}>
            <PasswordField
              ref={field}
              label="Senha temporária para reativação"
              autoComplete="new-password"
              hint={TEMPORARY_PASSWORD_HINT}
              value={password}
              error={error}
              disabled={!canWrite || submitting}
              required
              onChange={(event) => {
                setPassword(event.target.value);
                setError(undefined);
                setNotice(null);
              }}
            />
            {confirmationActions}
          </form>
        ) : (
          confirmationActions
        )}
      </section>
    </Modal>
  );
}
