import { render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { HealthStatus } from "./health-status";

afterEach(() => {
  vi.restoreAllMocks();
});

test("informa quando o frontend se conecta a API", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ status: "ok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );

  render(<HealthStatus />);

  expect(
    await screen.findByText("Frontend e API conectados"),
  ).toBeInTheDocument();
});
