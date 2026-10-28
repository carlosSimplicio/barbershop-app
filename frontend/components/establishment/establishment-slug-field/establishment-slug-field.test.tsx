import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { EstablishmentSlugField } from "./establishment-slug-field";

test("adds a slash prefix without including it in the slug value", () => {
  const handleChange = vi.fn();

  render(
    <EstablishmentSlugField
      disabled
      error="Endereço indisponível."
      hint="O slug não pode ser alterado."
      id="slug"
      label="Slug"
      onChange={handleChange}
      value="estudio-aurora"
    />,
  );

  const input = screen.getByRole("textbox", { name: "Slug" });
  const prefix = screen.getByText("/", { selector: "span" });

  expect(input).toHaveValue("estudio-aurora");
  expect(input).toBeDisabled();
  expect(prefix).toHaveAttribute("aria-hidden", "true");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAttribute("aria-describedby", "slug-hint slug-error");
  expect(screen.getByText("Endereço indisponível.")).toBeVisible();

  fireEvent.change(input, { target: { value: "novo-slug" } });
  expect(handleChange).toHaveBeenCalledOnce();
});
