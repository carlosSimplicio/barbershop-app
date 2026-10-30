"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Feedback, PasswordField } from "@/components/ui";
import { validatePassword } from "@/lib/auth/password-validation/password-validation";
import { TEMPORARY_PASSWORD_HINT } from "@/lib/professionals/professional-form-model";
import {
  getProfessional,
  reactivateProfessional,
  type Professional,
} from "@/lib/professionals/professionals-api";
import { classifyProfessionalOperation } from "@/lib/professionals/professional-operation-result";
import formStyles from "../create-professional-form/create-professional-form.module.css";
import styles from "./professional-reactivation-panel.module.css";

type Props = {
  record: Professional;
  canWrite: boolean;
  onClose: () => void;
  onRecordConfirmed: (record: Professional) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
  onReconciliationChange: (needed: boolean) => void;
};
type Notice = {
  variant: "success" | "error" | "warning";
  title: string;
  message: string;
};

export function ProfessionalReactivationPanel({
  record,
  canWrite,
  onClose,
  onRecordConfirmed,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  onReconciliationChange,
}: Props) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);
  const [needsConsult, setNeedsConsult] = useState(false);
  const [finished, setFinished] = useState(false);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const pending = useRef(false);
  const controller = useRef<AbortController | null>(null);
  const panel = useRef<HTMLElement>(null);
  const field = useRef<HTMLInputElement>(null);
  const focusError = useRef(false);
  const conflict = useRef(false);
  useEffect(() => {
    panel.current?.focus();
    return () => controller.current?.abort();
  }, []);
  useEffect(() => {
    if (!submitting && focusError.current) {
      focusError.current = false;
      field.current?.focus();
    }
  }, [submitting, error]);
  function markUncertain() {
    conflict.current = false;
    setNeedsConsult(true);
    onReconciliationChange(true);
    setNotice({
      variant: "warning",
      title: "Não foi possível confirmar a reativação",
      message:
        "Consulte o cadastro antes de outra ação. A senha permanece somente neste formulário enquanto a consulta está pendente; a reativação não será repetida automaticamente.",
    });
  }
  async function run(read: boolean) {
    if (
      pending.current ||
      (!read &&
        (!canWrite || needsConsult || finished || missing || record.active))
    )
      return;
    if (!read) {
      conflict.current = false;
      const validation = validatePassword(
        password,
        "Informe uma senha temporária.",
      );
      setError(validation);
      if (validation) {
        field.current?.focus();
        return;
      }
    }
    pending.current = true;
    setSubmitting(true);
    onBusyChange(true);
    const request = new AbortController();
    controller.current = request;
    try {
      const result = read
        ? await getProfessional(record.id, request.signal)
        : await reactivateProfessional(
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
        setNeedsConsult(false);
        onReconciliationChange(false);
        setNotice({
          variant: "warning",
          title: "Profissional não encontrado",
          message: "O cadastro não está disponível. Atualize a lista.",
        });
        return;
      }
      if (outcome.state === "already_active" && !read) {
        conflict.current = true;
        setPassword("");
        setNeedsConsult(true);
        onReconciliationChange(true);
        setNotice({
          variant: "warning",
          title: "Profissional já está ativo",
          message:
            "Esta tentativa não alterou a senha nem as sessões. Consulte o cadastro; o acesso com a senha informada não foi confirmado.",
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
        (read || outcome.record.active)
      ) {
        onRecordConfirmed(outcome.record);
        setPassword("");
        setError(undefined);
        setNeedsConsult(false);
        onReconciliationChange(false);
        setFinished(outcome.record.active);
        if (read) {
          setNotice({
            variant: "warning",
            title: outcome.record.active
              ? "Cadastro ativo; senha não confirmada"
              : "Cadastro consultado",
            message: outcome.record.active
              ? conflict.current
                ? "O profissional está ativo. Esta tentativa não alterou sua senha. Confirme o acesso existente com o profissional."
                : "A consulta confirma somente o estado do cadastro e não informa qual senha foi aplicada. Confirme o acesso com o profissional; nenhuma reativação foi repetida."
              : "O profissional está inativo. A senha foi limpa; preencha-a se decidir tentar novamente.",
          });
        } else
          setNotice({
            variant: "success",
            title: "Profissional reativado",
            message:
              "A troca de senha será obrigatória no próximo acesso. A senha temporária não poderá ser recuperada; compartilhe-a por um canal seguro.",
          });
        return;
      }
      if (read) {
        setNotice({
          variant: "error",
          title: "Não foi possível consultar o cadastro",
          message: "Tente consultar novamente antes de outra ação.",
        });
        return;
      }
      if (outcome.state === "validation" && outcome.fields.temporary_password) {
        setError(outcome.fields.temporary_password);
        focusError.current = true;
        return;
      }
      if (outcome.state === "uncertain" || outcome.state === "confirmed")
        markUncertain();
      else
        setNotice({
          variant: "error",
          title: "Não foi possível reativar o profissional",
          message: "Confira os dados e tente novamente.",
        });
    } catch {
      if (!request.signal.aborted) {
        if (read)
          setNotice({
            variant: "error",
            title: "Não foi possível consultar o cadastro",
            message: "Tente consultar novamente antes de outra ação.",
          });
        else markUncertain();
      }
    } finally {
      onBusyChange(false);
      if (!request.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void run(false);
  }
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-label="Reativar profissional"
      aria-busy={submitting}
      className={formStyles.panel}
    >
      <h2>Reativar {record.name}?</h2>
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
            disabled={!canWrite || submitting || needsConsult}
            required
            onChange={(event) => {
              setPassword(event.target.value);
              setError(undefined);
              setNotice(null);
            }}
          />
          <Button
            type="submit"
            disabled={!canWrite || submitting || needsConsult || record.active}
          >
            {submitting ? "Aguarde…" : "Confirmar reativação"}
          </Button>
        </form>
      ) : null}
      <div className={styles.actions}>
        {needsConsult ? (
          <Button disabled={submitting} onClick={() => void run(true)}>
            Consultar cadastro do profissional
          </Button>
        ) : null}
        <Button
          disabled={submitting || needsConsult}
          onClick={() => {
            setPassword("");
            onClose();
          }}
        >
          {finished || missing ? "Fechar reativação" : "Cancelar reativação"}
        </Button>
      </div>
    </section>
  );
}
