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
  const [uploading, setUploading] = useState(false);
  const [feedback, setFeedback] = useState<LogoFeedback | null>(null);
  const [imageVersion, setImageVersion] = useState(0);
  const [inputVersion, setInputVersion] = useState(0);
  const [uploadedWithoutRefresh, setUploadedWithoutRefresh] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const uploadRef = useRef(false);
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
    setFeedback(null);
    setFile(selected);
    setPreviewUrl(selected ? URL.createObjectURL(selected) : null);
  }

  function clearSelection() {
    setFeedback(null);
    resetSelection();
  }

  function resetSelection() {
    setFile(null);
    setPreviewUrl(null);
    setInputVersion((version) => version + 1);
    if (inputRef.current) inputRef.current.value = "";
  }

  function submit() {
    if (
      !file ||
      uploadRef.current ||
      uploading ||
      disabled ||
      !settings.can_write
    ) {
      return;
    }

    uploadRef.current = true;
    setUploading(true);
    onBusyChange(true);
    setFeedback(null);
    const controller = new AbortController();
    activeUpload.current = controller;

    void uploadEstablishmentLogo(file, controller.signal)
      .then(async (result) => {
        if (controller.signal.aborted) return;
        if (result.state === "error") {
          if (result.status >= 500) {
            showUploadError();
            return;
          }
          if (result.error.code === "read_only") {
            resetSelection();
            onSettingsChange({ ...settings, can_write: false });
            setFeedback({
              title: "Configurações somente para leitura",
              message:
                "O trial expirou. Você pode consultar o logo, mas não enviá-lo ou substituí-lo.",
              variant: "warning",
            });
            return;
          }
          resetSelection();
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
          showUploadError();
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
      .catch(() => {
        if (!controller.signal.aborted) showUploadError();
      })
      .finally(() => {
        if (controller.signal.aborted) return;
        uploadRef.current = false;
        activeUpload.current = null;
        setUploading(false);
        onBusyChange(false);
      });
  }

  function showUploadError() {
    setFeedback({
      variant: "error",
      title: "Não foi possível enviar o logo",
      message: "Tente novamente mais tarde.",
    });
  }

  async function readCurrentSettings(signal: AbortSignal) {
    try {
      const refreshed = await getEstablishmentSettings(signal);
      if (refreshed.state === "success" && refreshed.status === 200) {
        return refreshed.data;
      }
    } catch {
      // The upload already succeeded; a failed read must not repeat it.
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

  return (
    <section
      aria-busy={uploading}
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

      {previewUrl || currentLogoUrl ? (
        <div className={styles.preview}>
          {previewUrl ? (
            <>
              <Image
                alt="Pré-visualização do logo"
                height={180}
                src={previewUrl}
                unoptimized
                width={240}
              />
              {file ? (
                <Button
                  aria-label="Limpar arquivo selecionado"
                  className={styles.clearSelection}
                  disabled={uploading}
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
        disabled={!settings.can_write || disabled || uploading}
        hint="PNG, JPEG ou WebP. Até 2 MiB e 4.096 × 4.096 pixels. O servidor confirma se a imagem é válida."
        id="establishment-logo-file"
        key={inputVersion}
        label={settings.logo ? "Escolher novo logo" : "Selecionar logo"}
        onChange={(event) => chooseFile(event.currentTarget.files?.[0] ?? null)}
        ref={inputRef}
      />

      <div className={styles.actions}>
        <Button
          disabled={!file || !settings.can_write || disabled || uploading}
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
