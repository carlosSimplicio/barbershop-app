"use client";

import { useState } from "react";

import {
  AffixedInput,
  Button,
  Field,
  Feedback,
  IconButton,
  PasswordField,
  SelectField,
  TextField,
} from "@/components/ui";

import styles from "./showcase.module.css";

type Theme = "light" | "dark";

const swatches = [
  ["Azul principal", "--color-primary", "primary"],
  ["Azul suave", "--color-accent", "accent"],
  ["Texto", "--color-ink", "ink"],
  ["Superfície", "--color-surface", "surface"],
  ["Contorno", "--color-line", "line"],
  ["Erro", "--color-danger", "danger"],
] as const;

export function DesignSystemShowcase() {
  const [theme, setTheme] = useState<Theme>("light");
  const [actionMessage, setActionMessage] = useState(
    "Use os controles para validar as interações.",
  );
  const [iconPressed, setIconPressed] = useState(false);

  return (
    <main className={styles.page} data-theme={theme}>
      <header className={styles.header}>
        <div>
          <span className={styles.eyebrow}>Identidade visual</span>
          <h1>Clareza para cuidar.</h1>
          <p>
            Componentes reais, configuráveis e compartilhados por todas as
            páginas do produto.
          </p>
        </div>

        <fieldset className={styles.themeSwitcher}>
          <legend>Tema da referência</legend>
          {(["light", "dark"] as const).map((option) => (
            <button
              aria-pressed={theme === option}
              key={option}
              onClick={() => setTheme(option)}
              type="button"
            >
              <span aria-hidden="true">{option === "light" ? "☀" : "◐"}</span>
              {option === "light" ? "Claro" : "Escuro"}
            </button>
          ))}
        </fieldset>
      </header>

      <nav aria-label="Seções da referência" className={styles.navigation}>
        <a href="#foundations">Fundamentos</a>
        <a href="#actions">Ações</a>
        <a href="#fields">Campos</a>
        <a href="#states">Estados</a>
        <a href="#accessibility">Acessibilidade</a>
      </nav>

      <section className={styles.section} id="foundations">
        <SectionHeading
          description="Tokens semânticos permitem evoluir a aparência sem alterar as páginas."
          number="01"
          title="Fundamentos"
        />
        <Showcase title="Paleta essencial">
          <div className={styles.swatchGrid}>
            {swatches.map(([label, token, color]) => (
              <div className={styles.swatch} key={token}>
                <span className={styles[color]} />
                <strong>{label}</strong>
                <code>{token}</code>
              </div>
            ))}
          </div>
        </Showcase>
        <div className={styles.foundationGrid}>
          <Showcase title="Tipografia">
            <p className={styles.displaySample}>Organize o seu tempo.</p>
            <p className={styles.bodySample}>
              Montserrat mantém a leitura simples em títulos, textos e rótulos.
            </p>
          </Showcase>
          <Showcase title="Raios e espaço">
            <div className={styles.radiusGrid}>
              <span data-radius="sm">12px</span>
              <span data-radius="md">16px</span>
              <span data-radius="lg">32px</span>
            </div>
          </Showcase>
        </div>
      </section>

      <section className={styles.section} id="actions">
        <SectionHeading
          description="Labels, ícones e callbacks pertencem à página consumidora."
          number="02"
          title="Controles de ação"
        />
        <Showcase title="Botões">
          <div className={styles.actionGrid}>
            <Button
              onClick={() => setActionMessage("Ação primária executada.")}
            >
              Criar meu espaço <ChevronIcon />
            </Button>
            <Button disabled>Indisponível</Button>
            <IconButton
              aria-label={
                iconPressed ? "Desmarcar favorito" : "Marcar favorito"
              }
              onClick={() => {
                const nextPressed = !iconPressed;
                setIconPressed(nextPressed);
                setActionMessage(
                  nextPressed ? "Ícone pressionado." : "Ícone desmarcado.",
                );
              }}
              pressed={iconPressed}
            >
              <StarIcon />
            </IconButton>
            <IconButton aria-label="Ação indisponível" disabled>
              <StarIcon />
            </IconButton>
          </div>
          <p aria-live="polite" className={styles.status} role="status">
            {actionMessage}
          </p>
          <p className={styles.instruction}>
            Use ponteiro e Tab para conferir hover, foco, clique e estado ativo.
          </p>
        </Showcase>
      </section>

      <section className={styles.section} id="fields">
        <SectionHeading
          description="Primitivas flexíveis e wrappers prontos compartilham os mesmos estilos."
          number="03"
          title="Campos de formulário"
        />
        <Showcase title="Tipos utilizados">
          <div className={styles.fieldGrid}>
            <TextField
              hint="Para nomes e informações curtas."
              label="Texto"
              placeholder="Ex.: Estúdio Aurora"
            />
            <TextField
              defaultValue="voce@exemplo.com"
              hint="Use teclado e autocomplete adequados."
              label="E-mail"
              type="email"
            />
            <Field
              hint="O prefixo contextualiza sem fazer parte do valor."
              hintId="showcase-slug-hint"
              htmlFor="showcase-slug"
              label="Campo com prefixo"
            >
              <AffixedInput
                affix="https://"
                aria-describedby="showcase-slug-hint"
                defaultValue="estudio-aurora"
                id="showcase-slug"
              />
            </Field>
            <SelectField
              defaultValue="America/Sao_Paulo"
              hint="Mantenha uma opção inicial válida."
              label="Seleção"
            >
              <option value="America/Sao_Paulo">São Paulo (GMT−3)</option>
              <option value="America/Manaus">Manaus (GMT−4)</option>
            </SelectField>
            <PasswordField
              defaultValue="senhasegura"
              hint="Use de 8 a 32 caracteres, sem espaços."
              label="Senha"
            />
          </div>
        </Showcase>
      </section>

      <section className={styles.section} id="states">
        <SectionHeading
          description="O espaço de erro permanece reservado para evitar saltos no layout."
          number="04"
          title="Estados e validação"
        />
        <Showcase title="Estados observáveis">
          <div className={styles.fieldGrid}>
            <TextField hint="Ajuda opcional e objetiva." label="Padrão" />
            <TextField
              defaultValue="email-invalido"
              error="Digite um e-mail válido."
              hint="A instrução continua disponível."
              label="Erro"
              type="email"
            />
            <TextField
              defaultValue="Não editável"
              disabled
              hint="Explique no contexto por que está indisponível."
              label="Desabilitado"
            />
            <SelectField disabled label="Seleção desabilitada">
              <option>Não editável</option>
            </SelectField>
          </div>
        </Showcase>
        <Showcase title="Feedback assíncrono">
          <div className={styles.feedbackGrid}>
            <Feedback title="Cadastro recebido" variant="success">
              Enviamos as instruções de confirmação para o seu e-mail.
            </Feedback>
            <Feedback title="Atenção necessária" variant="warning">
              Sua sessão está perto de expirar.
            </Feedback>
            <Feedback
              action={<Button>Tentar novamente</Button>}
              title="Não foi possível concluir"
              variant="error"
            >
              Confira sua conexão e repita a solicitação.
            </Feedback>
          </div>
        </Showcase>
      </section>

      <section className={styles.section} id="accessibility">
        <SectionHeading
          description="Regras mínimas incorporadas aos componentes compartilhados."
          number="05"
          title="Acessibilidade"
        />
        <div className={styles.ruleGrid}>
          <Rule title="Rótulo persistente">
            Placeholder não substitui label.
          </Rule>
          <Rule title="Foco evidente">Todo controle responde ao teclado.</Rule>
          <Rule title="Erro associado">
            Mensagem, ajuda e campo compartilham IDs.
          </Rule>
          <Rule title="Alvo confortável">
            Ações possuem pelo menos 44 × 44 px.
          </Rule>
        </div>
      </section>

      <footer className={styles.footer}>
        Barbershop Design System <span aria-hidden="true">•</span> referência
        React disponível somente em desenvolvimento
      </footer>
    </main>
  );
}

function SectionHeading({
  description,
  number,
  title,
}: {
  description: string;
  number: string;
  title: string;
}) {
  return (
    <div className={styles.sectionHeading}>
      <div>
        <span>{number}</span>
        <h2>{title}</h2>
      </div>
      <p>{description}</p>
    </div>
  );
}

function Showcase({
  children,
  title,
}: {
  children: React.ReactNode;
  title: string;
}) {
  return (
    <article className={styles.showcase}>
      <h3>{title}</h3>
      {children}
    </article>
  );
}

function Rule({
  children,
  title,
}: {
  children: React.ReactNode;
  title: string;
}) {
  return (
    <article>
      <h3>{title}</h3>
      <p>{children}</p>
    </article>
  );
}

function ChevronIcon() {
  return (
    <svg aria-hidden="true" className={styles.inlineIcon} viewBox="0 0 24 24">
      <path d="m9 5 7 7-7 7" />
    </svg>
  );
}

function StarIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="m12 3 2.7 5.5 6.1.9-4.4 4.3 1 6.1-5.4-2.9-5.4 2.9 1-6.1-4.4-4.3 6.1-.9L12 3Z" />
    </svg>
  );
}
