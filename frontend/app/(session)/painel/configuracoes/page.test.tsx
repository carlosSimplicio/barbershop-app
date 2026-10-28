import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import EstablishmentSettingsPage from "./page";

const router = vi.hoisted(() => ({ replace: vi.fn() }));
const session = vi.hoisted(() => ({
  role: "owner" as "owner" | "professional",
}));
const getSettingsMock = vi.hoisted(() => vi.fn());

vi.mock("next/navigation", () => ({ useRouter: () => router }));
vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: () => ({
    state: {
      status: "authenticated",
      principal: {
        establishment_id: "establishment-id",
        email: "owner@example.test",
        password_change_required: false,
        role: session.role,
        user_id: "owner-id",
      },
    },
  }),
}));
vi.mock("@/lib/establishment/establishment-api", () => ({
  getEstablishmentSettings: getSettingsMock,
  patchEstablishmentSettings: vi.fn(),
}));

beforeEach(() => {
  session.role = "owner";
  router.replace.mockReset();
  getSettingsMock.mockReset().mockResolvedValue({
    data: {
      id: "establishment-id",
      name: "Estúdio Aurora",
      slug: "estudio-aurora",
      time_zone: "America/Sao_Paulo",
      description: null,
      address: null,
      contact_phone: null,
      social_links: [],
      minimum_booking_notice_minutes: null,
      booking_horizon_days: null,
      customer_change_deadline_minutes: null,
      logo: null,
      time_zone_editable: true,
      can_write: true,
      slug_conflict: false,
      is_complete: false,
      is_publishable: false,
    },
    state: "success",
    status: 200,
  });
});

test("exige sessão de proprietário para consultar configurações", async () => {
  render(<EstablishmentSettingsPage />);

  expect(
    await screen.findByRole("heading", { name: "Configurações" }),
  ).toBeVisible();
  expect(getSettingsMock).toHaveBeenCalledOnce();
  expect(router.replace).not.toHaveBeenCalled();
});

test("redireciona profissionais e não consulta settings", async () => {
  session.role = "professional";

  render(<EstablishmentSettingsPage />);

  await waitFor(() => expect(router.replace).toHaveBeenCalledWith("/testes"));
  expect(
    screen.queryByRole("heading", { name: "Configurações" }),
  ).not.toBeInTheDocument();
  expect(getSettingsMock).not.toHaveBeenCalled();
});
