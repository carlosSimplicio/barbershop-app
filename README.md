# Barbershop App

Monorepo da plataforma multi-tenant de agendamento e gestão financeira para
barbearias e estabelecimentos semelhantes.

## Stack fixada

- Go 1.27.2, Gin 1.12.0, pgx 5.11.0, sqlc 1.31.1, Goose 3.29.0 e
  oapi-codegen 2.8.0.
- Node.js 24.21.0 LTS, Next.js 16.4.2, React 19.3.1, TypeScript 5.9.3 e
  Vitest 5.0.5.
- PostgreSQL 18.6, Caddy 2.11.7 e Mailpit 1.31.5.

As versões são exatas e atualizadas deliberadamente. Não use tags flutuantes
nas imagens ou intervalos de versão nas dependências.

## Início rápido

Pré-requisitos: Docker com Compose e `make`.

```sh
cp .env.example .env
make dev
```

A aplicação fica em <http://localhost:8088> e o Mailpit em
<http://localhost:8025>. Quando houver migrations de domínio, aplique-as em
outro terminal:

```sh
make migrate-up
```

O Caddy é o único ponto público da aplicação. Ele encaminha `/api/v1` para o
backend e as demais rotas para o Next.js. PostgreSQL, backend, frontend e SMTP
permanecem nas redes internas do Compose.

## Desenvolvimento sem Compose

Instale as dependências e gere os artefatos:

```sh
make setup
make generate
```

Para executar as aplicações separadamente, configure `DATABASE_URL` para um
PostgreSQL acessível pelo host e use:

```sh
cd backend && go run ./cmd/api
cd frontend && npm run dev
```

O contrato é alterado primeiro em `backend/openapi/openapi.yaml`. `make generate`
atualiza o servidor Go, as consultas sqlc e os tipos TypeScript. Os diretórios
`backend/openapi/generated`, `backend/db/generated` e
`frontend/types/generated` não devem ser editados manualmente.

## Verificações

```sh
make test           # testes unitários de backend e frontend
make check          # geração, formato, vet, lint, tipos e testes
make build          # builds locais de produção
make compose-config # valida a composição de desenvolvimento
```

O gate de pull request executa somente os testes unitários, conforme o escopo
atual. As verificações completas rodam na branch principal e podem ser
executadas localmente antes de uma release.

## Estrutura

```text
backend/             API Go, contrato OpenAPI, migrations e SQL
frontend/            aplicação Next.js e tipos gerados do contrato
infra/caddy/         proxy reverso e terminação HTTPS
docs/                produto, arquitetura, operação e escopo vivo
```

Configurações reais ficam apenas no `.env` local ou na VPS. Nunca versione
senhas, tokens ou outros dados sensíveis.
