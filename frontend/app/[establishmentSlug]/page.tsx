import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { cache } from "react";

import { PublicEstablishmentProfile } from "@/components/establishment/public-establishment-profile/public-establishment-profile";
import {
  getPublicEstablishment,
  getServerConfig,
} from "@/lib/establishment/public-establishment-server";

const loadProfile = cache(getPublicEstablishment);
type Props = { params: Promise<{ establishmentSlug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { establishmentSlug } = await params;
  const profile = await loadProfile(establishmentSlug);
  if (!profile) notFound();
  const { publicAppUrl } = getServerConfig();
  return {
    title: profile.name,
    description: profile.description || `Conheça ${profile.name}.`,
    alternates: {
      canonical: new URL(`/${encodeURIComponent(profile.slug)}`, publicAppUrl)
        .href,
    },
  };
}

export default async function EstablishmentPage({ params }: Props) {
  const { establishmentSlug } = await params;
  const profile = await loadProfile(establishmentSlug);
  if (!profile) notFound();
  return <PublicEstablishmentProfile profile={profile} />;
}
