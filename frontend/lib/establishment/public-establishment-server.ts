import "server-only";

import type { components } from "@/types/generated/api";

export type PublicEstablishment =
  components["schemas"]["PublicEstablishmentProfileResponse"];

export function getServerConfig() {
  return {
    apiInternalUrl: origin("API_INTERNAL_URL", ["http:", "https:"]),
    publicAppUrl: origin("PUBLIC_APP_URL", ["https:"]),
  };
}

function origin(name: string, protocols: string[]) {
  try {
    const url = new URL(process.env[name] ?? "");
    if (
      !protocols.includes(url.protocol) ||
      url.username ||
      url.password ||
      url.pathname !== "/" ||
      url.search ||
      url.hash
    )
      throw new Error();
    return url.origin;
  } catch {
    throw new Error(
      `${name} deve ser uma origem válida sem credenciais, caminho, query ou fragmento.`,
    );
  }
}

export async function getPublicEstablishment(
  slug: string,
): Promise<PublicEstablishment | null> {
  const { apiInternalUrl } = getServerConfig();
  if (!/^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$/.test(slug)) return null;
  try {
    const response = await fetch(
      `${apiInternalUrl}/api/v1/public/establishments/${encodeURIComponent(slug)}`,
      {
        cache: "no-store",
        credentials: "omit",
        redirect: "error",
        signal: AbortSignal.timeout(5000),
      },
    );
    if (response.status === 404) return null;
    if (response.status !== 200) throw new Error();
    return (await response.json()) as PublicEstablishment;
  } catch {
    throw new Error("Não foi possível consultar o estabelecimento.");
  }
}
