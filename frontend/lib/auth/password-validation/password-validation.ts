export const INVALID_PASSWORD_MESSAGE =
  "Use de 8 a 32 caracteres, sem espaços ou caracteres invisíveis.";

const forbiddenPasswordCharacters =
  /[\p{White_Space}\p{Cc}\p{Cf}\p{Default_Ignorable_Code_Point}]/u;

export function validatePassword(
  value: string,
  requiredMessage: string,
): string | undefined {
  if (!value) return requiredMessage;
  if (
    [...value].length < 8 ||
    [...value].length > 32 ||
    forbiddenPasswordCharacters.test(value)
  ) {
    return INVALID_PASSWORD_MESSAGE;
  }
}
