export const EMAIL_MAX_LENGTH = 254;

export function validateEmail(value: string): string | undefined {
  const email = value.trim();
  if (!email) return "Informe seu e-mail.";
  if (
    email.length > EMAIL_MAX_LENGTH ||
    !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  ) {
    return "Digite um e-mail válido.";
  }
}
