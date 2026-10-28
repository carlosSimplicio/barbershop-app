import type { Metadata } from "next";

import { SessionGuard } from "@/components/session/session-guard/session-guard";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { OwnerPanelPage } from "@/components/session/owner-panel-shell/owner-panel-page";
import { ProfessionalsWorkspace } from "@/components/professionals/professionals-workspace/professionals-workspace";

export const metadata: Metadata = {
  title: "Profissionais | Barbershop",
  description: "Área de profissionais do estabelecimento.",
};

export default function ProfessionalsPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell>
        <OwnerPanelPage
          description="Cadastre e acompanhe os profissionais que atendem no estabelecimento."
          title="Profissionais"
        >
          <ProfessionalsWorkspace />
        </OwnerPanelPage>
      </OwnerPanelShell>
    </SessionGuard>
  );
}
