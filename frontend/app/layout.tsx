import type { Metadata } from "next";
import { Montserrat } from "next/font/google";
import type { ReactNode } from "react";

import "./globals.css";

const montserrat = Montserrat({
  display: "swap",
  subsets: ["latin"],
  variable: "--font-montserrat",
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Barbershop",
  description: "Agendamento e gestão para o seu estabelecimento.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  return (
    <html className={montserrat.variable} lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
