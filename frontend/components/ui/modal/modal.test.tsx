import { fireEvent, render, screen, within } from "@testing-library/react";
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

function NestedExample({ blocked = false }: { blocked?: boolean }) {
  const [confirming, setConfirming] = useState(false);
  const [editing, setEditing] = useState(true);
  const trigger = useRef<HTMLButtonElement>(null);
  return (
    <Modal
      open={editing}
      title="Editar registro"
      onClose={() => setEditing(false)}
      closeDisabled={blocked}
    >
      <input aria-label="Rascunho" defaultValue="Nome editado" />
      <button
        ref={trigger}
        disabled={confirming}
        onClick={() => setConfirming(true)}
      >
        Alterar estado
      </button>
      {confirming ? (
        <Modal
          open
          returnFocusRef={trigger}
          title="Confirmar estado"
          onClose={() => setConfirming(false)}
          closeDisabled={blocked}
        >
          <button disabled={blocked}>Confirmar estado</button>
          <button disabled={blocked} onClick={() => setConfirming(false)}>
            Cancelar confirmação
          </button>
        </Modal>
      ) : null}
    </Modal>
  );
}

test("Escape fecha somente a confirmação e preserva editor, rascunho e retorno do foco", () => {
  render(<NestedExample />);
  const trigger = screen.getByRole("button", { name: "Alterar estado" });
  fireEvent.click(trigger);
  const confirmation = screen.getByRole("dialog", { name: "Confirmar estado" });
  expect(
    within(confirmation).getByRole("heading", { name: "Confirmar estado" }),
  ).toHaveFocus();
  fireEvent(
    confirmation,
    new Event("cancel", { cancelable: true, bubbles: true }),
  );
  expect(
    screen.queryByRole("dialog", { name: "Confirmar estado" }),
  ).not.toBeInTheDocument();
  expect(screen.getByRole("dialog", { name: "Editar registro" })).toBeVisible();
  expect(screen.getByLabelText("Rascunho")).toHaveValue("Nome editado");
  expect(trigger).toHaveFocus();
});

test("Tab circula somente na confirmação sem ser capturado pelo editor abaixo", () => {
  const layout = vi
    .spyOn(HTMLElement.prototype, "getClientRects")
    .mockReturnValue([new DOMRect()] as unknown as DOMRectList);
  try {
    render(<NestedExample />);
    fireEvent.click(screen.getByRole("button", { name: "Alterar estado" }));
    const confirmation = screen.getByRole("dialog", {
      name: "Confirmar estado",
    });
    const first = within(confirmation).getByRole("button", {
      name: "Fechar modal",
    });
    const last = within(confirmation).getByRole("button", {
      name: "Cancelar confirmação",
    });
    last.focus();
    fireEvent.keyDown(last, { key: "Tab" });
    expect(first).toHaveFocus();
    fireEvent.keyDown(first, { key: "Tab", shiftKey: true });
    expect(last).toHaveFocus();
  } finally {
    layout.mockRestore();
  }
});

test("confirmação pendente protege Escape e recupera foco nela ao pressionar Tab fora do dialog", () => {
  render(<NestedExample blocked />);
  fireEvent.click(screen.getByRole("button", { name: "Alterar estado" }));
  const confirmation = screen.getByRole("dialog", { name: "Confirmar estado" });
  expect(
    within(confirmation).getByRole("button", { name: "Fechar modal" }),
  ).toBeDisabled();
  document.body.setAttribute("tabindex", "-1");
  document.body.focus();
  expect(document.body).toHaveFocus();
  fireEvent.keyDown(document.body, { key: "Tab", cancelable: true });
  document.body.removeAttribute("tabindex");
  expect(
    within(confirmation).getByRole("heading", { name: "Confirmar estado" }),
  ).toHaveFocus();
  expect(
    fireEvent.keyDown(document.body, { key: "Escape", cancelable: true }),
  ).toBe(false);
  fireEvent(
    confirmation,
    new Event("cancel", { cancelable: true, bubbles: true }),
  );
  expect(screen.getByRole("dialog", { name: "Editar registro" })).toBeVisible();
  expect(confirmation).toBeVisible();
});

test("destino explícito removido retorna ao fallback ao desmontar a modal", () => {
  function Example() {
    const [open, setOpen] = useState(false);
    const [removed, setRemoved] = useState(false);
    const target = useRef<HTMLButtonElement>(null);
    const fallback = useRef<HTMLHeadingElement>(null);
    return (
      <>
        <h1 ref={fallback} tabIndex={-1}>
          Lista
        </h1>
        {!removed ? (
          <button
            ref={target}
            onClick={() => {
              setRemoved(true);
              setOpen(true);
            }}
          >
            Editar
          </button>
        ) : null}
        {open ? (
          <Modal
            open
            title="Edição"
            returnFocusRef={target}
            fallbackFocusRef={fallback}
            onClose={() => setOpen(false)}
          >
            <button onClick={() => setOpen(false)}>Concluir</button>
          </Modal>
        ) : null}
      </>
    );
  }
  render(<Example />);
  fireEvent.click(screen.getByRole("button", { name: "Editar" }));
  fireEvent.click(screen.getByRole("button", { name: "Concluir" }));
  expect(screen.getByRole("heading", { name: "Lista" })).toHaveFocus();
});
