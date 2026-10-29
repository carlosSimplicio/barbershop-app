export const PASSWORD_MIN_CODE_POINTS = 8;
export const PASSWORD_MAX_CODE_POINTS = 32;
export const INVALID_PASSWORD_MESSAGE = `Use de ${PASSWORD_MIN_CODE_POINTS} a ${PASSWORD_MAX_CODE_POINTS} caracteres, sem espaços ou caracteres invisíveis.`;

const forbiddenPasswordCharacters =
  /[\p{White_Space}\p{Cc}\p{Cf}\p{Default_Ignorable_Code_Point}]/u;

export function validatePassword(
  value: string,
  requiredMessage: string,
): string | undefined {
  if (!value) return requiredMessage;
  if (
    [...value].length < PASSWORD_MIN_CODE_POINTS ||
    [...value].length > PASSWORD_MAX_CODE_POINTS ||
    forbiddenPasswordCharacters.test(value)
  ) {
    return INVALID_PASSWORD_MESSAGE;
  }
}
