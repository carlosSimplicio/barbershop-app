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
make auth-keys # copie a saída para as variáveis correspondentes do .env
make dev
```

`LOGO_STORAGE_DIR` aponta para `/var/lib/barbershop/logos` dentro do container.
O Compose inicializa o volume persistente e prepara suas permissões antes de
iniciar a API; o processo do backend continua executando como usuário não
privilegiado `app`. Preserve o volume `logos_data` ao recriar ou atualizar os
containers.

A aplicação fica em <https://localhost:8443> e o Mailpit em
<http://localhost:8025>. Quando houver migrations de domínio, aplique-as em
outro terminal:

```sh
make migrate-up
```

O Caddy usa uma CA interna no ambiente local. Depois do primeiro `make dev`,
use outro terminal para copiar o certificado e instale-o manualmente como
autoridade confiável no sistema ou navegador:

```sh
docker compose -f compose.yaml -f compose.dev.yaml cp \
  caddy:/data/caddy/pki/authorities/local/root.crt \
  /tmp/barbershop-caddy-root.crt
```

No Linux, a instalação normalmente consiste em copiar esse arquivo para
`/usr/local/share/ca-certificates/barbershop-caddy-root.crt` e executar
`sudo update-ca-certificates`. Outros sistemas e alguns navegadores possuem
repositório próprio. Nunca versione a CA ou as chaves exibidas por
`make auth-keys`, e gere valores diferentes em produção.

O Caddy é o único ponto público da aplicação. Ele encaminha `/api/v1` para o
backend e as demais rotas para o Next.js. PostgreSQL, backend, frontend e SMTP
permanecem nas redes internas do Compose.

## Onboarding local

Com o ambiente local ativo e as migrations aplicadas, abra
<https://localhost:8443>, siga para `/cadastro`, use um e-mail fictício
exclusivo, confira o link no Mailpit e abra-o no navegador. A confirmação
inicia a sessão e leva o proprietário a `/painel/profissionais`. Nessa página,
cadastre um profissional com e-mail fictício exclusivo e senha temporária; a
lista é atualizada após o cadastro e o campo da senha é limpo. Compartilhe a
senha apenas por um canal seguro, pois ela não pode ser recuperada. Saia da
sessão do proprietário e entre com a conta profissional: ela abre `/testes` e
exige a troca da senha antes de acessar recursos de domínio. Use o link “Área
de testes” para consultar a sessão e o estabelecimento e sair. Depois, `/login`
leva o proprietário de volta ao painel. Com trial expirado, a lista continua
disponível, mas novos cadastros ficam bloqueados.

Para validar a recuperação do proprietário, saia da sessão e, em `/login`, abra
“Esqueci minha senha”. Informe o e-mail confirmado, confira a mensagem mais
recente no Mailpit e abra seu link `/redefinir-senha#token=...` no navegador. A
página remove o token do endereço antes de apresentar o formulário. Defina uma
senha diferente da anterior, siga o link de volta ao login e entre com a nova
senha. O link anterior não pode ser reutilizado e as sessões anteriores deixam
de funcionar. Use apenas contas e senhas fictícias nesse roteiro.

Login e confirmação emitem apenas os cookies access e CSRF, ambos com
`Max-Age` de 15 dias. A API valida a sessão identificada no access JWT em cada
requisição autenticada; não há renovação nem endpoint de refresh. Ao receber
`401`, a interface encerra a sessão local. O logout revoga a sessão atual no
backend e limpa os cookies.

O roteiro abaixo continua disponível para diagnosticar o onboarding pela API.
O `.env.example` já aponta `EMAIL_PROVIDER=smtp`,
`SMTP_HOST=mailpit` e a porta
`1025` para o Mailpit de desenvolvimento; essa configuração não é aceita como
provedor real de produção. Todas as mutações precisam da origem HTTPS confiável:

```sh
curl --fail-with-body --silent --show-error --insecure \
  -H 'Origin: https://localhost:8443' \
  -H 'Content-Type: application/json' \
  --data '{"establishment_name":"Barbearia Exemplo","slug":"barbearia-exemplo-01","time_zone":"America/Sao_Paulo","email":"owner-onboarding@example.test","password":"senha-segura"}' \
  https://localhost:8443/api/v1/auth/register

curl --fail-with-body --silent --show-error --insecure \
  -H 'Origin: https://localhost:8443' \
  -H 'Content-Type: application/json' \
  --data '{"email":"owner-onboarding@example.test"}' \
  https://localhost:8443/api/v1/auth/email-confirmation
```

As duas respostas são `202` sem corpo. Abra <http://127.0.0.1:8025>, use a
mensagem mais recente e copie somente o valor depois de
`/confirmar-email#token=`. A segunda solicitação invalida o primeiro token.
Leia o token sem ecoá-lo, crie um cookie jar temporário e confirme:

```sh
read -r -s CONFIRMATION_TOKEN
COOKIE_JAR="$(mktemp)"
printf '{"token":"%s"}' "$CONFIRMATION_TOKEN" | curl \
  --fail-with-body --silent --show-error --insecure \
  -H 'Origin: https://localhost:8443' \
  -H 'Content-Type: application/json' \
  --data-binary @- \
  -c "$COOKIE_JAR" \
  https://localhost:8443/api/v1/auth/email-confirmation/verify

curl --fail-with-body --silent --show-error --insecure \
  -b "$COOKIE_JAR" \
  https://localhost:8443/api/v1/establishment

unset CONFIRMATION_TOKEN
rm -f "$COOKIE_JAR"
```

A confirmação responde `200`, autentica o proprietário e inicia o trial. A
consulta seguinte retorna somente o estabelecimento da sessão, com o fuso
persistido e limites do trial em UTC. Não grave o token, o link completo ou o
cookie jar em logs, arquivos versionados ou histórico de comandos.

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
make test           # testes unitários e integração do backend, mais testes do frontend
make test-unit      # testes unitários de backend e frontend, sem Docker
make test-integration # fluxos do backend contra PostgreSQL descartável; requer Docker
make check          # geração, formato, vet, lint, tipos e testes
make build          # builds locais de produção
make compose-config # valida a composição de desenvolvimento
```

O gate de pull request executa os testes unitários e a integração do backend.
As verificações completas rodam na branch principal e podem ser executadas
localmente antes de uma release.

## Estrutura

```text
backend/             API Go, contrato OpenAPI, migrations e SQL
frontend/            aplicação Next.js e tipos gerados do contrato
infra/caddy/         proxy reverso e terminação HTTPS
docs/                produto, arquitetura, operação e escopo vivo
```

Configurações reais ficam apenas no `.env` local ou na VPS. Nunca versione
senhas, tokens ou outros dados sensíveis.
