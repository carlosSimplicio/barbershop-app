export const ESTABLISHMENT_LIMITS = {
  nameMaxCodePoints: 120,
  profileTextMaxCodePoints: 500,
  socialLinksMaxCount: 5,
  socialLinkLabelMaxCodePoints: 60,
  socialLinkURLMaxCodePoints: 2_048,
  slug: { minLength: 3, maxLength: 63 },
  bookingLimitMax: 2_147_483_647,
} as const;

export const INVALID_ESTABLISHMENT_NAME_MESSAGE = `Informe um nome com até ${ESTABLISHMENT_LIMITS.nameMaxCodePoints} caracteres.`;
export const INVALID_ESTABLISHMENT_TEXT_MESSAGE = `O texto deve ter até ${ESTABLISHMENT_LIMITS.profileTextMaxCodePoints} caracteres.`;
export const INVALID_SOCIAL_LINK_LABEL_MESSAGE = `Informe um rótulo com até ${ESTABLISHMENT_LIMITS.socialLinkLabelMaxCodePoints} caracteres.`;
export const TOO_MANY_SOCIAL_LINKS_MESSAGE = `Informe no máximo ${ESTABLISHMENT_LIMITS.socialLinksMaxCount} links.`;

export const INVALID_PUBLIC_SLUG_MESSAGE = `Use de ${ESTABLISHMENT_LIMITS.slug.minLength} a ${ESTABLISHMENT_LIMITS.slug.maxLength} letras minúsculas, números ou hífens.`;
