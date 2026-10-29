"use client";

import { useEffect, useRef } from "react";

import { AffixedInput, Button, Field, SelectField } from "@/components/ui";
import { ESTABLISHMENT_LIMITS } from "@/lib/establishment/establishment-limits";
import {
  ESTABLISHMENT_SOCIAL_NETWORKS,
  type EstablishmentSocialNetworkId,
} from "@/lib/establishment/social-networks";

import {
  getSocialNetwork,
  getSocialNetworkSuffix,
} from "./establishment-settings-model";
import type {
  FieldErrors,
  SocialLinkDraft,
} from "./establishment-settings-model";
import styles from "./establishment-settings-workspace.module.css";

type SocialLinkFieldElement = HTMLInputElement | HTMLSelectElement | null;

type SocialLinksEditorProps = {
  errors: FieldErrors;
  links: SocialLinkDraft[];
  onChange: (links: SocialLinkDraft[], fieldsToClear: string[] | "all") => void;
  registerFieldRef: (id: string, element: SocialLinkFieldElement) => void;
};

export function SocialLinksEditor({
  errors,
  links,
  onChange,
  registerFieldRef,
}: SocialLinksEditorProps) {
  const nextLinkKey = useRef(links.length);
  const pendingFocusIndex = useRef<number | null>(null);
  const fieldRefs = useRef<Record<string, SocialLinkFieldElement>>({});

  useEffect(() => {
    const index = pendingFocusIndex.current;
    if (index === null) return;

    fieldRefs.current[`social_links.${index}.url`]?.focus();
    pendingFocusIndex.current = null;
  }, [links]);

  function registerSocialFieldRef(id: string, element: SocialLinkFieldElement) {
    fieldRefs.current[id] = element;
    registerFieldRef(id, element);
  }

  function addLink() {
    onChange(
      [
        ...links,
        {
          key: nextLinkKey.current++,
          label: "",
          network: "",
          url: "",
        },
      ],
      ["social_links"],
    );
  }

  function selectNetwork(
    index: number,
    network: EstablishmentSocialNetworkId | "",
  ) {
    const selectedNetwork = getSocialNetwork(network);
    pendingFocusIndex.current = selectedNetwork ? index : null;
    onChange(
      links.map((link, linkIndex) =>
        linkIndex === index
          ? {
              ...link,
              network,
              label: selectedNetwork?.label ?? "",
              url: selectedNetwork?.urlPrefix ?? "",
            }
          : link,
      ),
      [
        "social_links",
        `social_links.${index}.network`,
        `social_links.${index}.url`,
      ],
    );
  }

  function updateURL(index: number, url: string) {
    onChange(
      links.map((link, linkIndex) =>
        linkIndex === index ? { ...link, url } : link,
      ),
      ["social_links", `social_links.${index}.url`],
    );
  }

  function removeLink(key: number) {
    onChange(
      links.filter((link) => link.key !== key),
      "all",
    );
  }

  return (
    <div className={styles.socialLinks}>
      <div className={styles.groupHeading}>
        <h2>Redes sociais</h2>
        <Button
          aria-label="Adicionar rede social"
          className={styles.addLink}
          disabled={links.length >= ESTABLISHMENT_LIMITS.socialLinksMaxCount}
          onClick={addLink}
        >
          +
        </Button>
      </div>
      {errors.social_links ? (
        <p className={styles.groupError} role="alert">
          {errors.social_links}
        </p>
      ) : null}
      {links.length === 0 ? (
        <p className={styles.emptyLinks}>Nenhuma rede cadastrada.</p>
      ) : null}
      {links.map((link, index) => (
        <div
          aria-label={`Link de rede social ${index + 1}`}
          className={styles.socialLink}
          key={link.key}
          role="group"
        >
          <SelectField
            error={errors[`social_links.${index}.network`]}
            fieldClassName={styles.socialNetworkField}
            id={`social_links.${index}.network`}
            label={`Rede social ${index + 1}`}
            onChange={(event) =>
              selectNetwork(
                index,
                event.target.value as EstablishmentSocialNetworkId | "",
              )
            }
            ref={(element) =>
              registerSocialFieldRef(`social_links.${index}.network`, element)
            }
            value={link.network}
          >
            <option value="">Selecione uma rede</option>
            {ESTABLISHMENT_SOCIAL_NETWORKS.filter(
              (network) =>
                network.value === link.network ||
                !links.some(
                  (otherLink, otherIndex) =>
                    otherIndex !== index && otherLink.network === network.value,
                ),
            ).map((network) => (
              <option key={network.value} value={network.value}>
                {network.label}
              </option>
            ))}
          </SelectField>
          {getSocialNetwork(link.network) ? (
            <Field
              className={styles.socialUrlField}
              error={errors[`social_links.${index}.url`]}
              errorId={`social_links.${index}.url-error`}
              htmlFor={`social_links.${index}.url`}
              label={`URL do ${getSocialNetwork(link.network)?.label}`}
            >
              <AffixedInput
                aria-describedby={
                  errors[`social_links.${index}.url`]
                    ? `social_links.${index}.url-error`
                    : undefined
                }
                aria-invalid={Boolean(errors[`social_links.${index}.url`])}
                affix={getSocialNetwork(link.network)?.urlPrefix}
                autoCapitalize="none"
                autoComplete="url"
                id={`social_links.${index}.url`}
                maxLength={
                  ESTABLISHMENT_LIMITS.socialLinkURLMaxCodePoints -
                  (getSocialNetwork(link.network)?.urlPrefix.length ?? 0)
                }
                onChange={(event) =>
                  updateURL(
                    index,
                    `${getSocialNetwork(link.network)?.urlPrefix ?? ""}${event.target.value}`,
                  )
                }
                ref={(element) =>
                  registerSocialFieldRef(`social_links.${index}.url`, element)
                }
                spellCheck={false}
                value={getSocialNetworkSuffix(link.url, link.network)}
              />
            </Field>
          ) : link.label || link.url ? (
            <p className={`${styles.legacyLink} ${styles.socialUrlField}`}>
              Este link existente não corresponde a uma das redes disponíveis.
              Escolha uma rede para substituí-lo ou remova-o.
            </p>
          ) : null}
          <Button
            aria-label={`Remover rede social ${link.label || index + 1}`}
            className={styles.removeLink}
            onClick={() => removeLink(link.key)}
          >
            ×
          </Button>
        </div>
      ))}
    </div>
  );
}
