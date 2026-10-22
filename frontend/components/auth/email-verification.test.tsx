import { render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { useRouter } from "next/navigation";

import { useAuthSession } from "@/components/session/auth-provider";
import { verifyEmail } from "@/lib/auth/email-verification-api";

import { EmailVerification } from "./email-verification";

vi.mock("next/navigation", () => ({ useRouter: vi.fn() }));
vi.mock("@/components/session/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/auth/email-verification-api", () => ({ verifyEmail: vi.fn() }));

const replace = vi.fn();
const runAction = vi.fn(async (action: () => ReturnType<typeof verifyEmail>) =>
  action(),
);

beforeEach(() => {
  window.history.replaceState(null, "", "/confirmar-email");
  replace.mockReset();
  runAction.mockClear();
  vi.mocked(verifyEmail).mockReset();
  vi.mocked(useRouter).mockReturnValue({
    replace,
  } as unknown as ReturnType<typeof useRouter>);
  vi.mocked(useAuthSession).mockReturnValue({
    state: { status: "anonymous" },
    transitioning: false,
    reload: vi.fn(),
    runAction: runAction as unknown as ReturnType<
      typeof useAuthSession
    >["runAction"],
  });
});

afterEach(() => vi.restoreAllMocks());

test("sincroniza a sessão após sucesso e substitui a rota", async () => {
  vi.mocked(verifyEmail).mockResolvedValue({
    state: "success",
    status: 200,
    data: principal,
  });
  render(<EmailVerification token="opaque-sample" />);

  await waitFor(() =>
    expect(verifyEmail).toHaveBeenCalledWith({ token: "opaque-sample" }),
  );
  expect(runAction).toHaveBeenCalledWith(expect.any(Function), "authenticated");
  await waitFor(() => expect(replace).toHaveBeenCalledWith("/testes"));
  expect(document.body.textContent).not.toContain("opaque-sample");
  expect(screen.getByText(/e-mail confirmado/i)).toBeInTheDocument();
});

test("não envia token ausente e informa link inválido", async () => {
  render(<EmailVerification token={null} />);

  expect(
    await screen.findByText(/link inválido ou expirado/i),
  ).toBeInTheDocument();
  expect(verifyEmail).not.toHaveBeenCalled();
  expect(replace).not.toHaveBeenCalled();
});

test("trata token recusado sem mostrar a mensagem bruta", async () => {
  vi.mocked(verifyEmail).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "invalid_email_confirmation",
      message: "segredo bruto",
      request_id: "test",
    },
  });
  render(<EmailVerification token="opaque-sample" />);

  expect(
    await screen.findByText(/link inválido ou expirado/i),
  ).toBeInTheDocument();
  expect(screen.queryByText("segredo bruto")).not.toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("informa falha inesperada sem reutilizar o token", async () => {
  vi.mocked(verifyEmail).mockRejectedValue(new Error("network"));
  render(<EmailVerification token="opaque-sample" />);

  expect(
    await screen.findByText(/não foi possível confirmar/i),
  ).toBeInTheDocument();
  expect(verifyEmail).toHaveBeenCalledOnce();
});

test("Strict Mode não envia duas vezes um token de uso único", async () => {
  vi.mocked(verifyEmail).mockResolvedValue({
    state: "success",
    status: 200,
    data: principal,
  });
  render(
    <StrictMode>
      <EmailVerification token="opaque-sample" />
    </StrictMode>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/testes"));
  expect(verifyEmail).toHaveBeenCalledOnce();
});

const principal = {
  user_id: "user-id",
  establishment_id: "establishment-id",
  email: "owner@example.com",
  role: "owner" as const,
  password_change_required: false,
};
