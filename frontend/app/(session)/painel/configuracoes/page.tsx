import type { Metadata } from "next";

import { EstablishmentSettingsWorkspace } from "@/components/establishment/establishment-settings-workspace/establishment-settings-workspace";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { OwnerPanelPage } from "@/components/session/owner-panel-shell/owner-panel-page";
import { SessionGuard } from "@/components/session/session-guard/session-guard";

export const metadata: Metadata = {
  title: "Configurações | Barbershop",
  description: "Configurações do estabelecimento e da página pública.",
};

export default function EstablishmentSettingsPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell activePage="settings">
        <OwnerPanelPage
          description="Atualize os dados públicos e os limites para agendamentos."
          title="Configurações"
        >
          <EstablishmentSettingsWorkspace />
        </OwnerPanelPage>
      </OwnerPanelShell>
    </SessionGuard>
  );
}
