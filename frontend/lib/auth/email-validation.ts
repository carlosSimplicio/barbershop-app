export const EMAIL_MAX_LENGTH = 254;
export const INVALID_EMAIL_MESSAGE = "Digite um e-mail válido.";

export function validateEmail(value: string): string | undefined {
  const email = value.trim();
  if (!email) return "Informe seu e-mail.";
  if (
    email.length > EMAIL_MAX_LENGTH ||
    !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  ) {
    return INVALID_EMAIL_MESSAGE;
  }
}
