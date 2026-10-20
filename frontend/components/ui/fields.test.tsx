import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Field } from "./field";
import { Input } from "./input";
import { PasswordField } from "./password-field";
import { SelectField } from "./select-field";
import { TextField } from "./text-field";

test("permite compor um campo com controle próprio", () => {
  render(
    <Field
      error="Nome inválido."
      errorId="name-error"
      hint="Use o nome público."
      hintId="name-hint"
      htmlFor="name"
      label="Nome"
    >
      <Input
        aria-describedby="name-hint name-error"
        aria-invalid="true"
        id="name"
      />
    </Field>,
  );

  const input = screen.getByLabelText("Nome");

  expect(input).toHaveAccessibleDescription(
    "Use o nome público. Nome inválido.",
  );
  expect(input).toHaveAttribute("aria-invalid", "true");
});

test("associa automaticamente label, ajuda, erro e ref no campo de texto", () => {
  const ref = createRef<HTMLInputElement>();

  render(
    <>
      <p id="external-description">Informação externa.</p>
      <TextField
        aria-describedby="external-description"
        error="Digite um e-mail válido."
        hint="Use seu e-mail principal."
        id="email"
        label="E-mail"
        ref={ref}
      />
    </>,
  );

  const input = screen.getByLabelText("E-mail");

  expect(input).toHaveAttribute("id", "email");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAccessibleDescription(
    "Informação externa. Use seu e-mail principal. Digite um e-mail válido.",
  );
  expect(ref.current).toBe(input);
});

test("gera um id estável quando o campo de texto não recebe id", () => {
  const { rerender } = render(<TextField label="Estabelecimento" />);

  const input = screen.getByLabelText("Estabelecimento");
  const generatedId = input.getAttribute("id");

  expect(input).toHaveAttribute("id");
  expect(generatedId).not.toBe("");

  rerender(<TextField label="Estabelecimento" />);

  expect(screen.getByLabelText("Estabelecimento")).toHaveAttribute(
    "id",
    generatedId,
  );
});

test("oculta o wrapper completo do campo de texto", () => {
  const { container } = render(<TextField hidden label="Campo oculto" />);

  expect(screen.queryByRole("textbox", { name: "Campo oculto" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

test("associa o campo de seleção e preserva sua interação", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLSelectElement>();

  render(
    <SelectField
      disabled
      label="Fuso horário"
      onChange={handleChange}
      ref={ref}
    >
      <option value="America/Sao_Paulo">São Paulo</option>
      <option value="America/Manaus">Manaus</option>
    </SelectField>,
  );

  const select = screen.getByLabelText("Fuso horário");

  expect(select).toBeDisabled();
  expect(ref.current).toBe(select);
});

test("preserva a interação do campo de seleção habilitado", () => {
  const handleChange = vi.fn();

  render(
    <SelectField label="Fuso horário" onChange={handleChange}>
      <option value="America/Sao_Paulo">São Paulo</option>
      <option value="America/Manaus">Manaus</option>
    </SelectField>,
  );

  const select = screen.getByLabelText("Fuso horário");
  fireEvent.change(select, { target: { value: "America/Manaus" } });

  expect(select).toHaveValue("America/Manaus");
  expect(handleChange).toHaveBeenCalledOnce();
});

test("alterna a visibilidade da senha com mensagens e callback configuráveis", () => {
  const handleVisibilityChange = vi.fn();
  const ref = createRef<HTMLInputElement>();

  render(
    <PasswordField
      className="password-control"
      defaultValue="senhasegura"
      hidePasswordLabel="Esconder conteúdo"
      label="Senha"
      onVisibilityChange={handleVisibilityChange}
      ref={ref}
      showPasswordLabel="Revelar conteúdo"
    />,
  );

  const input = screen.getByLabelText("Senha");
  const showButton = screen.getByRole("button", {
    name: "Revelar conteúdo",
  });

  expect(input).toHaveAttribute("type", "password");
  expect(input).toHaveClass("password-control");
  expect(ref.current).toBe(input);
  fireEvent.click(showButton);

  expect(input).toHaveAttribute("type", "text");
  expect(input).toHaveValue("senhasegura");
  expect(handleVisibilityChange).toHaveBeenLastCalledWith(true);

  fireEvent.click(screen.getByRole("button", { name: "Esconder conteúdo" }));

  expect(input).toHaveAttribute("type", "password");
  expect(handleVisibilityChange).toHaveBeenLastCalledWith(false);
});

test("desabilita o controle e a ação de visibilidade da senha", () => {
  render(<PasswordField disabled label="Senha indisponível" />);

  expect(screen.getByLabelText("Senha indisponível")).toBeDisabled();
  expect(screen.getByRole("button", { name: "Mostrar senha" })).toBeDisabled();
});
