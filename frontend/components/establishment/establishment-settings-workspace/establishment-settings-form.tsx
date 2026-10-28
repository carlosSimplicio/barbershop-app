"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { EstablishmentSlugField } from "@/components/establishment/establishment-slug-field/establishment-slug-field";
import { BrazilianTimeZoneField } from "@/components/establishment/brazilian-time-zone-field/brazilian-time-zone-field";
import {
  BrazilianPhoneField,
  Button,
  Feedback,
  TextareaField,
  TextField,
} from "@/components/ui";
import {
  getEstablishmentSettings,
  patchEstablishmentSettings,
} from "@/lib/establishment/establishment-api";
import type {
  EstablishmentSettings,
  EstablishmentSettingsPatch,
} from "@/lib/establishment/establishment-api";
import { BRAZILIAN_TIME_ZONES } from "@/lib/establishment/brazilian-time-zones";

import {
  buildPatch,
  createDraft,
  hasDraftChanges,
  mapServerFieldErrors,
  validateDraft,
} from "./establishment-settings-model";
import type {
  FieldErrors,
  SettingsDraft,
} from "./establishment-settings-model";
import { SocialLinksEditor } from "./social-links-editor";
import styles from "./establishment-settings-workspace.module.css";

type FeedbackState = {
  variant: "error" | "success" | "warning";
  title: string;
  message: string;
};

export function SettingsForm({
  settings: initialSettings,
}: {
  settings: EstablishmentSettings;
}) {
  const [settings, setSettings] = useState(initialSettings);
  const [draft, setDraft] = useState(() => createDraft(initialSettings));
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const activeMutation = useRef<AbortController | null>(null);
  const fieldRefs = useRef<
    Record<
      string,
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement | null
    >
  >({});
  const hasChanges = hasDraftChanges(draft, settings);
  const timeZoneIsListed = BRAZILIAN_TIME_ZONES.some(
    (timeZone) => timeZone.value === draft.time_zone,
  );

  useEffect(
    () => () => {
      activeMutation.current?.abort();
    },
    [],
  );

  function updateDraft<K extends keyof SettingsDraft>(
    field: K,
    value: SettingsDraft[K],
  ) {
    setDraft((current) => ({ ...current, [field]: value }));
    setErrors((current) => {
      const next = { ...current };
      delete next[field];
      return next;
    });
    setFeedback(null);
  }

  function updateSocialLinks(
    socialLinks: SettingsDraft["social_links"],
    fieldsToClear: string[] | "all",
  ) {
    setDraft((current) => ({ ...current, social_links: socialLinks }));
    setErrors((current) => {
      const next = { ...current };
      delete next.social_links;
      if (fieldsToClear === "all") {
        for (const field of Object.keys(next)) {
          if (field.startsWith("social_links.")) delete next[field];
        }
      } else {
        for (const field of fieldsToClear) delete next[field];
      }
      return next;
    });
    setFeedback(null);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current || !settings.can_write) return;

    const nextErrors = validateDraft(
      draft,
      settings.time_zone_editable,
      timeZoneIsListed,
    );
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      fieldRefs.current[Object.keys(nextErrors)[0]]?.focus();
      return;
    }

    const patch = buildPatch(draft, settings);
    if (Object.keys(patch).length === 0) return;

    submittingRef.current = true;
    setSubmitting(true);
    setFeedback(null);
    const controller = new AbortController();
    activeMutation.current = controller;

    void saveSettings(patch, controller.signal)
      .then(({ result, refreshedSettings }) => {
        if (controller.signal.aborted) return;
        if (refreshedSettings) {
          setSettings(refreshedSettings);
          setDraft(createDraft(refreshedSettings));
        }
        if (result.state === "empty" && result.status === 204) {
          setFeedback({
            variant: "success",
            title: "Configurações salvas",
            message: refreshedSettings
              ? "Os dados do estabelecimento foram atualizados."
              : "Os dados foram salvos. Não foi possível atualizar a consulta agora.",
          });
          setErrors({});
          return;
        }
        if (result.state === "error" && result.error.code === "read_only") {
          setSettings((current) => ({ ...current, can_write: false }));
          setFeedback({
            variant: "warning",
            title: "Configurações somente para leitura",
            message:
              "O trial expirou. Você pode consultar as configurações, mas não salvá-las.",
          });
          return;
        }
        if (result.state === "error") {
          const serverErrors = mapServerFieldErrors(result.error.fields ?? []);
          if (Object.keys(serverErrors).length > 0) {
            setErrors(serverErrors);
            fieldRefs.current[Object.keys(serverErrors)[0]]?.focus();
            if (
              result.error.fields?.some(
                (field) => field.code === "time_zone_locked",
              )
            ) {
              setSettings((current) => ({
                ...current,
                time_zone_editable: false,
              }));
            }
            return;
          }
        }
        setFeedback({
          variant: "error",
          title: "Não foi possível salvar as configurações",
          message: "Verifique os dados e tente novamente mais tarde.",
        });
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setFeedback({
          variant: "error",
          title: "Não foi possível salvar as configurações",
          message: "Verifique sua conexão e tente novamente.",
        });
      })
      .finally(() => {
        if (controller.signal.aborted) return;
        submittingRef.current = false;
        activeMutation.current = null;
        setSubmitting(false);
      });
  }

  return (
    <section
      aria-labelledby="owner-panel-page-title"
      aria-busy={submitting}
      className={styles.panel}
    >
      {!settings.can_write ? (
        <Feedback title="Configurações somente para leitura" variant="warning">
          O trial expirou. Você pode consultar as configurações, mas não
          salvá-las.
        </Feedback>
      ) : null}
      {settings.slug_conflict ? (
        <Feedback title="Endereço público indisponível" variant="warning">
          O slug atual conflita com uma rota da plataforma. Ele foi preservado e
          não será alterado automaticamente; a publicação exige resolução
          operacional explícita.
        </Feedback>
      ) : null}
      {feedback ? (
        <Feedback title={feedback.title} variant={feedback.variant}>
          {feedback.message}
        </Feedback>
      ) : null}

      <form
        className={styles.form}
        noValidate
        onSubmit={(event) => void submit(event)}
      >
        <fieldset
          className={styles.fields}
          disabled={!settings.can_write || submitting}
        >
          <legend className={styles.visuallyHidden}>
            Dados do estabelecimento
          </legend>
          <TextField
            autoComplete="organization"
            error={errors.name}
            id="name"
            label="Nome do estabelecimento"
            maxLength={120}
            onChange={(event) => updateDraft("name", event.target.value)}
            ref={(element) => {
              fieldRefs.current.name = element;
            }}
            required
            value={draft.name}
          />
          <EstablishmentSlugField
            hint="O slug é imutável e faz parte do endereço público."
            id="slug"
            label="Slug"
            disabled
            value={settings.slug}
          />
          <BrazilianTimeZoneField
            error={errors.time_zone}
            emptyOptionLabel={
              timeZoneIsListed ? undefined : "Selecione um fuso horário"
            }
            hint={
              settings.time_zone_editable
                ? "Escolha um fuso horário do Brasil."
                : "O fuso foi bloqueado após o primeiro dado operacional."
            }
            id="time_zone"
            label="Fuso horário"
            onChange={(event) => updateDraft("time_zone", event.target.value)}
            disabled={!settings.time_zone_editable}
            ref={(element) => {
              fieldRefs.current.time_zone = element;
            }}
            required
            value={timeZoneIsListed ? draft.time_zone : ""}
          />
          <BrazilianPhoneField
            error={errors.contact_phone}
            hint="Opcional. Informe telefone brasileiro com DDD."
            id="contact_phone"
            label="Telefone de contato"
            onValueChange={(value) => updateDraft("contact_phone", value)}
            ref={(element) => {
              fieldRefs.current.contact_phone = element;
            }}
            value={draft.contact_phone}
          />
          <TextareaField
            error={errors.description}
            hint="Opcional. Até 500 caracteres."
            id="description"
            label="Descrição"
            onChange={(event) => updateDraft("description", event.target.value)}
            ref={(element) => {
              fieldRefs.current.description = element;
            }}
            value={draft.description}
          />
          <TextareaField
            error={errors.address}
            hint="Opcional. Até 500 caracteres."
            id="address"
            label="Endereço"
            onChange={(event) => updateDraft("address", event.target.value)}
            ref={(element) => {
              fieldRefs.current.address = element;
            }}
            value={draft.address}
          />
          <SocialLinksEditor
            errors={errors}
            links={draft.social_links}
            onChange={updateSocialLinks}
            registerFieldRef={(id, element) => {
              fieldRefs.current[id] = element;
            }}
          />

          <TextField
            error={errors.minimum_booking_notice_minutes}
            hint="Minutos. Vazio remove o limite; zero não exige antecedência mínima."
            id="minimum_booking_notice_minutes"
            label="Prazo de antecedência mínima"
            max={2147483647}
            min={0}
            onChange={(event) =>
              updateDraft("minimum_booking_notice_minutes", event.target.value)
            }
            ref={(element) => {
              fieldRefs.current.minimum_booking_notice_minutes = element;
            }}
            step={1}
            type="number"
            value={draft.minimum_booking_notice_minutes}
          />
          <TextField
            error={errors.booking_horizon_days}
            hint="Dias no futuro. Vazio remove o limite."
            id="booking_horizon_days"
            label="Horizonte máximo de agendamento"
            max={2147483647}
            min={1}
            onChange={(event) =>
              updateDraft("booking_horizon_days", event.target.value)
            }
            ref={(element) => {
              fieldRefs.current.booking_horizon_days = element;
            }}
            step={1}
            type="number"
            value={draft.booking_horizon_days}
          />
          <TextField
            error={errors.customer_change_deadline_minutes}
            hint="Minutos antes do atendimento. Vazio remove o limite; zero não exige antecedência."
            id="customer_change_deadline_minutes"
            label="Prazo para alterar agendamento"
            max={2147483647}
            min={0}
            onChange={(event) =>
              updateDraft(
                "customer_change_deadline_minutes",
                event.target.value,
              )
            }
            ref={(element) => {
              fieldRefs.current.customer_change_deadline_minutes = element;
            }}
            step={1}
            type="number"
            value={draft.customer_change_deadline_minutes}
          />
        </fieldset>

        <Button
          disabled={!settings.can_write || submitting || !hasChanges}
          type="submit"
        >
          {submitting ? "Salvando…" : "Salvar configurações"}
        </Button>
      </form>
    </section>
  );
}

async function saveSettings(
  patch: EstablishmentSettingsPatch,
  signal: AbortSignal,
) {
  const result = await patchEstablishmentSettings(patch, signal);
  if (result.state !== "empty" || result.status !== 204) {
    return { result, refreshedSettings: null };
  }

  try {
    const refreshed = await getEstablishmentSettings(signal);
    return {
      result,
      refreshedSettings:
        refreshed.state === "success" && refreshed.status === 200
          ? refreshed.data
          : null,
    };
  } catch {
    return { result, refreshedSettings: null };
  }
}
