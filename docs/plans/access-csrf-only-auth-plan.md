# Autenticação somente com access token e CSRF por 15 dias

Remover o refresh token e a renovação da sessão. Access JWT e CSRF terão validade fixa de 15 dias; o logout continuará revogando imediatamente a sessão atual no backend. Tokens e sessões locais emitidos pelo modelo anterior podem ser invalidados.

### 1. Remover a renovação do frontend

- **O que:** eliminar a chamada automática e manual a `POST /auth/refresh`, Web Lock, deduplicação de rotação e painel de renovação em `/testes`. Manter descoberta por `GET /auth/me`, guardas, sincronização após login, confirmação, troca de senha e logout, e envio de CSRF nas mutações. Atualizar a descrição da interface em `docs/product.md`, da sessão no frontend em `docs/architecture.md` e os itens correspondentes de `docs/implementation-scope.md`.
- **Motivação:** retirar a coordenação de concorrência criada pelo refresh rotativo. Enquanto o backend ainda emitir access de 15 minutos, um `401` exigirá novo login; a subtarefa 3 ampliará esse prazo.
- **Onde:** `frontend/lib/api/client.ts`, `frontend/lib/api/session.ts`, `frontend/lib/auth/session-api.ts`, `frontend/components/session/auth-provider.tsx`, `frontend/components/testing/manual-refresh-panel.tsx`, `frontend/app/(session)/testes/page.tsx`, testes relacionados e documentos citados.
- **Base documental:** decisão desta conversa; `docs/product.md`, seção “Experiência de uso”; `docs/architecture.md`, seção “Contrato de sessão”; `docs/implementation-scope.md`, seção 5, todos a atualizar para o novo fluxo.
- **Verificação:** `401` em leitura autenticada leva ao estado sem sessão sem retry nem chamada de refresh; falha de rede ou `5xx` continua indicada como indisponibilidade; ações explícitas não deixam respostas antigas sobrescrever o principal novo; `/testes` mantém consulta, troca de senha e logout. Testes Vitest protegem esses fluxos.

### 2. Fazer o CSRF expirar no backend após 15 dias

- **O que:** incluir prazo assinado no token CSRF e validá-lo no backend; ajustar o cookie CSRF para `Max-Age` de 15 dias, preservando `Secure`, `SameSite=Strict`, `Path=/`, comparação com `X-CSRF-Token` e validação de `Origin`. Atualizar `docs/architecture.md` no mesmo trabalho.
- **Motivação:** hoje somente o cookie expira, após 7 dias; a assinatura aceita indefinidamente um valor reapresentado. A validade de 15 dias deve valer também para o verificador.
- **Onde:** `backend/internal/auth/security/token.go`, `backend/internal/auth/middleware.go`, `backend/internal/auth/session_response.go`, testes correspondentes e `docs/architecture.md`.
- **Base documental:** confirmação desta conversa, que substitui o comportamento descrito em `docs/architecture.md`, seções “Autenticação” e “Contrato de sessão”.
- **Verificação:** CSRF válido é aceito até seu prazo; expirado, alterado, ausente ou diferente do header é recusado; cookie tem `Max-Age` de 15 dias. Cobrir em testes Go e integração HTTP com relógio controlado.

### 3. Substituir o ciclo de sessão no backend e no contrato

- **O que:** em uma mudança coesa, emitir access JWT com ID de sessão e expiração fixa de 15 dias no login, na confirmação de e-mail e na troca de senha temporária; criar as sessões sem refresh token; validar em cada requisição o JWT, a sessão não revogada/não expirada e usuário, estabelecimento, papel e versão da credencial. Revogar no logout somente a sessão identificada pelo access token e limpar access e CSRF; com access ausente ou inválido, limpar cookies e responder idempotentemente após CSRF válido. Remover `POST /auth/refresh`, o cookie de refresh, emissão, rotação, detecção de reutilização e todo fallback para tokens antigos. Renomear `idle_expires_at` para refletir expiração fixa e retirar `refresh_tokens` em migration Goose nova; remover queries obsoletas. Atualizar OpenAPI e regenerar `sqlc`, `oapi-codegen` e `openapi-typescript`. Sincronizar `docs/architecture.md`, `docs/operations.md`, `README.md` e os checkboxes afetados de `docs/implementation-scope.md` nesta subtarefa.
- **Motivação:** a revogação imediata de uma sessão específica exige seu ID no access JWT e consulta ao backend. Login, confirmação, troca de senha e logout precisam adotar o mesmo contrato de uma vez, sem suporte transitório a refresh ou JWT antigo.
- **Onde:** `backend/internal/auth/`, `backend/internal/onboarding/confirmation_verification.go`, `backend/internal/onboarding/store.go`, `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/db/query/authentication.sql`, nova migration em `backend/db/migrations/`, `backend/openapi/openapi.yaml`, artefatos gerados, testes unitários e `backend/integration/`.
- **Base documental:** decisão desta conversa, que substitui access de 15 minutos e refresh deslizante de 7 dias em `docs/architecture.md`, seções “Autenticação”, “Contrato de sessão” e “Retenção dos dados de autenticação”; `docs/product.md`, seções “Cadastro, usuários e permissões” e “Planos, assinatura e trial”; `AGENTS.md`, regras de isolamento, migrations e código gerado.
- **Verificação:** login e confirmação emitem somente access e CSRF; confirmação mantém consumo único do token, trial e emissão da sessão na mesma transação; troca de senha revoga todas as sessões anteriores e emite uma nova; logout revoga a sessão atual sem afetar outra sessão do usuário; JWT de outro tenant, papel ou sessão é recusado; access expira após 15 dias sem prorrogação por uso; `/auth/refresh` não existe no contrato nem no roteador. Testes de integração exercitam HTTP, cookies, transações e PostgreSQL reais. A migration apaga os hashes de refresh existentes: sem backup, seu `Down` não poderá recuperá-los; nenhum dado de agendamento ou financeiro é afetado.
- **Dependências:** 1 e 2.

### 4. Revisar a entrega completa

- **O que:** conferir diff, fluxos, testes, OpenAPI, documentação e checklist; identificar os planos históricos `docs/plans/authentication-plan.md` e `docs/plans/frontend-auth-pages-plan.md` como substituídos por este plano na parte de refresh, sem apagar seu registro de implementação.
- **Motivação:** impedir que o modelo antigo permaneça apresentado como contrato vigente ou que a simplificação introduza regressões de autorização, tenant ou tratamento de erros.
- **Onde:** documentos citados, `docs/implementation-scope.md`, OpenAPI e fluxos de autenticação afetados.
- **Base documental:** `AGENTS.md`, seções “Documentação obrigatória por contexto” e “Revisão final de features”; decisão desta conversa.
- **Verificação:** código, testes, contrato, documentação e checkboxes descrevem somente access e CSRF de 15 dias; instruções manuais percorrem confirmação, `/auth/me`, logout e novo login sem refresh; nenhum token aparece em URL, logs ou armazenamento web.
- **Dependências:** 1 a 3.

## Verificação conjunta

- Executar `make generate-check`, testes unitários Go e Vitest, testes de integração do backend, `gofmt`, `go vet -tags=integration ./...`, ESLint, Prettier, checagem de tipos e builds de produção.
- Revisar comportamento em abas simultâneas, logout com access expirado, bloqueio de profissional com senha temporária e isolamento entre estabelecimentos. Tokens e sessões locais anteriores podem ser invalidados imediatamente pela mudança.
