export const ESTABLISHMENT_SOCIAL_NETWORKS = [
  {
    value: "instagram",
    label: "Instagram",
    urlPrefix: "https://instagram.com/",
    hostnames: ["instagram.com", "www.instagram.com"],
  },
  {
    value: "facebook",
    label: "Facebook",
    urlPrefix: "https://facebook.com/",
    hostnames: ["facebook.com", "www.facebook.com", "m.facebook.com"],
  },
  {
    value: "x",
    label: "X",
    urlPrefix: "https://x.com/",
    hostnames: ["x.com", "www.x.com", "twitter.com", "www.twitter.com"],
  },
  {
    value: "youtube",
    label: "YouTube",
    urlPrefix: "https://youtube.com/",
    hostnames: ["youtube.com", "www.youtube.com", "youtu.be"],
  },
  {
    value: "tiktok",
    label: "TikTok",
    urlPrefix: "https://tiktok.com/@",
    hostnames: ["tiktok.com", "www.tiktok.com"],
  },
  {
    value: "linkedin",
    label: "LinkedIn",
    urlPrefix: "https://linkedin.com/in/",
    hostnames: ["linkedin.com", "www.linkedin.com"],
  },
] as const;

export type EstablishmentSocialNetworkId =
  (typeof ESTABLISHMENT_SOCIAL_NETWORKS)[number]["value"];
