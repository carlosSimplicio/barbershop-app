import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { MAX_PAGE_OFFSET } from "@/lib/api/pagination";
import {
  RecordListPanel,
  type RecordListPanelProps,
} from "./record-list-panel";

type Entry = { id: string; name: string };

function propsFor(
  overrides: Partial<RecordListPanelProps<Entry>> = {},
): RecordListPanelProps<Entry> {
  return {
    title: "Registros cadastrados",
    label: "Lista de registros",
    paginationLabel: "Paginação de registros",
    view: {
      status: "success",
      page: {
        items: [{ id: "example", name: "Registro de teste" }],
        limit: 2,
        offset: 2,
        has_more: true,
      },
    },
    busy: false,
    messages: {
      loading: "Carregando registros…",
      empty: "Nenhum registro cadastrado.",
      errorTitle: "Não foi possível consultar os registros",
      errorDescription: "Tente novamente mais tarde.",
    },
    getKey: (record) => record.id,
    renderItem: (record) => <strong>{record.name}</strong>,
    onRefresh: vi.fn(),
    onPageChange: vi.fn(),
    ...overrides,
  };
}

test("substitui a lista por carregamento e erro, com nova tentativa coordenada", () => {
  const props = propsFor();
  const view = render(<RecordListPanel {...props} />);
  expect(screen.getByRole("list")).toHaveTextContent("Registro de teste");
  view.rerender(<RecordListPanel {...props} view={{ status: "loading" }} />);
  expect(
    screen.getByRole("region", { name: "Lista de registros" }),
  ).toHaveAttribute("aria-busy", "true");
  expect(screen.getByRole("status")).toHaveTextContent("Carregando registros…");
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  view.rerender(<RecordListPanel {...props} view={{ status: "error" }} busy />);
  expect(screen.getByRole("alert")).toHaveTextContent(
    "Não foi possível consultar os registros",
  );
  expect(
    screen.getByRole("button", { name: "Tentar novamente" }),
  ).toBeDisabled();
  view.rerender(<RecordListPanel {...props} view={{ status: "error" }} />);
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  expect(props.onRefresh).toHaveBeenCalledOnce();
});

test("permite focar o painel e a área de itens e encaminha a atualização manual", () => {
  const ref = createRef<HTMLElement>();
  const props = propsFor({ ref });
  render(<RecordListPanel {...props} />);
  ref.current?.focus();
  expect(
    screen.getByRole("region", { name: "Lista de registros" }),
  ).toHaveFocus();
  const items = screen.getByRole("region", { name: "Registros cadastrados" });
  items.focus();
  expect(items).toHaveFocus();
  fireEvent.click(screen.getByRole("button", { name: "Atualizar lista" }));
  expect(props.onRefresh).toHaveBeenCalledOnce();
});

test("usa o tamanho da página recebida para calcular avanço, retorno e número da página", () => {
  const props = propsFor();
  render(<RecordListPanel {...props} />);
  expect(screen.getByText("Página 2")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Anterior" }));
  expect(props.onPageChange).toHaveBeenLastCalledWith(0);
  fireEvent.click(screen.getByRole("button", { name: "Próxima" }));
  expect(props.onPageChange).toHaveBeenLastCalledWith(4);
});

test.each([
  {
    offset: 0,
    has_more: true,
    busy: false,
    previousDisabled: true,
    nextDisabled: false,
  },
  {
    offset: 2,
    has_more: false,
    busy: false,
    previousDisabled: false,
    nextDisabled: true,
  },
  {
    offset: 2,
    has_more: true,
    busy: true,
    previousDisabled: true,
    nextDisabled: true,
  },
  {
    offset: MAX_PAGE_OFFSET - 1,
    has_more: true,
    busy: false,
    previousDisabled: false,
    nextDisabled: true,
  },
])(
  "respeita fronteiras de paginação e operações pendentes: %j",
  ({ offset, has_more, busy, previousDisabled, nextDisabled }) => {
    const props = propsFor({
      busy,
      view: {
        status: "success",
        page: { items: [], limit: 2, offset, has_more },
      },
    });
    render(<RecordListPanel {...props} />);
    expect(
      screen.getByRole("button", { name: "Anterior" }).hasAttribute("disabled"),
    ).toBe(previousDisabled);
    expect(
      screen.getByRole("button", { name: "Próxima" }).hasAttribute("disabled"),
    ).toBe(nextDisabled);
  },
);

test("anuncia lista vazia sem itens fictícios e mantém a paginação visível com botões desabilitados", () => {
  const props = propsFor({
    view: {
      status: "success",
      page: { items: [], limit: 20, offset: 0, has_more: false },
    },
  });
  render(<RecordListPanel {...props} />);
  expect(screen.getByRole("status")).toHaveTextContent(
    "Nenhum registro cadastrado.",
  );
  expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
  expect(
    screen.getByRole("navigation", { name: "Paginação de registros" }),
  ).toBeVisible();
  expect(screen.getByText("Página 1")).toBeVisible();
  expect(screen.getByRole("button", { name: "Anterior" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Próxima" })).toBeDisabled();
});

test("retira conteúdo e ações quando a sessão deixa de estar autorizada", () => {
  const props = propsFor();
  const view = render(<RecordListPanel {...props} />);
  view.rerender(
    <RecordListPanel {...props} view={{ status: "unauthorized" }} />,
  );
  expect(screen.queryByRole("region")).not.toBeInTheDocument();
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
