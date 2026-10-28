import type { components } from "@/types/generated/api";
import styles from "./public-establishment-profile.module.css";

type Profile = components["schemas"]["PublicEstablishmentProfileResponse"];

export function PublicEstablishmentProfile({ profile }: { profile: Profile }) {
  return (
    <main className={styles.page}>
      <article className={styles.profile}>
        <header className={styles.header}>
          {/* The stable public endpoint revalidates the current logo on every visit. */}
          <img
            className={styles.logo}
            src={profile.logo_url}
            alt={`Logo de ${profile.name}`}
            width={192}
            height={192}
          />
          <h1 className={styles.name}>{profile.name}</h1>
        </header>
        {profile.description ? (
          <section aria-label="Sobre">
            <p className={styles.text}>{profile.description}</p>
          </section>
        ) : null}
        {profile.address ? (
          <section aria-labelledby="address-title">
            <h2 id="address-title">Endereço</h2>
            <p className={styles.text}>{profile.address}</p>
          </section>
        ) : null}
        {profile.contact_phone ? (
          <section aria-labelledby="contact-title">
            <h2 id="contact-title">Contato</h2>
            <a className={styles.link} href={`tel:${profile.contact_phone}`}>
              {profile.contact_phone}
            </a>
          </section>
        ) : null}
        {profile.social_links?.length ? (
          <nav aria-label="Redes sociais">
            <ul className={styles.links}>
              {profile.social_links.map((link) => (
                <li key={link.url}>
                  <a className={styles.link} href={link.url} rel="noreferrer">
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        ) : null}
      </article>
    </main>
  );
}
