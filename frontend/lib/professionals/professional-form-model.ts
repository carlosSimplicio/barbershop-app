import { INVALID_EMAIL_MESSAGE } from "@/lib/auth/email-validation/email-validation";
import { INVALID_PASSWORD_MESSAGE } from "@/lib/auth/password-validation/password-validation";
import { PROFESSIONAL_NAME_MAX_CODE_POINTS } from "./professional-limits";

export type ProfessionalField = "name" | "email" | "temporary_password";
export type ProfessionalFieldErrors = Partial<
  Record<ProfessionalField, string>
>;

export const PROFESSIONAL_NAME_HINT = `Nome obrigatório, com até ${PROFESSIONAL_NAME_MAX_CODE_POINTS} caracteres.`;
export const INVALID_PROFESSIONAL_NAME_MESSAGE = `Informe um nome com até ${PROFESSIONAL_NAME_MAX_CODE_POINTS} caracteres.`;
export const TEMPORARY_PASSWORD_HINT = `${INVALID_PASSWORD_MESSAGE} A troca será obrigatória no primeiro acesso.`;

export function validateProfessionalName(value: string): string | undefined {
  if (!value.trim()) return "Informe o nome do profissional.";
  if (
    value.includes("\0") ||
    [...value].length > PROFESSIONAL_NAME_MAX_CODE_POINTS
  )
    return INVALID_PROFESSIONAL_NAME_MESSAGE;
}

export function professionalFieldErrors(
  issues: Array<{ field: string; code: string }> = [],
): ProfessionalFieldErrors {
  const messages: Record<string, string> = {
    invalid_name: INVALID_PROFESSIONAL_NAME_MESSAGE,
    invalid_email: INVALID_EMAIL_MESSAGE,
    email_unavailable: "Este e-mail já está cadastrado.",
    invalid_password: INVALID_PASSWORD_MESSAGE,
  };
  const fields: ProfessionalFieldErrors = {};
  for (const issue of issues) {
    if (
      (issue.field === "name" ||
        issue.field === "email" ||
        issue.field === "temporary_password") &&
      messages[issue.code]
    )
      fields[issue.field] = messages[issue.code];
  }
  return fields;
}
