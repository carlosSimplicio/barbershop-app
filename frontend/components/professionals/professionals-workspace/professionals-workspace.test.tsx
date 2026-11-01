import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  getProfessional,
  updateProfessional,
  createProfessional,
  listProfessionals,
} from "@/lib/professionals/professionals-api";

import { ProfessionalsWorkspace } from "./professionals-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  getProfessional: vi.fn(),
  updateProfessional: vi.fn(),
  createProfessional: vi.fn(),
  listProfessionals: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "owner-id",
        establishment_id: "shop-id",
        email: "owner@example.test",
        role: "owner",
        password_change_required: false,
      },
    },
    transitioning: false,
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
  vi.mocked(getProfessional).mockReset();
  vi.mocked(updateProfessional).mockReset();
  vi.mocked(createProfessional).mockReset();
  vi.mocked(listProfessionals).mockReset();
});

test("atualiza a lista após criar o profissional", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce(page([]))
    .mockResolvedValueOnce(
      page([
        {
          id: "professional-id",
          name: "Ana Lima",
          email: "ana@example.test",
          active: true,
        },
      ]),
    );
  vi.mocked(createProfessional).mockResolvedValue({
    state: "success",
    status: 201,
    data: {
      id: "professional-id",
      name: "Ana Lima",
      email: "ana@example.test",
      active: true,
    },
  });

  render(<ProfessionalsWorkspace />);
  await screen.findByText("Nenhum profissional cadastrado.");
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  const action = screen.getByRole("button", { name: "Adicionar profissional" });
  expect(action.closest("header")).toContainElement(
    screen.getByRole("heading", { name: "Profissionais", level: 1 }),
  );
  fireEvent.click(action);
  expect(
    screen.getByRole("dialog", { name: "Cadastrar profissional" }),
  ).toBeVisible();
  fireEvent.change(screen.getByLabelText("Nome"), {
    target: { value: "Ana Lima" },
  });
  fireEvent.change(screen.getByLabelText("E-mail"), {
    target: { value: "ana@example.test" },
  });
  fireEvent.change(screen.getByLabelText("Senha temporária"), {
    target: { value: "senha-segura" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar" }));

  expect(await screen.findByText("Ana Lima")).toBeInTheDocument();
  await waitFor(() => expect(listProfessionals).toHaveBeenCalledTimes(2));
  expect(screen.getByLabelText("Senha temporária")).toHaveValue("");
});

test("erro no cadastro permite fechar e reabrir sem conferir a lista", async () => {
  vi.mocked(createProfessional).mockRejectedValue(new Error("network"));
  vi.mocked(listProfessionals).mockResolvedValue(page([]));
  render(<ProfessionalsWorkspace />);
  await screen.findByText("Nenhum profissional cadastrado.");
  fillCreation();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar" }));
  await screen.findByText("Não foi possível cadastrar o profissional");
  expect(screen.getByRole("button", { name: "Cadastrar" })).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Fechar modal" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  );
  expect(screen.getByRole("button", { name: "Cadastrar" })).toBeEnabled();
  expect(createProfessional).toHaveBeenCalledOnce();
  expect(listProfessionals).toHaveBeenCalledOnce();
});

function page(
  items: Array<{
    id: string;
    name: string;
    email: string;
    active: boolean;
  }>,
) {
  return {
    state: "success" as const,
    status: 200,
    data: { items, limit: 20, offset: 0, has_more: false, can_write: true },
  };
}

test("bloqueia cadastro enquanto direitos estão pendentes ou recusados", async () => {
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: false,
    },
  });
  render(<ProfessionalsWorkspace />);
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
  await screen.findByText("Nenhum profissional cadastrado.");
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
});

const professional = {
  id: "ana-id",
  name: "Ana Lima",
  email: "ana@example.test",
  active: false,
};
function recordResult(record = professional) {
  return { state: "success" as const, status: 200, data: record };
}
function failure(status: number, code: string) {
  return {
    state: "error" as const,
    status,
    error: { code, message: "detalhe privado", request_id: "id" },
  };
}
async function openEditor() {
  vi.mocked(listProfessionals).mockResolvedValue(page([professional]));
  vi.mocked(getProfessional).mockResolvedValue(recordResult());
  const rendered = render(<ProfessionalsWorkspace />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Editar Ana Lima" }),
  );
  await waitFor(() =>
    expect(screen.getByLabelText("Nome do profissional")).toBeEnabled(),
  );
  return rendered;
}

test("consulta inativo, mantém e-mail somente leitura e cancela com foco na origem", async () => {
  await openEditor();
  expect(screen.getByLabelText("E-mail do profissional")).toHaveAttribute(
    "readonly",
  );
  expect(screen.getByText("Estado: Inativo")).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Cancelar" }));
  expect(updateProfessional).not.toHaveBeenCalled();
  expect(screen.getByRole("button", { name: "Editar Ana Lima" })).toHaveFocus();
});

test("envia apenas nome alterado uma vez e sincroniza lista com o registro persistido", async () => {
  await openEditor();
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  expect(updateProfessional).not.toHaveBeenCalled();
  let finish!: (value: ReturnType<typeof recordResult>) => void;
  vi.mocked(updateProfessional).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: " Beatriz " },
  });
  const form = screen.getByRole("button", { name: "Salvar" }).closest("form")!;
  act(() => {
    fireEvent.submit(form);
    fireEvent.submit(form);
  });
  expect(updateProfessional).toHaveBeenCalledExactlyOnceWith(
    "ana-id",
    { name: "Beatriz" },
    expect.any(AbortSignal),
  );
  expect(screen.getByRole("button", { name: "Fechar modal" })).toBeDisabled();
  fireEvent(
    screen.getByRole("dialog", { name: "Editar profissional" }),
    new Event("cancel", { cancelable: true }),
  );
  expect(
    screen.getByRole("dialog", { name: "Editar profissional" }),
  ).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
  await act(async () =>
    finish(recordResult({ ...professional, name: "Beatriz" })),
  );
  expect(screen.getByRole("list")).toHaveTextContent("Beatriz");
  expect(screen.getByLabelText("Nome do profissional")).toHaveValue("Beatriz");
});

test("erro de edição preserva rascunho sem atualizar a lista nem consultar o cadastro", async () => {
  await openEditor();
  vi.mocked(updateProfessional).mockRejectedValue(new Error("network"));
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  await screen.findByText("Não foi possível salvar o profissional");
  expect(screen.getByRole("button", { name: "Salvar" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Fechar modal" })).toBeEnabled();
  expect(screen.getByLabelText("Nome do profissional")).toHaveValue("Rascunho");
  expect(screen.getByRole("list")).not.toHaveTextContent("Rascunho");
  expect(getProfessional).toHaveBeenCalledOnce();
  expect(updateProfessional).toHaveBeenCalledOnce();
});

test("propaga read_only da edição a todas as escritas mantendo leitura", async () => {
  await openEditor();
  vi.mocked(updateProfessional).mockResolvedValue(failure(403, "read_only"));
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Beatriz" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  await screen.findByText("Modo somente leitura");
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
  expect(screen.getByLabelText("Nome do profissional")).toBeDisabled();
  expect(screen.getByRole("list")).toHaveTextContent("Ana Lima");
});

test("remove conteúdo protegido após 401 da edição sem consultar sessão novamente", async () => {
  await openEditor();
  vi.mocked(updateProfessional).mockResolvedValue(
    failure(401, "authentication_required"),
  );
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Beatriz" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  await waitFor(() =>
    expect(screen.queryByRole("list")).not.toBeInTheDocument(),
  );
  expect(screen.queryByLabelText("E-mail")).not.toBeInTheDocument();
  expect(getProfessional).toHaveBeenCalledOnce();
});

test("foca erro de campo local sem mostrar mensagens brutas do servidor", async () => {
  await openEditor();
  vi.mocked(updateProfessional).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "privado",
      request_id: "id",
      fields: [{ field: "name", code: "invalid_name", message: "privado" }],
    },
  });
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Beatriz" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  await waitFor(() =>
    expect(screen.getByLabelText("Nome do profissional")).toHaveFocus(),
  );
  expect(screen.getByLabelText("Nome do profissional")).toHaveAttribute(
    "aria-invalid",
    "true",
  );
  expect(screen.queryByText("privado")).not.toBeInTheDocument();
});

test("mantém sucesso do cadastro se a atualização da lista falhar", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce(page([]))
    .mockRejectedValueOnce(new Error("offline"));
  vi.mocked(createProfessional).mockResolvedValue({
    state: "success",
    status: 201,
    data: professional,
  });
  render(<ProfessionalsWorkspace />);
  await screen.findByText("Nenhum profissional cadastrado.");
  fillCreation();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar" }));
  await screen.findByText("Não foi possível consultar os profissionais");
  expect(screen.getByText("Profissional cadastrado")).toBeInTheDocument();
  expect(screen.getByLabelText("Senha temporária")).toHaveValue("");
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
});
function fillCreation() {
  fireEvent.click(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  );
  fireEvent.change(screen.getByLabelText("Nome"), { target: { value: "Ana" } });
  fireEvent.change(screen.getByLabelText("E-mail"), {
    target: { value: "ana@example.test" },
  });
  fireEvent.change(screen.getByLabelText("Senha temporária"), {
    target: { value: "senha-temporaria" },
  });
}

test("read_only recebido no cadastro bloqueia também editor", async () => {
  await openEditor();
  vi.mocked(createProfessional).mockResolvedValue(failure(403, "read_only"));
  fillCreation();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar" }));
  await screen.findByText("Modo somente leitura");
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
});

test("descarta PATCH obsoleto após troca de identidade e cancela a requisição", async () => {
  const rendered = await openEditor();
  let finish!: (value: ReturnType<typeof recordResult>) => void;
  let signal!: AbortSignal;
  vi.mocked(updateProfessional).mockImplementation(
    (_id, _body, currentSignal) =>
      new Promise((resolve) => {
        finish = resolve;
        signal = currentSignal!;
      }),
  );
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Antigo" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  const session = vi.mocked(useAuthSession).mock.results.at(-1)!
    .value as ReturnType<typeof useAuthSession>;
  if (session.state.status !== "authenticated")
    throw new Error("expected authenticated fixture");
  vi.mocked(useAuthSession).mockReturnValue({
    ...session,
    state: {
      ...session.state,
      principal: {
        ...session.state.principal,
        user_id: "other-owner",
        establishment_id: "other-shop",
      },
    },
  });
  vi.mocked(listProfessionals).mockResolvedValue(
    page([{ ...professional, id: "new-id", name: "Outra equipe" }]),
  );
  rendered.rerender(<ProfessionalsWorkspace />);
  await screen.findByText("Outra equipe");
  expect(signal.aborted).toBe(true);
  await act(async () =>
    finish(recordResult({ ...professional, name: "Antigo" })),
  );
  expect(screen.queryByText("Antigo")).not.toBeInTheDocument();
  expect(
    screen.queryByLabelText("Nome do profissional"),
  ).not.toBeInTheDocument();
});

test.each([500, 503])(
  "erro HTTP %i da edição libera nova tentativa sem consultar cadastro",
  async (status) => {
    await openEditor();
    vi.mocked(updateProfessional)
      .mockResolvedValueOnce(failure(status, "internal_error"))
      .mockResolvedValueOnce(
        recordResult({ ...professional, name: "Registro novo" }),
      );
    fireEvent.change(screen.getByLabelText("Nome do profissional"), {
      target: { value: "Registro novo" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
    await screen.findByText("Não foi possível salvar o profissional");
    expect(screen.getByRole("button", { name: "Salvar" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Fechar modal" })).toBeEnabled();
    expect(updateProfessional).toHaveBeenCalledOnce();
    expect(getProfessional).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
    await screen.findByText("Profissional atualizado");
    expect(screen.getByRole("list")).toHaveTextContent("Registro novo");
    expect(updateProfessional).toHaveBeenCalledTimes(2);
    expect(getProfessional).toHaveBeenCalledOnce();
  },
);

test("devolve foco após cancelar edição com erro", async () => {
  await openEditor();
  vi.mocked(updateProfessional).mockRejectedValue(new Error("network"));
  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
  await screen.findByText("Não foi possível salvar o profissional");
  fireEvent.click(screen.getByRole("button", { name: "Cancelar" }));
  expect(screen.getByRole("button", { name: "Editar Ana Lima" })).toHaveFocus();
});

test("o lápis abre edição em modal e as ações de estado usam o cadastro consultado", async () => {
  vi.mocked(listProfessionals).mockResolvedValue(
    page([{ ...professional, active: true }]),
  );
  vi.mocked(getProfessional).mockResolvedValue(recordResult(professional));
  render(<ProfessionalsWorkspace />);
  const list = await screen.findByRole("list", {
    name: "Profissionais cadastrados",
  });
  expect(
    within(list).queryByRole("button", { name: /Desativar|Reativar/ }),
  ).not.toBeInTheDocument();
  fireEvent.click(
    within(list).getByRole("button", { name: "Editar Ana Lima" }),
  );
  const modal = screen.getByRole("dialog", { name: "Editar profissional" });
  await waitFor(() =>
    expect(
      within(modal).getByRole("button", { name: "Reativar Ana Lima" }),
    ).toBeEnabled(),
  );
  expect(
    within(modal).getByRole("heading", { name: "Editar profissional" }),
  ).toHaveFocus();
  expect(
    within(modal).queryByRole("button", { name: "Desativar Ana Lima" }),
  ).not.toBeInTheDocument();
});
