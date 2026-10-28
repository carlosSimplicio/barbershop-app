import { render, screen } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { getPublicEstablishment } from "@/lib/establishment/public-establishment-server";
import EstablishmentPage, { generateMetadata } from "./page";

vi.mock("@/lib/establishment/public-establishment-server", () => ({
  getPublicEstablishment: vi.fn(),
  getServerConfig: () => ({ publicAppUrl: "https://example.test" }),
}));
vi.mock("next/navigation", () => ({
  notFound: () => {
    throw new Error("NOT_FOUND");
  },
}));
const profile = {
  name: "Estúdio Exemplo",
  slug: "estudio-exemplo",
  logo_url: "/api/v1/public/establishments/estudio-exemplo/logo",
};
const props = { params: Promise.resolve({ establishmentSlug: profile.slug }) };
beforeEach(() => {
  vi.mocked(getPublicEstablishment).mockReset().mockResolvedValue(profile);
});
test("renders identification and canonical metadata without empty sections or scheduling controls", async () => {
  render(await EstablishmentPage(props));
  expect(screen.getByRole("heading", { name: profile.name })).toBeVisible();
  expect(
    screen.getByRole("img", { name: `Logo de ${profile.name}` }),
  ).toHaveAttribute("src", profile.logo_url);
  expect(screen.queryByRole("region")).not.toBeInTheDocument();
  expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
  expect(await generateMetadata(props)).toMatchObject({
    title: profile.name,
    alternates: { canonical: "https://example.test/estudio-exemplo" },
  });
});
test("renders only filled optional data, preserving plain text and link order", async () => {
  vi.mocked(getPublicEstablishment).mockResolvedValue({
    ...profile,
    description: "<b>Sobre o estúdio</b>",
    address: "Rua Exemplo, 10",
    contact_phone: "+5511999999999",
    social_links: [
      { label: "Instagram", url: "https://instagram.com/example" },
      { label: "YouTube", url: "https://youtube.com/@example" },
    ],
  });
  render(await EstablishmentPage(props));
  expect(screen.getByText("<b>Sobre o estúdio</b>")).toBeVisible();
  expect(screen.getByText("Rua Exemplo, 10")).toBeVisible();
  expect(screen.getByRole("link", { name: "+5511999999999" })).toHaveAttribute(
    "href",
    "tel:+5511999999999",
  );
  expect(
    screen
      .getAllByRole("link")
      .slice(1)
      .map((link) => link.textContent),
  ).toEqual(["Instagram", "YouTube"]);
});
test("uses not found for an absent profile and propagates temporary failure", async () => {
  vi.mocked(getPublicEstablishment)
    .mockResolvedValueOnce(null)
    .mockRejectedValueOnce(new Error("Temporary failure"));
  await expect(EstablishmentPage(props)).rejects.toThrow("NOT_FOUND");
  await expect(EstablishmentPage(props)).rejects.toThrow("Temporary failure");
});
