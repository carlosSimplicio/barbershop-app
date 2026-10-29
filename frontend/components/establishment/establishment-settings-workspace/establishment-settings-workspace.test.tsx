import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import type { EstablishmentSettings } from "@/lib/establishment/establishment-api";
import { BRAZILIAN_TIME_ZONES } from "@/lib/establishment/brazilian-time-zones";

import { EstablishmentSettingsWorkspace } from "./establishment-settings-workspace";

const session = vi.hoisted(() => ({
  establishmentId: "establishment-id",
  userId: "owner-id",
}));
const getSettingsMock = vi.hoisted(() => vi.fn());
const patchSettingsMock = vi.hoisted(() => vi.fn());
const uploadLogoMock = vi.hoisted(() => vi.fn());

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: () => ({
    state: {
      status: "authenticated",
      principal: {
        establishment_id: session.establishmentId,
        email: "owner@example.test",
        password_change_required: false,
        role: "owner",
        user_id: session.userId,
      },
    },
  }),
}));

vi.mock("@/lib/establishment/establishment-api", () => ({
  getEstablishmentSettings: getSettingsMock,
  patchEstablishmentSettings: patchSettingsMock,
  uploadEstablishmentLogo: uploadLogoMock,
}));

beforeEach(() => {
  session.establishmentId = "establishment-id";
  session.userId = "owner-id";
  getSettingsMock.mockReset();
  patchSettingsMock.mockReset();
  uploadLogoMock.mockReset();
});

afterEach(() => vi.restoreAllMocks());

test("carrega dados do perfil e apresenta o slug somente para leitura", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Nome do estabelecimento")).toBeVisible();
  expect(screen.getByLabelText("Nome do estabelecimento")).toHaveValue(
    "Estúdio Aurora",
  );
  expect(screen.getByLabelText("Slug")).toHaveValue("estudio-aurora");
  expect(screen.getByLabelText("Slug")).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "Rede social 1" })).toHaveValue(
    "instagram",
  );
  expect(
    Array.from(
      screen
        .getByRole("combobox", { name: "Rede social 1" })
        .querySelectorAll("option"),
    ).map((option) => option.textContent?.trim()),
  ).toEqual([
    "Selecione uma rede",
    "Instagram",
    "Facebook",
    "X",
    "YouTube",
    "TikTok",
    "LinkedIn",
  ]);
  expect(screen.getByLabelText("URL do Instagram")).toHaveValue("aurora");
  expect(screen.getByText("https://instagram.com/")).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Adicionar rede social" }),
  ).toHaveTextContent("+");
  expect(
    screen.getByRole("button", { name: "Remover rede social Instagram" }),
  ).toHaveTextContent("×");
  expect(
    Array.from(
      document.querySelectorAll(
        "#name, #slug, #time_zone, #contact_phone, #description, #address",
      ),
    ).map((field) => field.id),
  ).toEqual([
    "name",
    "slug",
    "time_zone",
    "contact_phone",
    "description",
    "address",
  ]);
  const timeZoneSelect = screen.getByRole("combobox", { name: "Fuso horário" });
  expect(timeZoneSelect).toHaveValue("America/Sao_Paulo");
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option")).map(
      (option) => (option as HTMLOptionElement).value,
    ),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ value }) => value));
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option")).map((option) =>
      option.textContent?.trim(),
    ),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ label }) => label));
  expect(screen.queryByLabelText("Identificador IANA")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Descrição")).toHaveValue("Cortes e barba");
  expect(screen.getByLabelText("Prazo de antecedência mínima")).toHaveValue(30);
});

test("preserva como legado um link cujo rótulo não corresponde ao domínio", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({
        social_links: [
          { label: "Instagram", url: "https://example.test/profile" },
        ],
      }),
    ),
  );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Descrição")).toBeVisible();
  expect(screen.getByRole("combobox", { name: "Rede social 1" })).toHaveValue(
    "",
  );
  expect(screen.getByText(/Este link existente não corresponde/)).toBeVisible();

  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { name: "Novo nome" },
    expect.any(AbortSignal),
  );
});

test.each([
  ["LinkedIn", "https://linkedin.com/company/aurora"],
  ["YouTube", "https://youtu.be/aurora"],
  ["TikTok", "https://tiktok.com/business/aurora"],
])(
  "preserva o link de %s incompatível com o prefixo do editor",
  async (label, url) => {
    getSettingsMock.mockResolvedValue(
      successfulResult(
        successfulSettings({
          social_links: [
            { label, url },
            { label: "Instagram", url: "https://instagram.com/aurora" },
          ],
        }),
      ),
    );
    patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

    render(<EstablishmentSettingsWorkspace />);

    expect(
      await screen.findByRole("combobox", { name: "Rede social 1" }),
    ).toHaveValue("");
    expect(screen.queryByLabelText(`URL do ${label}`)).not.toBeInTheDocument();
    expect(
      screen.getByText(/Este link existente não corresponde/),
    ).toBeVisible();

    fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
      target: { value: "Novo nome" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Remover rede social Instagram" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Salvar configurações" }),
    );

    await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
    expect(patchSettingsMock).toHaveBeenCalledWith(
      { name: "Novo nome", social_links: [{ label, url }] },
      expect.any(AbortSignal),
    );
  },
);

test.each([
  [
    "LinkedIn",
    "https://www.linkedin.com/in/aurora",
    "https://linkedin.com/in/aurora-nova",
  ],
  ["X", "https://twitter.com/aurora", "https://x.com/aurora-nova"],
])(
  "mantém o caminho de perfil ao editar um domínio alternativo de %s",
  async (label, url, updatedURL) => {
    getSettingsMock.mockResolvedValue(
      successfulResult(
        successfulSettings({
          social_links: [{ label, url }],
        }),
      ),
    );
    patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

    render(<EstablishmentSettingsWorkspace />);

    fireEvent.change(await screen.findByLabelText(`URL do ${label}`), {
      target: { value: "aurora-nova" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Salvar configurações" }),
    );

    await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
    expect(patchSettingsMock).toHaveBeenCalledWith(
      { social_links: [{ label, url: updatedURL }] },
      expect.any(AbortSignal),
    );
  },
);

test("reconcilia o salvamento confirmado antes de permitir novas alterações", async () => {
  const savedSettings = successfulSettings({ name: "Novo nome" });
  getSettingsMock
    .mockResolvedValueOnce(successfulResult())
    .mockRejectedValueOnce(new Error("temporary refresh failure"))
    .mockRejectedValueOnce(new Error("temporary retry failure"))
    .mockResolvedValueOnce(successfulResult(savedSettings))
    .mockResolvedValueOnce(successfulResult());
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  fireEvent.change(await screen.findByLabelText("Nome do estabelecimento"), {
    target: { value: savedSettings.name },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  const refresh = await screen.findByRole("button", {
    name: "Atualizar configurações",
  });
  expect(screen.getByLabelText("Nome do estabelecimento")).toBeDisabled();
  expect(screen.getByLabelText("Selecionar logo")).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Salvar configurações" }),
  ).toBeDisabled();

  fireEvent.click(refresh);
  await screen.findByText("Não foi possível consultar as configurações");
  expect(
    screen.queryByLabelText("Nome do estabelecimento"),
  ).not.toBeInTheDocument();
  expect(patchSettingsMock).toHaveBeenCalledOnce();

  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  const nameField = await screen.findByLabelText("Nome do estabelecimento");
  expect(nameField).toBeEnabled();
  expect(nameField).toHaveValue(savedSettings.name);
  fireEvent.change(nameField, { target: { value: "Estúdio Aurora" } });
  const save = screen.getByRole("button", { name: "Salvar configurações" });
  expect(save).toBeEnabled();
  fireEvent.click(save);

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledTimes(2));
  expect(patchSettingsMock).toHaveBeenLastCalledWith(
    { name: "Estúdio Aurora" },
    expect.any(AbortSignal),
  );
});

test("permite limpar dados opcionais, links e limites", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Descrição"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Endereço"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Telefone de contato"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Prazo de antecedência mínima"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Horizonte máximo de agendamento"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Prazo para alterar agendamento"), {
    target: { value: "" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Remover rede social Instagram" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    {
      description: null,
      address: null,
      contact_phone: null,
      social_links: [],
      minimum_booking_notice_minutes: null,
      booking_horizon_days: null,
      customer_change_deadline_minutes: null,
    },
    expect.any(AbortSignal),
  );
});

test("valida limite inteiro e URL HTTPS antes de enviar", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Horizonte máximo de agendamento"), {
    target: { value: "0" },
  });
  fireEvent.change(screen.getByLabelText("URL do Instagram"), {
    target: { value: "" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Informe um número inteiro maior que zero."),
  ).toBeVisible();
  expect(screen.getByText("Informe o endereço do perfil.")).toBeVisible();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("seleciona uma rede social e mantém o prefixo da URL fora do campo editável", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(successfulSettings({ social_links: [] })),
  );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);
  await screen.findByLabelText("Descrição");

  fireEvent.click(
    screen.getByRole("button", { name: "Adicionar rede social" }),
  );
  fireEvent.change(screen.getByRole("combobox", { name: "Rede social 1" }), {
    target: { value: "linkedin" },
  });

  expect(screen.getByText("https://linkedin.com/in/")).toBeVisible();
  expect(screen.getByLabelText("URL do LinkedIn")).toHaveValue("");
  expect(screen.getByLabelText("URL do LinkedIn")).toHaveFocus();
  fireEvent.change(screen.getByLabelText("URL do LinkedIn"), {
    target: { value: "estudio-aurora" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    {
      social_links: [
        {
          label: "LinkedIn",
          url: "https://linkedin.com/in/estudio-aurora",
        },
      ],
    },
    expect.any(AbortSignal),
  );
});

test("permite consultar novamente após uma falha temporária sem mostrar erro bruto", async () => {
  getSettingsMock
    .mockRejectedValueOnce(new Error("internal database details"))
    .mockResolvedValueOnce(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByText("Tente novamente mais tarde.")).toBeVisible();
  expect(
    screen.queryByText("internal database details"),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  expect(await screen.findByLabelText("Descrição")).toHaveValue(
    "Cortes e barba",
  );
});

test("mapeia erro do telefone sem renderizar mensagem bruta do backend", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({
    error: {
      code: "validation_failed",
      fields: [
        {
          code: "invalid_phone",
          field: "contact_phone",
          message: "backend internal message",
        },
      ],
      message: "backend internal message",
      request_id: "request-id",
    },
    state: "error",
    status: 400,
  });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Telefone de contato"), {
    target: { value: "+5511988887777" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Informe um telefone brasileiro com DDD."),
  ).toBeVisible();
  expect(
    screen.queryByText("backend internal message"),
  ).not.toBeInTheDocument();
});

test("bloqueia escrita e alteração de fuso conforme as permissões do backend", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({ can_write: false, time_zone_editable: false }),
    ),
  );

  render(<EstablishmentSettingsWorkspace />);

  expect(
    await screen.findByText(
      "O trial expirou. Você pode consultar as configurações, mas não salvá-las.",
    ),
  ).toBeVisible();
  expect(screen.getByLabelText("Fuso horário")).toBeDisabled();
  expect(screen.getByLabelText("Descrição")).toBeDisabled();
  expect(screen.getByLabelText("Selecionar logo")).toBeDisabled();
  expect(screen.getByRole("button", { name: "Enviar logo" })).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Salvar configurações" }),
  ).toBeDisabled();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("pré-visualiza e envia o logo, atualiza a publicação e limpa a seleção", async () => {
  const createObjectURL = vi
    .spyOn(URL, "createObjectURL")
    .mockReturnValue("blob:logo-preview");
  const revokeObjectURL = vi
    .spyOn(URL, "revokeObjectURL")
    .mockImplementation(() => {});
  getSettingsMock
    .mockResolvedValueOnce(successfulResult())
    .mockResolvedValueOnce(
      successfulResult(
        successfulSettings({
          logo: {
            url: "/api/v1/public/establishments/estudio-aurora/logo",
            content_type: "image/png",
            byte_size: 4,
            width: 1,
            height: 1,
          },
          is_complete: true,
          is_publishable: true,
        }),
      ),
    );
  uploadLogoMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);
  const file = new File(["png!"], "logo.png", { type: "image/png" });
  const input = await screen.findByLabelText("Selecionar logo");
  fireEvent.change(input, { target: { files: [file] } });

  expect(screen.getByText("logo.png")).toBeVisible();
  expect(createObjectURL).toHaveBeenCalledWith(file);
  expect(
    screen.getByRole("img", { name: "Pré-visualização do logo" }),
  ).toHaveAttribute("src", "blob:logo-preview");
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));

  expect(await screen.findByText("Logo atualizado com sucesso.")).toBeVisible();
  expect(uploadLogoMock).toHaveBeenCalledWith(file, expect.any(AbortSignal));
  expect(
    screen.getByRole("link", { name: "Abrir página pública" }),
  ).toHaveAttribute("href", "/estudio-aurora");
  expect(
    screen.getByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).toHaveAttribute(
    "src",
    "/api/v1/public/establishments/estudio-aurora/logo?version=1",
  );
  expect(input).toHaveValue("");
  expect(revokeObjectURL).toHaveBeenCalledWith("blob:logo-preview");
});

test("limpa o arquivo selecionado sem remover o logo publicado", async () => {
  vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:temporary-logo");
  const revokeObjectURL = vi
    .spyOn(URL, "revokeObjectURL")
    .mockImplementation(() => {});
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({
        logo: {
          url: "/api/v1/public/establishments/estudio-aurora/logo",
          content_type: "image/png",
          byte_size: 4,
          width: 1,
          height: 1,
        },
        is_complete: true,
        is_publishable: true,
      }),
    ),
  );

  render(<EstablishmentSettingsWorkspace />);
  const input = await screen.findByLabelText("Escolher novo logo");
  fireEvent.change(input, {
    target: {
      files: [new File(["new!"], "replacement.png", { type: "image/png" })],
    },
  });

  expect(
    screen.getByRole("img", { name: "Pré-visualização do logo" }),
  ).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Limpar arquivo selecionado" })
      .parentElement,
  ).toContainElement(
    screen.getByRole("img", { name: "Pré-visualização do logo" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Limpar arquivo selecionado" }),
  );

  expect(screen.getByText("Nenhum arquivo selecionado")).toBeVisible();
  expect(
    screen.queryByRole("img", { name: "Pré-visualização do logo" }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).toHaveAttribute("src", "/api/v1/public/establishments/estudio-aurora/logo");
  expect(screen.getByRole("button", { name: "Enviar logo" })).toBeDisabled();
  expect(uploadLogoMock).not.toHaveBeenCalled();
  expect(revokeObjectURL).toHaveBeenCalledWith("blob:temporary-logo");
});

test("mantém o logo publicado após falha e permite tentar novamente", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({
        logo: {
          url: "/api/v1/public/establishments/estudio-aurora/logo",
          content_type: "image/png",
          byte_size: 4,
          width: 1,
          height: 1,
        },
        is_complete: true,
        is_publishable: true,
      }),
    ),
  );
  uploadLogoMock
    .mockResolvedValueOnce({
      error: {
        code: "validation_failed",
        fields: [
          {
            code: "invalid_logo",
            field: "file",
            message: "backend internal message",
          },
        ],
        message: "backend internal message",
        request_id: "request-id",
      },
      state: "error",
      status: 400,
    })
    .mockResolvedValueOnce({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);
  const input = await screen.findByLabelText("Escolher novo logo");
  const file = new File(["png!"], "replacement.png", { type: "image/png" });
  fireEvent.change(input, { target: { files: [file] } });
  expect(
    screen.getByRole("img", { name: "Pré-visualização do logo" }),
  ).toBeVisible();
  expect(
    screen.queryByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).not.toBeInTheDocument();
  expect(screen.getAllByRole("img")).toHaveLength(1);
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));

  expect(
    await screen.findByText("Escolha uma imagem PNG, JPEG ou WebP válida."),
  ).toBeVisible();
  expect(
    screen.getByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).toHaveAttribute("src", "/api/v1/public/establishments/estudio-aurora/logo");
  expect(
    screen.queryByRole("img", { name: "Pré-visualização do logo" }),
  ).not.toBeInTheDocument();
  expect(screen.getByText("Nenhum arquivo selecionado")).toBeVisible();
  expect(screen.getByRole("button", { name: "Enviar logo" })).toBeDisabled();
  expect(
    screen.queryByText("backend internal message"),
  ).not.toBeInTheDocument();

  fireEvent.change(screen.getByLabelText("Escolher novo logo"), {
    target: { files: [file] },
  });
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));
  expect(await screen.findByText("Logo atualizado com sucesso.")).toBeVisible();
  expect(uploadLogoMock).toHaveBeenCalledTimes(2);
});

test.each([
  [400, "validation_failed"],
  [413, "request_too_large"],
  [403, "csrf_failed"],
  [403, "read_only"],
])(
  "limpa o primeiro logo recusado com status %i e código %s",
  async (status, code) => {
    getSettingsMock.mockResolvedValue(successfulResult());
    const revoke = vi.spyOn(URL, "revokeObjectURL");
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:first-logo");
    uploadLogoMock.mockResolvedValue({
      state: "error",
      status,
      error: {
        code,
        message: "backend internal message",
        request_id: "request-id",
      },
    });
    render(<EstablishmentSettingsWorkspace />);
    fireEvent.change(await screen.findByLabelText("Selecionar logo"), {
      target: {
        files: [new File(["invalid"], "rejected.png", { type: "image/png" })],
      },
    });
    fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));
    expect(await screen.findByText("Nenhum arquivo selecionado")).toBeVisible();
    expect(
      screen.queryByRole("img", { name: "Pré-visualização do logo" }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Enviar logo" })).toBeDisabled();
    expect(revoke).toHaveBeenCalledWith("blob:first-logo");
    expect(
      screen.queryByText("backend internal message"),
    ).not.toBeInTheDocument();
  },
);

test("mostra o endereço quando o upload confirma mas a atualização falha", async () => {
  getSettingsMock
    .mockResolvedValueOnce(successfulResult())
    .mockRejectedValueOnce(new Error("temporary refresh failure"));
  uploadLogoMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);
  const input = await screen.findByLabelText("Selecionar logo");
  const file = new File(["png!"], "logo.png", { type: "image/png" });
  fireEvent.change(input, { target: { files: [file] } });
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));

  expect(
    await screen.findByText(
      "Logo enviado. Não foi possível atualizar as configurações agora.",
    ),
  ).toBeVisible();
  expect(
    screen.getByRole("link", { name: "Abrir página pública" }),
  ).toHaveAttribute("href", "/estudio-aurora");
  expect(
    screen.getByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).toHaveAttribute(
    "src",
    "/api/v1/public/establishments/estudio-aurora/logo?version=1",
  );
});

test("reconcilia um erro interno do upload antes de limpar a seleção", async () => {
  const refreshedSettings = successfulSettings({
    logo: {
      url: "/api/v1/public/establishments/estudio-aurora/logo",
      content_type: "image/webp",
      byte_size: 8,
      width: 2,
      height: 2,
    },
    is_complete: true,
    is_publishable: true,
  });
  getSettingsMock
    .mockResolvedValueOnce(
      successfulResult(
        successfulSettings({
          logo: {
            url: "/api/v1/public/establishments/estudio-aurora/logo",
            content_type: "image/png",
            byte_size: 4,
            width: 1,
            height: 1,
          },
          is_complete: true,
          is_publishable: true,
        }),
      ),
    )
    .mockResolvedValueOnce(successfulResult(refreshedSettings));
  uploadLogoMock.mockResolvedValue({
    error: {
      code: "internal_error",
      message: "Não foi possível enviar o logo.",
      request_id: "request-id",
    },
    state: "error",
    status: 500,
  });

  render(<EstablishmentSettingsWorkspace />);
  const input = await screen.findByLabelText("Escolher novo logo");
  fireEvent.change(input, {
    target: {
      files: [
        new File(["new logo"], "replacement.webp", { type: "image/webp" }),
      ],
    },
  });
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));

  expect(
    await screen.findByText(
      "Confira o logo exibido antes de iniciar outro envio.",
    ),
  ).toBeVisible();
  expect(
    screen.getByRole("img", { name: "Logo de Estúdio Aurora" }),
  ).toHaveAttribute(
    "src",
    "/api/v1/public/establishments/estudio-aurora/logo?version=1",
  );
  expect(
    screen.queryByRole("img", { name: "Pré-visualização do logo" }),
  ).not.toBeInTheDocument();
  expect(screen.getByText("Nenhum arquivo selecionado")).toBeVisible();
});

test("mantém a seleção incerta e oferece reconciliação quando a consulta também falha", async () => {
  const currentSettings = successfulSettings({
    logo: {
      url: "/api/v1/public/establishments/estudio-aurora/logo",
      content_type: "image/png",
      byte_size: 4,
      width: 1,
      height: 1,
    },
    is_complete: true,
    is_publishable: true,
  });
  getSettingsMock
    .mockResolvedValueOnce(successfulResult(currentSettings))
    .mockRejectedValueOnce(new Error("settings temporarily unavailable"))
    .mockResolvedValueOnce(successfulResult(currentSettings));
  uploadLogoMock.mockResolvedValue({
    error: {
      code: "internal_error",
      message: "Não foi possível enviar o logo.",
      request_id: "request-id",
    },
    state: "error",
    status: 500,
  });

  render(<EstablishmentSettingsWorkspace />);
  fireEvent.change(await screen.findByLabelText("Escolher novo logo"), {
    target: {
      files: [
        new File(["new logo"], "replacement.webp", { type: "image/webp" }),
      ],
    },
  });
  fireEvent.click(screen.getByRole("button", { name: "Enviar logo" }));

  expect(
    await screen.findByText(
      "A consulta das configurações também falhou. Atualize o estado do logo antes de enviar novamente.",
    ),
  ).toBeVisible();
  expect(
    screen.getByRole("img", { name: "Pré-visualização do logo" }),
  ).toBeVisible();
  expect(screen.getByText("replacement.webp")).toBeVisible();
  expect(screen.getByRole("button", { name: "Enviar logo" })).toBeDisabled();

  fireEvent.click(
    screen.getByRole("button", { name: "Atualizar estado do logo" }),
  );
  expect(
    await screen.findByText(
      "Confira o logo exibido antes de iniciar outro envio.",
    ),
  ).toBeVisible();
  expect(screen.getByText("Nenhum arquivo selecionado")).toBeVisible();
  expect(
    screen.queryByRole("img", { name: "Pré-visualização do logo" }),
  ).not.toBeInTheDocument();
});

test("mantém o fuso bloqueado sem impedir a edição dos demais dados", async () => {
  const currentSettings = successfulSettings({ time_zone_editable: false });
  getSettingsMock
    .mockResolvedValueOnce(successfulResult(currentSettings))
    .mockResolvedValueOnce(
      successfulResult({ ...currentSettings, description: "Nova descrição" }),
    );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  expect(screen.getByLabelText("Fuso horário")).toBeDisabled();
  fireEvent.change(screen.getByLabelText("Descrição"), {
    target: { value: "Nova descrição" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { description: "Nova descrição" },
    expect.any(AbortSignal),
  );
});

test("permite escolher qualquer fuso listado e salva a opção selecionada", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  const timeZoneSelect = await screen.findByRole("combobox", {
    name: "Fuso horário",
  });
  fireEvent.change(timeZoneSelect, {
    target: { value: "America/Manaus" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { time_zone: "America/Manaus" },
    expect.any(AbortSignal),
  );
});

test("não representa fuso fora da lista como se fosse uma opção brasileira", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(successfulSettings({ time_zone: "America/New_York" })),
  );

  render(<EstablishmentSettingsWorkspace />);

  const timeZoneSelect = await screen.findByRole("combobox", {
    name: "Fuso horário",
  });
  expect(timeZoneSelect).toHaveValue("");
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option"))
      .map((option) => (option as HTMLOptionElement).value)
      .filter(Boolean),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ value }) => value));
  expect(
    screen.getByRole("option", { name: "Selecione um fuso horário" }),
  ).toBeDisabled();

  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText(
      "Escolha um dos quatro fusos horários brasileiros.",
    ),
  ).toBeVisible();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("não exibe detalhes brutos quando o salvamento falha temporariamente", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockRejectedValue(new Error("database connection details"));

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Verifique sua conexão e tente novamente."),
  ).toBeVisible();
  expect(
    screen.queryByText("database connection details"),
  ).not.toBeInTheDocument();
});

test("ignora a resposta de um tenant anterior após troca de sessão", async () => {
  let resolveOldRequest:
    ((value: ReturnType<typeof successfulResult>) => void) | undefined;
  getSettingsMock.mockImplementation((signal: AbortSignal) => {
    if (session.establishmentId === "old-establishment") {
      return new Promise((resolve) => {
        resolveOldRequest = resolve;
        signal.addEventListener("abort", () =>
          resolve(successfulResult(successfulSettings({ name: "Antigo" }))),
        );
      });
    }
    return Promise.resolve(
      successfulResult(successfulSettings({ name: "Atual" })),
    );
  });
  session.establishmentId = "old-establishment";

  const { rerender } = render(<EstablishmentSettingsWorkspace />);
  session.establishmentId = "new-establishment";
  session.userId = "new-owner";
  rerender(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Nome do estabelecimento")).toHaveValue(
    "Atual",
  );
  resolveOldRequest?.(successfulResult(successfulSettings({ name: "Antigo" })));
  expect(screen.getByLabelText("Nome do estabelecimento")).toHaveValue("Atual");
});

test("impede submissões simultâneas", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  let resolvePatch:
    ((value: { state: "empty"; status: number }) => void) | undefined;
  patchSettingsMock.mockImplementation(
    () =>
      new Promise((resolve) => {
        resolvePatch = resolve;
      }),
  );

  render(<EstablishmentSettingsWorkspace />);
  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  const form = screen
    .getByRole("button", { name: "Salvar configurações" })
    .closest("form");
  if (!form) throw new Error("Settings form was not rendered");
  fireEvent.submit(form);
  fireEvent.submit(form);

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  resolvePatch?.({ state: "empty", status: 204 });
});

function successfulSettings(
  overrides: Partial<EstablishmentSettings> = {},
): EstablishmentSettings {
  return {
    id: "establishment-id",
    name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    description: "Cortes e barba",
    address: "Rua Central, 123",
    contact_phone: "+5511999990000",
    social_links: [{ label: "Instagram", url: "https://instagram.com/aurora" }],
    minimum_booking_notice_minutes: 30,
    booking_horizon_days: 60,
    customer_change_deadline_minutes: 120,
    logo: null,
    time_zone_editable: true,
    can_write: true,
    slug_conflict: false,
    is_complete: false,
    is_publishable: false,
    ...overrides,
  };
}

function successfulResult(settings = successfulSettings()) {
  return { data: settings, state: "success", status: 200 } as const;
}
