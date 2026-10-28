import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import EstablishmentError from "./error";

test("requests a new server render when retrying a failed public profile", () => {
  const retry = vi.fn();
  const reset = vi.fn();

  render(<EstablishmentError {...{ retry, reset }} />);
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  expect(retry).toHaveBeenCalledOnce();
  expect(reset).not.toHaveBeenCalled();
});
