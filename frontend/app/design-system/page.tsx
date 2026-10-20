import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { isDesignSystemAvailable } from "./availability";
import { DesignSystemShowcase } from "./showcase";

export const metadata: Metadata = {
  title: "Design system — Barbershop",
  description: "Referência visual dos componentes do frontend.",
  robots: { index: false, follow: false },
};

export default function DesignSystemPage() {
  if (!isDesignSystemAvailable(process.env.NODE_ENV)) {
    notFound();
  }

  return <DesignSystemShowcase />;
}
