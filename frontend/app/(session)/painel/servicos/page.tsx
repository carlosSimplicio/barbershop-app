import type { Metadata } from "next";
import { SessionGuard } from "@/components/session/session-guard/session-guard";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { CatalogWorkspace } from "@/components/catalog/catalog-workspace/catalog-workspace";

export const metadata: Metadata = {
  title: "Serviços | Barbershop",
  description: "Catálogo de serviços do estabelecimento.",
};
export default function ServicesPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell activePage="services">
        <CatalogWorkspace />
      </OwnerPanelShell>
    </SessionGuard>
  );
}
