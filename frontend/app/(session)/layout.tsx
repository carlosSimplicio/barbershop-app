import type { ReactNode } from "react";

import { AuthProvider } from "@/components/session/auth-provider";

export default function SessionLayout({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}
