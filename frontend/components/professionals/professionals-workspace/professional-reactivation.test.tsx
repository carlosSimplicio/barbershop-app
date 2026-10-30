import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  listProfessionals,
  getProfessional,
  reactivateProfessional,
} from "@/lib/professionals/professionals-api";
import { ProfessionalsWorkspace } from "./professionals-workspace";
vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  listProfessionals: vi.fn(),
  createProfessional: vi.fn(),
  getProfessional: vi.fn(),
  updateProfessional: vi.fn(),
  deactivateProfessional: vi.fn(),
  reactivateProfessional: vi.fn(),
}));
const record = {
  id: "ana-id",
  name: "Ana",
  email: "ana@example.test",
  active: false,
};
beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "owner",
        establishment_id: "shop",
        role: "owner",
        email: "owner@example.test",
        password_change_required: false,
      },
    },
    transitioning: false,
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [record],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
});
test("oferece reativação somente para inativo e permite cancelar limpando a senha", async () => {
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Reativar Ana" }));
  fireEvent.change(screen.getByLabelText("Senha temporária para reativação"), {
    target: { value: "senha-temporaria" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Cancelar reativação" }));
  fireEvent.click(screen.getByRole("button", { name: "Reativar Ana" }));
  expect(screen.getByLabelText("Senha temporária para reativação")).toHaveValue(
    "",
  );
  expect(reactivateProfessional).not.toHaveBeenCalled();
});

function result(active: boolean) {
  return {
    state: "success" as const,
    status: 200,
    data: { ...record, active },
  };
}
function failure(status: number, code: string) {
  return {
    state: "error" as const,
    status,
    error: { code, message: "privado", request_id: "id" },
  };
}
async function openPanel() {
  const rendered = render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Reativar Ana" }));
  return rendered;
}
function enterPassword(value = "senha-temporaria") {
  fireEvent.change(screen.getByLabelText("Senha temporária para reativação"), {
    target: { value },
  });
}
function confirm() {
  fireEvent.click(screen.getByRole("button", { name: "Confirmar reativação" }));
}

test("valida senha sem comparação com a anterior e confirma somente após resposta", async () => {
  let finish!: (value: ReturnType<typeof result>) => void;
  vi.mocked(reactivateProfessional).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  await openPanel();
  confirm();
  expect(
    screen.getByLabelText("Senha temporária para reativação"),
  ).toHaveFocus();
  expect(reactivateProfessional).not.toHaveBeenCalled();
  enterPassword("curta");
  confirm();
  expect(reactivateProfessional).not.toHaveBeenCalled();
  enterPassword();
  const form = screen
    .getByRole("button", { name: "Confirmar reativação" })
    .closest("form")!;
  act(() => {
    fireEvent.submit(form);
    fireEvent.submit(form);
  });
  expect(reactivateProfessional).toHaveBeenCalledExactlyOnceWith(
    "ana-id",
    { temporary_password: "senha-temporaria" },
    expect.any(AbortSignal),
  );
  expect(screen.queryByText("Profissional reativado")).not.toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  await act(async () => finish(result(true)));
  expect(screen.getByText("Profissional reativado")).toBeInTheDocument();
  expect(
    screen.getByText(/troca de senha será obrigatória/i),
  ).toBeInTheDocument();
  expect(
    screen.queryByLabelText("Senha temporária para reativação"),
  ).not.toBeInTheDocument();
  expect(screen.getByRole("list")).toHaveTextContent("Ativo");
  expect(
    screen.queryByRole("button", { name: "Reativar Ana" }),
  ).not.toBeInTheDocument();
});

test.each([true, false])(
  "reconcilia resultado incerto com active=%s sem afirmar senha aplicada",
  async (active) => {
    vi.mocked(reactivateProfessional).mockRejectedValue(new Error("network"));
    vi.mocked(getProfessional).mockResolvedValue(result(active));
    await openPanel();
    enterPassword();
    confirm();
    await screen.findByText("Não foi possível confirmar a reativação");
    expect(
      screen.getByLabelText("Senha temporária para reativação"),
    ).toHaveValue("senha-temporaria");
    expect(
      screen.getByRole("button", { name: "Confirmar reativação" }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Cancelar reativação" }),
    ).toBeDisabled();
    fireEvent.click(
      screen.getByRole("button", {
        name: "Consultar cadastro do profissional",
      }),
    );
    if (active) {
      await screen.findByText("Cadastro ativo; senha não confirmada");
      expect(
        screen.getByText(/não informa qual senha foi aplicada/i),
      ).toBeInTheDocument();
      expect(
        screen.queryByLabelText("Senha temporária para reativação"),
      ).not.toBeInTheDocument();
    } else
      await waitFor(() =>
        expect(
          screen.getByLabelText("Senha temporária para reativação"),
        ).toHaveValue(""),
      );
    expect(
      screen.queryByText("Profissional reativado"),
    ).not.toBeInTheDocument();
    expect(reactivateProfessional).toHaveBeenCalledOnce();
    expect(getProfessional).toHaveBeenCalledExactlyOnceWith(
      "ana-id",
      expect.any(AbortSignal),
    );
  },
);

test("conflito não confirma credencial e exige consulta sem repetir a reativação", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue(
    failure(409, "professional_already_active"),
  );
  vi.mocked(getProfessional).mockResolvedValue(result(true));
  await openPanel();
  enterPassword();
  confirm();
  await screen.findByText("Profissional já está ativo");
  expect(
    screen.getByText(/Esta tentativa não alterou a senha nem as sessões/),
  ).toBeInTheDocument();
  expect(screen.getByLabelText("Senha temporária para reativação")).toHaveValue(
    "",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar cadastro do profissional" }),
  );
  await screen.findByText("Cadastro ativo; senha não confirmada");
  expect(screen.queryByText("Profissional reativado")).not.toBeInTheDocument();
  expect(reactivateProfessional).toHaveBeenCalledOnce();
  expect(screen.queryByText("privado")).not.toBeInTheDocument();
});

test("sincroniza lista e editor aberto preservando nome editado", async () => {
  vi.mocked(getProfessional)
    .mockResolvedValueOnce(result(false))
    .mockResolvedValue(result(true));
  vi.mocked(reactivateProfessional).mockResolvedValue(result(true));
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Ana" }));
  await waitFor(() =>
    expect(screen.getByLabelText("Nome do profissional")).toBeEnabled(),
  );
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Reativar Ana" }));
  enterPassword();
  confirm();
  await screen.findByText("Estado: Ativo");
  expect(screen.getByLabelText("Nome do profissional")).toHaveValue("Rascunho");
  expect(screen.getByRole("list")).not.toHaveTextContent("Rascunho");
  expect(getProfessional).toHaveBeenCalledTimes(2);
});

test("mantém senha só na memória enquanto consulta falha e limpa ao reconciliar", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue(
    failure(503, "internal_error"),
  );
  vi.mocked(getProfessional)
    .mockRejectedValueOnce(new Error("offline"))
    .mockResolvedValue(result(false));
  await openPanel();
  enterPassword();
  confirm();
  await screen.findByText("Não foi possível confirmar a reativação");
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar cadastro do profissional" }),
  );
  await screen.findByText("Não foi possível consultar o cadastro");
  expect(screen.getByLabelText("Senha temporária para reativação")).toHaveValue(
    "senha-temporaria",
  );
  expect(
    screen.getByRole("button", { name: "Confirmar reativação" }),
  ).toBeDisabled();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar cadastro do profissional" }),
  );
  await waitFor(() =>
    expect(
      screen.getByLabelText("Senha temporária para reativação"),
    ).toHaveValue(""),
  );
  expect(reactivateProfessional).toHaveBeenCalledOnce();
});

test("read_only bloqueia demais escritas sem anunciar sucesso", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue(
    failure(403, "read_only"),
  );
  await openPanel();
  enterPassword();
  confirm();
  await screen.findByText("Modo somente leitura");
  expect(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Confirmar reativação" }),
  ).toBeDisabled();
  expect(screen.queryByText("Profissional reativado")).not.toBeInTheDocument();
});

test("401 limpa conteúdo protegido sem outra consulta de sessão", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue(
    failure(401, "authentication_required"),
  );
  await openPanel();
  enterPassword();
  confirm();
  await waitFor(() =>
    expect(screen.queryByRole("list")).not.toBeInTheDocument(),
  );
  expect(
    screen.queryByLabelText("Senha temporária para reativação"),
  ).not.toBeInTheDocument();
  expect(getProfessional).not.toHaveBeenCalled();
  expect(listProfessionals).toHaveBeenCalledOnce();
});

test("troca de identidade cancela envio e não publica confirmação antiga", async () => {
  let finish!: (value: ReturnType<typeof result>) => void;
  let signal!: AbortSignal;
  vi.mocked(reactivateProfessional).mockImplementation(
    (_id, _body, currentSignal) =>
      new Promise((resolve) => {
        finish = resolve;
        signal = currentSignal!;
      }),
  );
  const rendered = await openPanel();
  enterPassword();
  confirm();
  const session = vi.mocked(useAuthSession).mock.results.at(-1)!
    .value as ReturnType<typeof useAuthSession>;
  if (session.state.status !== "authenticated")
    throw new Error("fixture not authenticated");
  vi.mocked(useAuthSession).mockReturnValue({
    ...session,
    state: {
      ...session.state,
      principal: {
        ...session.state.principal,
        user_id: "other",
        establishment_id: "other-shop",
      },
    },
  });
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [{ ...record, id: "other-id", name: "Outra equipe" }],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
  rendered.rerender(<ProfessionalsWorkspace />);
  await screen.findByText("Outra equipe");
  expect(signal.aborted).toBe(true);
  await act(async () => finish(result(true)));
  expect(screen.queryByText("Profissional reativado")).not.toBeInTheDocument();
  expect(
    screen.queryByLabelText("Senha temporária para reativação"),
  ).not.toBeInTheDocument();
});

test("descarta contexto do conflito anterior antes de nova tentativa incerta", async () => {
  vi.mocked(reactivateProfessional)
    .mockResolvedValueOnce(failure(409, "professional_already_active"))
    .mockRejectedValueOnce(new Error("network"));
  vi.mocked(getProfessional)
    .mockResolvedValueOnce(result(false))
    .mockResolvedValueOnce(result(true));
  await openPanel();
  enterPassword();
  confirm();
  await screen.findByText("Profissional já está ativo");
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar cadastro do profissional" }),
  );
  await waitFor(() =>
    expect(
      screen.getByLabelText("Senha temporária para reativação"),
    ).toBeEnabled(),
  );
  enterPassword("outra-temporaria");
  confirm();
  await screen.findByText("Não foi possível confirmar a reativação");
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar cadastro do profissional" }),
  );
  await screen.findByText("Cadastro ativo; senha não confirmada");
  expect(
    screen.getByText(/não informa qual senha foi aplicada/i),
  ).toBeInTheDocument();
  expect(
    screen.queryByText(/Esta tentativa não alterou sua senha/),
  ).not.toBeInTheDocument();
  expect(reactivateProfessional).toHaveBeenCalledTimes(2);
});
