"use client";

import { useRef, useState } from "react";
import type { FormEvent } from "react";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button, Feedback, PasswordField } from "@/components/ui";
import { changeTemporaryPassword } from "@/lib/auth/password-api";

import styles from "./testing-panel.module.css";

type FeedbackKind = "error" | "session" | "denied" | null;

export function PasswordChangePanel() {
  const { state, transitioning, runAction } = useAuthSession();
  if (
    state.status !== "authenticated" ||
    !state.principal.password_change_required
  ) {
    return null;
  }

  const identity = `${state.principal.user_id}:${state.principal.establishment_id}`;
  return (
    <PasswordChangeForm
      key={identity}
      transitioning={transitioning}
      runAction={runAction}
    />
  );
}

function PasswordChangeForm({
  transitioning,
  runAction,
}: {
  transitioning: boolean;
  runAction: ReturnType<typeof useAuthSession>["runAction"];
}) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<FeedbackKind>(null);
  const [submitting, setSubmitting] = useState(false);
  const pending = useRef(false);
  const field = useRef<HTMLInputElement>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending.current || transitioning) return;

    const validationError = validatePassword(password);
    if (validationError) {
      setError(validationError);
      field.current?.focus();
      return;
    }

    pending.current = true;
    setSubmitting(true);
    setError(null);
    setFeedback(null);
    try {
      const result = await runAction(
        () => changeTemporaryPassword({ new_password: password }),
        "authenticated",
      );
      if (result.status === 204 && result.state !== "error") return;
      if (result.state === "error") {
        const fieldError = result.error.fields?.find(
          (issue) => issue.field === "new_password",
        );
        if (fieldError?.code === "invalid_password") {
          setError("Use de 8 a 32 caracteres, sem espaços.");
          field.current?.focus();
          return;
        }
        if (fieldError?.code === "password_unchanged") {
          setError("Escolha uma senha diferente da temporária.");
          field.current?.focus();
          return;
        }
        if (result.status === 401) {
          setFeedback("session");
          return;
        }
        if (result.error.code === "password_change_not_allowed") {
          setFeedback("denied");
          return;
        }
      }
      setFeedback("error");
    } catch {
      setFeedback("error");
    } finally {
      pending.current = false;
      setSubmitting(false);
    }
  }

  return (
    <form
      aria-labelledby="password-change-panel-title"
      className={styles.panel}
      noValidate
      onSubmit={(event) => void submit(event)}
    >
      <h2 id="password-change-panel-title">Trocar senha temporária</h2>
      <p>Defina uma nova senha para liberar as funções do estabelecimento.</p>
      {feedback ? (
        <Feedback title="Não foi possível trocar a senha" variant="error">
          {feedback === "session"
            ? "Consulte sua sessão novamente antes de tentar."
            : feedback === "denied"
              ? "A troca não está disponível nesta sessão. Consulte sua sessão novamente."
              : "Tente novamente mais tarde."}
        </Feedback>
      ) : null}
      <PasswordField
        autoComplete="new-password"
        error={error}
        label="Nova senha"
        onChange={(event) => {
          setPassword(event.target.value);
          setError(null);
          setFeedback(null);
        }}
        readOnly={submitting || transitioning}
        ref={field}
        required
        spellCheck={false}
        value={password}
      />
      <Button disabled={submitting || transitioning} type="submit">
        {submitting ? "Trocando senha…" : "Trocar senha"}
      </Button>
    </form>
  );
}

function validatePassword(password: string) {
  if (!password) return "Informe uma nova senha.";
  if (
    [...password].length < 8 ||
    [...password].length > 32 ||
    /\s/.test(password)
  ) {
    return "Use de 8 a 32 caracteres, sem espaços.";
  }
  return null;
}
