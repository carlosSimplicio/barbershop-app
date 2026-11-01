import type { Metadata } from "next";

import { SessionGuard } from "@/components/session/session-guard/session-guard";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { ProfessionalsWorkspace } from "@/components/professionals/professionals-workspace/professionals-workspace";

export const metadata: Metadata = {
  title: "Profissionais | Barbershop",
  description: "Área de profissionais do estabelecimento.",
};

export default function ProfessionalsPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell>
        <ProfessionalsWorkspace />
      </OwnerPanelShell>
    </SessionGuard>
  );
}
