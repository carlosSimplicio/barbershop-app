export const INVALID_PASSWORD_MESSAGE =
  "Use de 8 a 32 caracteres, sem espaços.";

export function validatePassword(
  value: string,
  requiredMessage: string,
): string | undefined {
  if (!value) return requiredMessage;
  if (
    [...value].length < 8 ||
    [...value].length > 32 ||
    /\p{White_Space}/u.test(value)
  ) {
    return INVALID_PASSWORD_MESSAGE;
  }
}
