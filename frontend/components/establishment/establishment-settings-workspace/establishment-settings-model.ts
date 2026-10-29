import {
  ESTABLISHMENT_LIMITS,
  INVALID_ESTABLISHMENT_NAME_MESSAGE,
  INVALID_ESTABLISHMENT_TEXT_MESSAGE,
  INVALID_SOCIAL_LINK_LABEL_MESSAGE,
  TOO_MANY_SOCIAL_LINKS_MESSAGE,
} from "@/lib/establishment/establishment-limits";
import type {
  EstablishmentSettings,
  EstablishmentSettingsPatch,
} from "@/lib/establishment/establishment-api";
import {
  ESTABLISHMENT_SOCIAL_NETWORKS,
  type EstablishmentSocialNetworkId,
} from "@/lib/establishment/social-networks";

export type SocialLinkDraft = {
  key: number;
  label: string;
  network: EstablishmentSocialNetworkId | "";
  url: string;
};

export type SettingsDraft = {
  name: string;
  time_zone: string;
  description: string;
  address: string;
  contact_phone: string;
  social_links: SocialLinkDraft[];
  minimum_booking_notice_minutes: string;
  booking_horizon_days: string;
  customer_change_deadline_minutes: string;
};

export type FieldErrors = Record<string, string>;

const serverMessages: Record<string, string> = {
  invalid_name: INVALID_ESTABLISHMENT_NAME_MESSAGE,
  invalid_time_zone: "Informe um fuso horário IANA válido.",
  time_zone_locked:
    "O fuso não pode ser alterado após o primeiro dado operacional.",
  invalid_text: INVALID_ESTABLISHMENT_TEXT_MESSAGE,
  invalid_phone: "Informe um telefone brasileiro com DDD.",
  too_many_links: TOO_MANY_SOCIAL_LINKS_MESSAGE,
  invalid_label: INVALID_SOCIAL_LINK_LABEL_MESSAGE,
  invalid_url: "Informe uma URL HTTPS válida sem credenciais.",
  invalid_limit: "Informe um número inteiro não negativo.",
  invalid_horizon: "Informe um número inteiro maior que zero.",
};

export function createDraft(settings: EstablishmentSettings): SettingsDraft {
  return {
    name: settings.name,
    time_zone: settings.time_zone,
    description: settings.description ?? "",
    address: settings.address ?? "",
    contact_phone: settings.contact_phone ?? "",
    social_links: settings.social_links.map((link, index) => ({
      key: index,
      label: link.label,
      network: identifySocialNetwork(link.url) ?? "",
      url: link.url,
    })),
    minimum_booking_notice_minutes:
      settings.minimum_booking_notice_minutes?.toString() ?? "",
    booking_horizon_days: settings.booking_horizon_days?.toString() ?? "",
    customer_change_deadline_minutes:
      settings.customer_change_deadline_minutes?.toString() ?? "",
  };
}

export function hasDraftChanges(
  draft: SettingsDraft,
  settings: EstablishmentSettings,
) {
  return (
    JSON.stringify(comparableDraft(draft)) !==
    JSON.stringify(comparableSettings(settings))
  );
}

export function getSocialNetwork(network: SocialLinkDraft["network"]) {
  return ESTABLISHMENT_SOCIAL_NETWORKS.find(
    (option) => option.value === network,
  );
}

export function getSocialNetworkSuffix(
  url: string,
  networkId: SocialLinkDraft["network"],
) {
  const network = getSocialNetwork(networkId);
  if (!network || !url) return "";
  if (url.startsWith(network.urlPrefix)) {
    return url.slice(network.urlPrefix.length);
  }

  try {
    const parsed = new URL(url);
    const prefixPath = new URL(network.urlPrefix).pathname;
    const path = parsed.pathname.startsWith(prefixPath)
      ? parsed.pathname.slice(prefixPath.length)
      : parsed.pathname.replace(/^\/+/, "");
    return `${path}${parsed.search}${parsed.hash}`;
  } catch {
    return url;
  }
}

export function validateDraft(
  draft: SettingsDraft,
  timeZoneEditable: boolean,
  timeZoneIsListed: boolean,
): FieldErrors {
  const errors: FieldErrors = {};
  const name = draft.name.trim();
  if (!name || characterCount(name) > ESTABLISHMENT_LIMITS.nameMaxCodePoints) {
    errors.name = name
      ? INVALID_ESTABLISHMENT_NAME_MESSAGE
      : "Informe o nome do estabelecimento.";
  }
  if (timeZoneEditable && !timeZoneIsListed) {
    errors.time_zone = "Escolha um dos quatro fusos horários brasileiros.";
  }
  for (const field of ["description", "address"] as const) {
    const value = draft[field].trim();
    if (characterCount(value) > ESTABLISHMENT_LIMITS.profileTextMaxCodePoints) {
      errors[field] = INVALID_ESTABLISHMENT_TEXT_MESSAGE;
    }
  }
  if (
    draft.contact_phone.trim() &&
    !isValidBrazilianPhone(draft.contact_phone)
  ) {
    errors.contact_phone = "Informe um telefone brasileiro com DDD.";
  }

  for (const [index, link] of draft.social_links.entries()) {
    const label = link.label.trim();
    const url = link.url.trim();
    if (!label && !url) continue;
    if (link.network && !getSocialNetworkSuffix(url, link.network).trim()) {
      errors[`social_links.${index}.url`] = "Informe o endereço do perfil.";
      continue;
    }
    // Existing links from before the supported network selector remain intact
    // until the owner replaces or removes them.
    if (!link.network) continue;
    if (
      !label ||
      characterCount(label) > ESTABLISHMENT_LIMITS.socialLinkLabelMaxCodePoints
    ) {
      errors[`social_links.${index}.label`] = INVALID_SOCIAL_LINK_LABEL_MESSAGE;
    }
    if (!isValidSocialURL(url)) {
      errors[`social_links.${index}.url`] =
        "Informe uma URL HTTPS válida sem credenciais.";
    }
  }

  validateLimit(
    draft.minimum_booking_notice_minutes,
    "minimum_booking_notice_minutes",
    false,
    errors,
  );
  validateLimit(
    draft.booking_horizon_days,
    "booking_horizon_days",
    true,
    errors,
  );
  validateLimit(
    draft.customer_change_deadline_minutes,
    "customer_change_deadline_minutes",
    false,
    errors,
  );
  return errors;
}

export function buildPatch(
  draft: SettingsDraft,
  settings: EstablishmentSettings,
): EstablishmentSettingsPatch {
  const patch: EstablishmentSettingsPatch = {};
  const name = draft.name.trim();
  const timeZone = draft.time_zone.trim();
  const description = draft.description.trim() || null;
  const address = draft.address.trim() || null;
  const contactPhone = draft.contact_phone.trim() || null;
  const socialLinks = draft.social_links
    .filter((link) => link.label.trim() || link.url.trim())
    .map(({ label, url }) => ({ label: label.trim(), url: url.trim() }));

  if (name !== settings.name) patch.name = name;
  if (settings.time_zone_editable && timeZone !== settings.time_zone) {
    patch.time_zone = timeZone;
  }
  if (description !== (settings.description ?? null)) {
    patch.description = description;
  }
  if (address !== (settings.address ?? null)) patch.address = address;
  if (contactPhone !== (settings.contact_phone ?? null)) {
    patch.contact_phone = contactPhone;
  }
  if (JSON.stringify(socialLinks) !== JSON.stringify(settings.social_links)) {
    patch.social_links = socialLinks;
  }

  setChangedLimit(
    patch,
    draft.minimum_booking_notice_minutes,
    settings.minimum_booking_notice_minutes,
    "minimum_booking_notice_minutes",
  );
  setChangedLimit(
    patch,
    draft.booking_horizon_days,
    settings.booking_horizon_days,
    "booking_horizon_days",
  );
  setChangedLimit(
    patch,
    draft.customer_change_deadline_minutes,
    settings.customer_change_deadline_minutes,
    "customer_change_deadline_minutes",
  );
  return patch;
}

export function mapServerFieldErrors(
  fields: { field: string; code: string }[],
): FieldErrors {
  const errors: FieldErrors = {};
  for (const issue of fields) {
    const message = serverMessages[issue.code];
    if (message) errors[issue.field] = message;
  }
  return errors;
}

function comparableDraft(draft: SettingsDraft) {
  return {
    name: draft.name,
    time_zone: draft.time_zone,
    description: draft.description,
    address: draft.address,
    contact_phone: draft.contact_phone,
    social_links: draft.social_links
      .filter((link) => link.label.trim() || link.url.trim())
      .map(({ label, url }) => ({ label, url })),
    minimum_booking_notice_minutes: draft.minimum_booking_notice_minutes,
    booking_horizon_days: draft.booking_horizon_days,
    customer_change_deadline_minutes: draft.customer_change_deadline_minutes,
  };
}

function comparableSettings(settings: EstablishmentSettings) {
  return {
    name: settings.name,
    time_zone: settings.time_zone,
    description: settings.description ?? "",
    address: settings.address ?? "",
    contact_phone: settings.contact_phone ?? "",
    social_links: settings.social_links.map(({ label, url }) => ({
      label,
      url,
    })),
    minimum_booking_notice_minutes:
      settings.minimum_booking_notice_minutes?.toString() ?? "",
    booking_horizon_days: settings.booking_horizon_days?.toString() ?? "",
    customer_change_deadline_minutes:
      settings.customer_change_deadline_minutes?.toString() ?? "",
  };
}

function identifySocialNetwork(url: string) {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return undefined;
  }

  if (
    parsed.protocol !== "https:" ||
    parsed.username ||
    parsed.password ||
    parsed.port
  ) {
    return undefined;
  }

  return ESTABLISHMENT_SOCIAL_NETWORKS.find(
    (network) =>
      network.hostnames.some((host) => host === parsed.hostname) &&
      parsed.pathname.startsWith(new URL(network.urlPrefix).pathname),
  )?.value;
}

function validateLimit(
  value: string,
  field: string,
  positive: boolean,
  errors: FieldErrors,
) {
  if (!value.trim()) return;
  const parsed = Number(value);
  if (
    !Number.isInteger(parsed) ||
    parsed < (positive ? 1 : 0) ||
    parsed > ESTABLISHMENT_LIMITS.bookingLimitMax
  ) {
    errors[field] = positive
      ? "Informe um número inteiro maior que zero."
      : "Informe um número inteiro não negativo.";
  }
}

function setChangedLimit(
  patch: EstablishmentSettingsPatch,
  value: string,
  previous: number | null,
  field:
    | "minimum_booking_notice_minutes"
    | "booking_horizon_days"
    | "customer_change_deadline_minutes",
) {
  const parsed = value.trim() ? Number(value) : null;
  if (parsed !== previous) patch[field] = parsed;
}

function isValidBrazilianPhone(value: string) {
  if (!/^\+?[\d\s().-]+$/u.test(value.trim())) return false;
  let digits = value.replace(/\D/gu, "");
  if (value.trim().startsWith("+")) {
    if (!digits.startsWith("55")) return false;
    digits = digits.slice(2);
  } else if (
    (digits.length === 12 || digits.length === 13) &&
    digits.startsWith("55")
  ) {
    digits = digits.slice(2);
  }
  return digits.length === 10 || digits.length === 11;
}

function isValidSocialURL(value: string) {
  if (
    !value ||
    characterCount(value) > ESTABLISHMENT_LIMITS.socialLinkURLMaxCodePoints
  )
    return false;
  try {
    const parsed = new URL(value);
    return (
      parsed.protocol === "https:" &&
      parsed.hostname !== "" &&
      parsed.username === "" &&
      parsed.password === ""
    );
  } catch {
    return false;
  }
}

function characterCount(value: string) {
  return Array.from(value).length;
}
