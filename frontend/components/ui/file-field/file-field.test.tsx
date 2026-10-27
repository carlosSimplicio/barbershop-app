import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { FileField } from "./file-field";

test("associa mensagens acessíveis e permite selecionar imagens aceitas", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLInputElement>();
  const image = new File(["image bytes"], "logo.png", { type: "image/png" });

  render(
    <>
      <p id="external-description">O logo aparece na página pública.</p>
      <FileField
        aria-describedby="external-description"
        error="Escolha uma imagem compatível."
        hint="PNG, JPEG ou WebP, até 2 MiB e 4.096 × 4.096 pixels."
        id="logo"
        label="Logo do estabelecimento"
        onChange={handleChange}
        ref={ref}
      />
    </>,
  );

  const input = screen.getByLabelText<HTMLInputElement>(
    "Logo do estabelecimento",
  );
  expect(input).toHaveAttribute("type", "file");
  expect(input).toHaveAttribute("accept", "image/png,image/jpeg,image/webp");
  expect(screen.getByText("Escolher arquivo")).toBeInTheDocument();
  expect(screen.queryByText("Choose File")).not.toBeInTheDocument();
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAccessibleDescription(
    "O logo aparece na página pública. PNG, JPEG ou WebP, até 2 MiB e 4.096 × 4.096 pixels. Escolha uma imagem compatível.",
  );
  expect(ref.current).toBe(input);

  fireEvent.change(input, { target: { files: [image] } });

  expect(input.files?.[0]).toBe(image);
  expect(screen.getByText("logo.png")).toBeInTheDocument();
  expect(handleChange).toHaveBeenCalledOnce();
});

test("preserva o estado desabilitado", () => {
  render(<FileField disabled label="Envio indisponível" />);

  expect(screen.getByLabelText("Envio indisponível")).toBeDisabled();
  expect(screen.getByText("Escolher arquivo")).toBeInTheDocument();
});
