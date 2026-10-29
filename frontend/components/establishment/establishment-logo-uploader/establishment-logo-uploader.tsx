"use client";

import Image from "next/image";
import { useEffect, useRef, useState } from "react";

import { Button, Feedback, FileField } from "@/components/ui";
import {
  getEstablishmentSettings,
  uploadEstablishmentLogo,
} from "@/lib/establishment/establishment-api";
import type { EstablishmentSettings } from "@/lib/establishment/establishment-api";

import styles from "./establishment-logo-uploader.module.css";

type LogoFeedback = {
  title: string;
  message: string;
  variant: "error" | "success" | "warning";
};

export function EstablishmentLogoUploader({
  settings,
  disabled,
  onSettingsChange,
  onBusyChange,
}: {
  settings: EstablishmentSettings;
  disabled: boolean;
  onSettingsChange: (settings: EstablishmentSettings) => void;
  onBusyChange: (busy: boolean) => void;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [showSelectedPreview, setShowSelectedPreview] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [feedback, setFeedback] = useState<LogoFeedback | null>(null);
  const [imageVersion, setImageVersion] = useState(0);
  const [inputVersion, setInputVersion] = useState(0);
  const [uploadedWithoutRefresh, setUploadedWithoutRefresh] = useState(false);
  const [confirmationNeeded, setConfirmationNeeded] = useState(false);
  const [refreshingStatus, setRefreshingStatus] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const uploadRef = useRef(false);
  const refreshingStatusRef = useRef(false);
  const activeUpload = useRef<AbortController | null>(null);

  useEffect(
    () => () => {
      activeUpload.current?.abort();
    },
    [],
  );

  useEffect(() => {
    if (!previewUrl) return;
    return () => URL.revokeObjectURL(previewUrl);
  }, [previewUrl]);

  const publicLogoUrl = settings.logo?.url;
  const fallbackLogoUrl =
    uploadedWithoutRefresh && settings.name.trim() && !settings.slug_conflict
      ? `/api/v1/public/establishments/${encodeURIComponent(settings.slug)}/logo`
      : null;
  const currentLogoUrl = publicLogoUrl ?? fallbackLogoUrl;
  const publicPageAvailable =
    settings.is_publishable ||
    (uploadedWithoutRefresh &&
      Boolean(settings.name.trim()) &&
      !settings.slug_conflict);
  const publicPageUrl = publicPageAvailable
    ? `/${encodeURIComponent(settings.slug)}`
    : null;

  function chooseFile(selected: File | null) {
    if (confirmationNeeded || refreshingStatus) return;
    setFeedback(null);
    setFile(selected);
    setPreviewUrl(selected ? URL.createObjectURL(selected) : null);
    setShowSelectedPreview(Boolean(selected));
  }

  function clearSelection() {
    setFeedback(null);
    resetSelection();
  }

  function resetSelection() {
    setFile(null);
    setPreviewUrl(null);
    setInputVersion((version) => version + 1);
    setShowSelectedPreview(false);
    if (inputRef.current) inputRef.current.value = "";
  }

  function restoreSavedLogo() {
    resetSelection();
  }

  function submit() {
    if (
      !file ||
      uploadRef.current ||
      uploading ||
      confirmationNeeded ||
      refreshingStatus ||
      disabled ||
      !settings.can_write
    ) {
      return;
    }

    uploadRef.current = true;
    setUploading(true);
    setShowSelectedPreview(true);
    onBusyChange(true);
    setFeedback(null);
    const controller = new AbortController();
    activeUpload.current = controller;

    void uploadEstablishmentLogo(file, controller.signal)
      .then(async (result) => {
        if (controller.signal.aborted) return;
        if (result.state === "error") {
          if (result.status >= 500) {
            await reconcileUploadOutcome(controller.signal);
            return;
          }
          if (result.error.code === "read_only") {
            restoreSavedLogo();
            onSettingsChange({ ...settings, can_write: false });
            setFeedback({
              title: "Configurações somente para leitura",
              message:
                "O trial expirou. Você pode consultar o logo, mas não enviá-lo ou substituí-lo.",
              variant: "warning",
            });
            return;
          }
          restoreSavedLogo();
          const fieldError = result.error.fields?.find(
            (fieldError) => fieldError.field === "file",
          );
          setFeedback({
            title: "Não foi possível enviar o logo",
            message:
              fieldError?.code === "invalid_logo"
                ? "Escolha uma imagem PNG, JPEG ou WebP válida."
                : result.error.code === "request_too_large"
                  ? "O arquivo excede o limite de 2 MiB. Selecione a imagem novamente para tentar."
                  : "Verifique a imagem e selecione-a novamente para tentar mais tarde.",
            variant: "error",
          });
          return;
        }
        if (result.state !== "empty" || result.status !== 204) {
          await reconcileUploadOutcome(controller.signal);
          return;
        }

        const updatedSettings = await readCurrentSettings(controller.signal);

        if (controller.signal.aborted) return;
        applySettingsRefresh(updatedSettings);
        setFeedback({
          title: "Logo enviado",
          message: updatedSettings
            ? "Logo atualizado com sucesso."
            : "Logo enviado. Não foi possível atualizar as configurações agora.",
          variant: "success",
        });
        resetSelection();
      })
      .catch(async () => {
        if (controller.signal.aborted) return;
        await reconcileUploadOutcome(controller.signal);
      })
      .finally(() => {
        if (controller.signal.aborted) return;
        uploadRef.current = false;
        activeUpload.current = null;
        setUploading(false);
        onBusyChange(false);
      });
  }

  async function reconcileUploadOutcome(signal: AbortSignal) {
    setConfirmationNeeded(true);
    setShowSelectedPreview(Boolean(previewUrl));
    await refreshCurrentSettings(signal);
  }

  async function readCurrentSettings(signal: AbortSignal) {
    try {
      const refreshed = await getEstablishmentSettings(signal);
      if (refreshed.state === "success" && refreshed.status === 200) {
        return refreshed.data;
      }
    } catch {
      // The caller decides whether a failed refresh is recoverable or uncertain.
    }
    return null;
  }

  function applySettingsRefresh(
    refreshedSettings: EstablishmentSettings | null,
  ) {
    if (refreshedSettings) {
      onSettingsChange(refreshedSettings);
      setUploadedWithoutRefresh(false);
    } else {
      setUploadedWithoutRefresh(true);
    }
    setImageVersion((version) => version + 1);
  }

  async function refreshCurrentSettings(signal: AbortSignal) {
    if (refreshingStatusRef.current) return;
    refreshingStatusRef.current = true;
    setRefreshingStatus(true);
    setConfirmationNeeded(true);
    onBusyChange(true);

    let refreshedSettings: EstablishmentSettings | null;
    try {
      refreshedSettings = await readCurrentSettings(signal);
    } finally {
      refreshingStatusRef.current = false;
      if (!signal.aborted) {
        setRefreshingStatus(false);
        onBusyChange(false);
      }
    }

    if (signal.aborted) return;
    if (!refreshedSettings) {
      setFeedback({
        title: "Não foi possível confirmar o envio",
        message:
          "A consulta das configurações também falhou. Atualize o estado do logo antes de enviar novamente.",
        variant: "warning",
      });
      return;
    }

    applySettingsRefresh(refreshedSettings);
    setConfirmationNeeded(false);
    resetSelection();
    setFeedback({
      title: "Estado do logo atualizado",
      message: "Confira o logo exibido antes de iniciar outro envio.",
      variant: "warning",
    });
  }

  function retryStatusRefresh() {
    if (refreshingStatusRef.current) return;
    const controller = new AbortController();
    activeUpload.current = controller;
    void refreshCurrentSettings(controller.signal).finally(() => {
      if (activeUpload.current === controller) activeUpload.current = null;
    });
  }

  return (
    <section
      aria-busy={uploading || refreshingStatus}
      aria-labelledby="establishment-logo-title"
      className={styles.section}
    >
      <div className={styles.heading}>
        <h2 id="establishment-logo-title">Logo do estabelecimento</h2>
        <p>
          {settings.logo
            ? settings.is_publishable
              ? "O logo aparece na página pública. Envie outra imagem para substituí-lo."
              : settings.slug_conflict
                ? "O logo está salvo, mas o conflito do slug impede a publicação."
                : "O logo está salvo, mas o estabelecimento ainda não pode ser publicado."
            : "Nenhum logo enviado. Selecione uma imagem para concluir os dados obrigatórios e habilitar a publicação."}
        </p>
      </div>

      {feedback ? (
        <Feedback title={feedback.title} variant={feedback.variant}>
          {feedback.message}
        </Feedback>
      ) : null}

      {(showSelectedPreview && previewUrl) || currentLogoUrl ? (
        <div className={styles.preview}>
          {showSelectedPreview && previewUrl ? (
            <>
              <Image
                alt="Pré-visualização do logo"
                height={180}
                src={previewUrl}
                unoptimized
                width={240}
              />
              {file && !confirmationNeeded ? (
                <Button
                  aria-label="Limpar arquivo selecionado"
                  className={styles.clearSelection}
                  disabled={uploading || refreshingStatus}
                  onClick={clearSelection}
                  type="button"
                >
                  ×
                </Button>
              ) : null}
            </>
          ) : currentLogoUrl ? (
            <Image
              key={`${currentLogoUrl}:${imageVersion}`}
              alt={`Logo de ${settings.name}`}
              height={180}
              src={
                imageVersion === 0
                  ? currentLogoUrl
                  : `${currentLogoUrl}${currentLogoUrl.includes("?") ? "&" : "?"}version=${imageVersion}`
              }
              unoptimized
              width={240}
            />
          ) : null}
        </div>
      ) : null}

      <FileField
        accept="image/png,image/jpeg,image/webp"
        disabled={
          !settings.can_write ||
          disabled ||
          uploading ||
          confirmationNeeded ||
          refreshingStatus
        }
        hint="PNG, JPEG ou WebP. Até 2 MiB e 4.096 × 4.096 pixels. O servidor confirma se a imagem é válida."
        id="establishment-logo-file"
        key={inputVersion}
        label={settings.logo ? "Escolher novo logo" : "Selecionar logo"}
        onChange={(event) => chooseFile(event.currentTarget.files?.[0] ?? null)}
        ref={inputRef}
      />

      <div className={styles.actions}>
        {confirmationNeeded ? (
          <Button
            disabled={uploading || refreshingStatus}
            onClick={retryStatusRefresh}
            type="button"
          >
            {refreshingStatus ? "Atualizando…" : "Atualizar estado do logo"}
          </Button>
        ) : null}
        <Button
          disabled={
            !file ||
            !settings.can_write ||
            disabled ||
            uploading ||
            confirmationNeeded ||
            refreshingStatus
          }
          onClick={submit}
          type="button"
        >
          {uploading ? "Enviando…" : "Enviar logo"}
        </Button>
        {publicPageUrl ? (
          <a href={publicPageUrl}>Abrir página pública</a>
        ) : null}
      </div>
    </section>
  );
}
