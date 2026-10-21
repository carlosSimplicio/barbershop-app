import { render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import Home from "./page";

afterEach(() => vi.restoreAllMocks());

test("apresenta a plataforma e encaminha para cadastro e login sem consultar a API", () => {
  const fetchMock = vi.spyOn(globalThis, "fetch");

  render(<Home />);

  expect(
    screen.getByRole("heading", {
      level: 1,
      name: /mais tempo para cuidar/i,
    }),
  ).toBeInTheDocument();
  expect(screen.getByText(/reúna agenda e finanças/i)).toBeInTheDocument();
  expect(
    screen.getByRole("link", { name: /criar minha conta/i }),
  ).toHaveAttribute("href", "/cadastro");
  expect(screen.getByRole("link", { name: /entrar/i })).toHaveAttribute(
    "href",
    "/login",
  );
  expect(fetchMock).not.toHaveBeenCalled();
});
