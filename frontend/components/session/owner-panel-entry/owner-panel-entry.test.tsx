import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useRouter } from "next/navigation";

import { getEstablishmentSettings } from "@/lib/establishment/establishment-api";

import { OwnerPanelEntry } from "./owner-panel-entry";

vi.mock("next/navigation", () => ({ useRouter: vi.fn() }));
vi.mock("@/lib/establishment/establishment-api", () => ({
  getEstablishmentSettings: vi.fn(),
}));

const replace = vi.fn();
const settings = (hasLogo: boolean) => ({
  id: "establishment-id",
  name: "Barbearia",
  slug: "barbearia",
  time_zone: "America/Sao_Paulo",
  description: null,
  address: null,
  contact_phone: null,
  social_links: [],
  minimum_booking_notice_minutes: null,
  booking_horizon_days: null,
  customer_change_deadline_minutes: null,
  logo: hasLogo
    ? {
        id: "logo-id",
        content_type: "image/png" as const,
        byte_size: 1,
        width: 1,
        height: 1,
        url: "/api/v1/public/establishments/barbearia/logo",
      }
    : null,
  time_zone_editable: true,
  can_write: true,
  slug_conflict: false,
  is_complete: hasLogo,
  is_publishable: hasLogo,
});

beforeEach(() => {
  replace.mockReset();
  vi.mocked(useRouter).mockReturnValue({ replace } as unknown as ReturnType<
    typeof useRouter
  >);
  vi.mocked(getEstablishmentSettings).mockReset();
});

test.each([
  [false, "/painel/configuracoes"],
  [true, "/painel/profissionais"],
] as const)(
  "direciona o proprietário para %s quando o estabelecimento tem logo: %s",
  async (hasLogo, destination) => {
    vi.mocked(getEstablishmentSettings).mockResolvedValue({
      state: "success",
      status: 200,
      data: settings(hasLogo),
    });

    render(<OwnerPanelEntry />);

    await waitFor(() => expect(replace).toHaveBeenCalledWith(destination));
    expect(getEstablishmentSettings).toHaveBeenCalledOnce();
  },
);

test("mostra erro temporário e tenta consultar configurações novamente", async () => {
  vi.mocked(getEstablishmentSettings)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: settings(true),
    });

  render(<OwnerPanelEntry />);

  expect(
    await screen.findByText("Não foi possível consultar as configurações"),
  ).toBeInTheDocument();
  expect(screen.queryByText(/sessão/i)).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  await waitFor(() =>
    expect(replace).toHaveBeenCalledWith("/painel/profissionais"),
  );
  expect(getEstablishmentSettings).toHaveBeenCalledTimes(2);
});

test("oferece nova tentativa quando a API retorna uma falha temporária", async () => {
  vi.mocked(getEstablishmentSettings)
    .mockResolvedValueOnce({
      state: "error",
      status: 503,
      error: {
        code: "internal_error",
        message: "detalhe interno",
        request_id: "request-id",
      },
    })
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: settings(false),
    });

  render(<OwnerPanelEntry />);

  expect(
    await screen.findByText("Não foi possível consultar as configurações"),
  ).toBeInTheDocument();
  expect(screen.queryByText("detalhe interno")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  await waitFor(() =>
    expect(replace).toHaveBeenCalledWith("/painel/configuracoes"),
  );
});
