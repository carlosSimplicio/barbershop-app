import type { Metadata } from "next";

import { OwnerPanelEntry } from "@/components/session/owner-panel-entry/owner-panel-entry";
import { OwnerPanelPage } from "@/components/session/owner-panel-shell/owner-panel-page";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { SessionGuard } from "@/components/session/session-guard/session-guard";

export const metadata: Metadata = {
  title: "Painel | Barbershop",
  description: "Painel de administração do estabelecimento.",
};

export default function OwnerPanelEntryPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell activePage="none">
        <OwnerPanelPage
          description="Estamos preparando o painel do seu estabelecimento."
          title="Painel"
        >
          <OwnerPanelEntry />
        </OwnerPanelPage>
      </OwnerPanelShell>
    </SessionGuard>
  );
}
