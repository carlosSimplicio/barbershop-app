import { render, screen, within } from "@testing-library/react";
import { createRef } from "react";
import { expect, test } from "vitest";

import { Feedback } from "./feedback";

test.each([
  ["success", "status"],
  ["warning", "status"],
  ["error", "alert"],
] as const)(
  "expõe feedback %s com a semântica acessível apropriada",
  (variant, role) => {
    render(
      <Feedback title="Solicitação concluída" variant={variant}>
        Você já pode continuar.
      </Feedback>,
    );

    const feedback = screen.getByRole(role);

    expect(feedback).toHaveTextContent("Solicitação concluída");
    expect(feedback).toHaveTextContent("Você já pode continuar.");
    expect(feedback).toHaveAccessibleName("Solicitação concluída");
    expect(feedback).toHaveAccessibleDescription("Você já pode continuar.");
    expect(feedback).toHaveAttribute("aria-atomic", "true");
    expect(feedback).toHaveAttribute("data-variant", variant);
  },
);

test("permite ação relacionada, atributos nativos e ref", () => {
  const ref = createRef<HTMLDivElement>();

  render(
    <Feedback
      action={<button type="button">Tentar novamente</button>}
      data-context="cadastro"
      ref={ref}
      title="Não foi possível concluir"
      variant="error"
    >
      Confira sua conexão.
    </Feedback>,
  );

  const feedback = screen.getByRole("alert");

  expect(
    within(feedback).getByRole("button", { name: "Tentar novamente" }),
  ).toBeVisible();
  expect(feedback).toHaveAttribute("data-context", "cadastro");
  expect(ref.current).toBe(feedback);
});
