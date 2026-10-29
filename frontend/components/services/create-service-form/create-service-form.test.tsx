import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { createService } from "@/lib/services/services-api";
import { CreateServiceForm } from "./create-service-form";
vi.mock("@/lib/services/services-api", () => ({ createService: vi.fn() }));
beforeEach(() => {
  vi.mocked(createService).mockReset();
});
function fill() {
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Corte" },
  });
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "0,01" },
  });
}
function form(canWrite = true) {
  const props = {
    canWrite,
    onCreated: vi.fn(),
    onReadOnly: vi.fn(),
    onUnauthorized: vi.fn(),
  };
  render(<CreateServiceForm {...props} />);
  return props;
}
test("exige campos e foca o primeiro erro", () => {
  form();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  expect(screen.getByLabelText("Nome do serviço")).toHaveFocus();
  expect(createService).not.toHaveBeenCalled();
});
test("envia uma vez, aplica 30 minutos pelo valor nulo e confirma criação", async () => {
  vi.mocked(createService).mockResolvedValue({
    state: "success",
    status: 201,
    data: {
      id: "id",
      name: "Corte",
      price_cents: 1,
      duration_minutes: 30,
      active: true,
    },
  });
  const props = form();
  fill();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  expect(await screen.findByText("Serviço cadastrado")).toBeVisible();
  expect(props.onCreated).toHaveBeenCalledOnce();
  expect(createService).toHaveBeenCalledWith(
    { name: "Corte", price_cents: 1, duration_minutes: null },
    expect.any(AbortSignal),
  );
});
test("preserva dados e mostra erros locais por campo", async () => {
  vi.mocked(createService).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "raw",
      request_id: "test",
      fields: [{ field: "name", code: "invalid_name", message: "raw" }],
    },
  });
  form();
  fill();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  await waitFor(() =>
    expect(screen.getByLabelText("Nome do serviço")).toHaveAttribute(
      "aria-invalid",
      "true",
    ),
  );
  expect(screen.getByLabelText("Nome do serviço")).toHaveValue("Corte");
  expect(screen.queryByText("raw")).not.toBeInTheDocument();
});
test("não envia com somente leitura e informa expiração recebida", async () => {
  const props = form(false);
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeDisabled();
  expect(createService).not.toHaveBeenCalled();
  expect(props.onReadOnly).not.toHaveBeenCalled();
});
test.each([
  null,
  {
    state: "error",
    status: 503,
    error: { code: "internal_error", message: "raw", request_id: "test" },
  } as const,
  { state: "empty", status: 201 } as const,
])("não anuncia sucesso nem repete resultado ambíguo %j", async (result) => {
  if (result) vi.mocked(createService).mockResolvedValue(result);
  else vi.mocked(createService).mockRejectedValue(new Error("raw"));
  form();
  fill();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  expect(
    await screen.findByText("Não foi possível confirmar o cadastro"),
  ).toBeVisible();
  expect(createService).toHaveBeenCalledOnce();
  expect(screen.queryByText("Serviço cadastrado")).not.toBeInTheDocument();
});

test("impede submissões simultâneas e informa read_only", async () => {
  let finish!: (value: Awaited<ReturnType<typeof createService>>) => void;
  vi.mocked(createService).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  const props = form();
  fill();
  const element = screen
    .getByRole("button", { name: "Cadastrar serviço" })
    .closest("form")!;
  fireEvent.submit(element);
  fireEvent.submit(element);
  expect(createService).toHaveBeenCalledOnce();
  finish({
    state: "error",
    status: 403,
    error: { code: "read_only", message: "raw", request_id: "test" },
  });
  await waitFor(() => expect(props.onReadOnly).toHaveBeenCalledOnce());
  expect(await screen.findByText("Cadastro indisponível")).toBeVisible();
});

test("foca preço inválido da API após habilitar os controles", async () => {
  vi.mocked(createService).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "raw",
      request_id: "test",
      fields: [{ field: "price_cents", code: "invalid_price", message: "raw" }],
    },
  });
  form();
  fill();
  screen.getByLabelText("Nome do serviço").focus();
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  await waitFor(() =>
    expect(screen.getByLabelText("Preço")).toHaveAttribute(
      "aria-invalid",
      "true",
    ),
  );
  expect(screen.getByLabelText("Preço")).toHaveFocus();
});
