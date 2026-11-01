import { fireEvent, render, screen } from "@testing-library/react";
import { createRef, StrictMode, useRef, useState } from "react";
import { expect, test, vi } from "vitest";

import { Modal } from "./modal";

function Example({ blocked = false }: { blocked?: boolean }) {
  const [open, setOpen] = useState(false);
  const fallback = useRef<HTMLHeadingElement>(null);
  return (
    <>
      <h1 ref={fallback} tabIndex={-1}>
        Registros
      </h1>
      <button disabled={open && blocked} onClick={() => setOpen(true)}>
        Adicionar
      </button>
      <Modal
        open={open}
        title="Novo registro"
        description="Preencha os campos."
        onClose={() => setOpen(false)}
        closeDisabled={blocked}
        fallbackFocusRef={fallback}
      >
        <input aria-label="Nome" />
        <button disabled>Indisponível</button>
        <div hidden>
          <button>Oculto</button>
        </div>
      </Modal>
    </>
  );
}

test("abre com nome e descrição acessíveis, foca o título e preserva rascunho ao reabrir", () => {
  render(
    <StrictMode>
      <Example />
    </StrictMode>,
  );
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  const origin = screen.getByRole("button", { name: "Adicionar" });
  origin.focus();
  fireEvent.click(origin);
  expect(
    screen.getByRole("dialog", { name: "Novo registro" }),
  ).toHaveAccessibleDescription("Preencha os campos.");
  expect(screen.getByRole("heading", { name: "Novo registro" })).toHaveFocus();
  fireEvent.change(screen.getByLabelText("Nome"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Fechar modal" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(origin).toHaveFocus();
  fireEvent.click(origin);
  expect(screen.getByLabelText("Nome")).toHaveValue("Rascunho");
});

test("Escape fecha a modal e devolve foco ao acionador", () => {
  render(<Example />);
  const origin = screen.getByRole("button", { name: "Adicionar" });
  origin.focus();
  fireEvent.click(origin);
  fireEvent(
    screen.getByRole("dialog"),
    new Event("cancel", { cancelable: true }),
  );
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(origin).toHaveFocus();
});

test("Tab e Shift+Tab circulam nos controles disponíveis sem escapar para fora da modal", () => {
  // jsdom has no layout; provide rectangles only to enable our visibility filter.
  const layout = vi
    .spyOn(HTMLElement.prototype, "getClientRects")
    .mockReturnValue([new DOMRect()] as unknown as DOMRectList);
  try {
    render(<Example />);
    fireEvent.click(screen.getByRole("button", { name: "Adicionar" }));
    const first = screen.getByRole("button", { name: "Fechar modal" });
    const last = screen.getByLabelText("Nome");
    fireEvent.keyDown(document.activeElement!, { key: "Tab", shiftKey: true });
    expect(last).toHaveFocus();
    fireEvent.keyDown(last, { key: "Tab" });
    expect(first).toHaveFocus();
    fireEvent.keyDown(first, { key: "Tab", shiftKey: true });
    expect(last).toHaveFocus();
  } finally {
    layout.mockRestore();
  }
});

test("bloqueia Escape e fechar durante envio", () => {
  const view = render(<Example blocked />);
  const origin = screen.getByRole("button", { name: "Adicionar" });
  origin.focus();
  fireEvent.click(origin);
  expect(screen.getByRole("button", { name: "Fechar modal" })).toBeDisabled();
  for (let attempt = 0; attempt < 2; attempt++) {
    expect(
      fireEvent.keyDown(
        attempt === 0 ? screen.getByRole("dialog") : document.body,
        {
          key: "Escape",
          cancelable: true,
        },
      ),
    ).toBe(false);
  }
  fireEvent(
    screen.getByRole("dialog"),
    new Event("cancel", { cancelable: true }),
  );
  expect(screen.getByRole("dialog")).toBeVisible();
  view.unmount();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(
    fireEvent.keyDown(document.body, { key: "Escape", cancelable: true }),
  ).toBe(true);
});

test("devolve foco ao destino alternativo se a origem estiver desabilitada ao fechar", () => {
  const fallback = createRef<HTMLHeadingElement>();
  function content(open: boolean) {
    return (
      <>
        <h1 ref={fallback} tabIndex={-1}>
          Lista
        </h1>
        <button>Adicionar</button>
        <Modal
          open={open}
          title="Novo registro"
          onClose={() => {}}
          fallbackFocusRef={fallback}
        >
          Conteúdo
        </Modal>
      </>
    );
  }
  const view = render(content(false));
  const origin = screen.getByRole("button", { name: "Adicionar" });
  origin.focus();
  view.rerender(content(true));
  origin.setAttribute("disabled", "");
  view.rerender(content(false));
  expect(screen.getByRole("heading", { name: "Lista" })).toHaveFocus();
});
