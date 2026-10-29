export const OFFERING_LIMITS = {
  nameCodePoints: { max: 120 },
  priceCents: { min: 0, max: 9_999_999 },
  durationMinutes: { min: 1, default: 30, max: 1_440 },
} as const;
