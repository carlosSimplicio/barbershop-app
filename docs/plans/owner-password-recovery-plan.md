# Recuperação de senha do proprietário

Permitir que o proprietário com e-mail confirmado solicite um link de recuperação, defina uma senha diferente da atual e volte ao login, com todas as sessões anteriores revogadas. Decisões confirmadas nesta conversa: link válido por 1 hora; somente o link mais recente funciona; a recuperação não confirma e-mail nem cria sessão.

### 1. Persistir o ciclo de vida do token de recuperação

- **O que:** criar uma migration Goose aditiva para `password_recovery_tokens`, com vínculo ao usuário, hash do token, expiração, consumo, invalidação, restrição de um token ativo por usuário e índices para consulta e retenção futura. Adicionar as queries necessárias em SQL explícito e regenerar `sqlc`, sem editar código gerado.
- **Motivação:** garantir que o token de uso único não fique armazenado de forma utilizável, que solicitações posteriores substituam a anterior e que concorrência não produza dois links ativos.
- **Onde:** `backend/db/migrations/00004_owner_password_recovery.sql` (novo), `backend/db/query/password_recovery.sql` (novo), `backend/db/generated/`, `docs/architecture.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, seções “Proprietário e profissional” e “Retenção dos dados de autenticação”; `docs/operations.md`, seção “Manutenção dos dados de autenticação”; decisão confirmada nesta conversa sobre 1 hora e substituição do link. Atualizar documentos e checklist para incluir os novos tokens na retenção futura, sem declarar a limpeza como entregue.
- **Verificação:** migration aplica em PostgreSQL descartável e `sqlc` gera sem divergências; constraints impedem hashes duplicados e mais de um token ativo por usuário, e o schema não possui coluna para o token em claro. A emissão e o consumo são verificados por HTTP nas subtarefas 2 e 3.

### 2. Solicitar recuperação por e-mail na API

- **O que:** adicionar `POST /api/v1/auth/password/recovery` ao OpenAPI e implementar validação de e-mail, busca exclusiva de proprietário ativo com e-mail confirmado, emissão e substituição transacional do token, rate limiting por IP e e-mail normalizado em escopo próprio, resposta genérica `202` e envio pelo `mailer.Sender` existente. Falha de transporte após persistência mantém o `202` e gera apenas evento operacional sem dados sensíveis, conforme o padrão atual de confirmação. Versionar o template `pt-BR` com link cujo token aparece somente no fragmento; permitir a rota no middleware público, mantendo validação de `Origin`. Regenerar os artefatos OpenAPI afetados.
- **Motivação:** oferecer a solicitação sem revelar se o e-mail pertence a uma conta elegível, conter abuso e reutilizar o transporte já configurado com Mailpit no ambiente local.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/auth/` (novo serviço, handler e ajuste do middleware), `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/db/query/password_recovery.sql`, `backend/internal/auth/templates/password_recovery.pt-BR.txt` (novo), `backend/integration/`, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, seções “Autenticação”, “Proprietário e profissional” e “E-mail transacional”; `docs/implementation-scope.md`, itens 5, 21 e 22; `AGENTS.md`, regra universal de token no fragmento; decisões confirmadas nesta conversa sobre elegibilidade e substituição. Usar as janelas e respostas de rate limiting já implementadas, sem criar uma política nova.
- **Verificação:** testes unitários cobrem erro de validação para e-mail inválido; contas inexistentes, profissionais e proprietários não confirmados recebem o mesmo `202` genérico sem e-mail enviado. Cobrir substituição do token, limite e falha de envio sem exposição de dados. Integração HTTP com migrations reais verifica `202`, mensagem no fake de e-mail, token só no fragmento, persistência do hash, invalidação do token anterior no banco e `429` com `Retry-After`. Logs e erros não contêm e-mail completo, token nem URL. Desdobrar checkboxes mistos de `docs/implementation-scope.md` para marcar somente os comportamentos concluídos.
- **Dependências:** subtarefa 1.

### 3. Redefinir senha e revogar sessões na API

- **O que:** adicionar `POST /api/v1/auth/password/recovery/verify` ao OpenAPI. Validar token e nova senha no backend, revalidar sob bloqueio que o token pertence a um proprietário ativo com e-mail confirmado, recusar senha igual à atual, consumir o token uma única vez e, na mesma transação, atualizar o hash Argon2id, incrementar `credential_version` e revogar todas as sessões desse usuário. Responder sem criar sessão; tratar tokens inexistentes, expirados, invalidados ou usados de forma indistinguível. Regenerar os artefatos OpenAPI afetados.
- **Motivação:** fazer a nova senha substituir a credencial anterior sem deixar JWTs ou sessões antigos válidos, inclusive sob requisições concorrentes ou falha intermediária.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/auth/`, `backend/db/query/password_recovery.sql`, `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/integration/`, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, seções “Autenticação” e “Proprietário e profissional”; `docs/implementation-scope.md`, itens 5 e 22; decisões confirmadas nesta conversa sobre senha distinta e retorno ao login. Documentar essas decisões em `docs/product.md` e `docs/architecture.md` junto com a implementação.
- **Verificação:** testes unitários cobrem senha inválida ou igual à atual, token inválido, conta que perdeu elegibilidade e falha sem mudança parcial. Integração HTTP com PostgreSQL real cobre sucesso, replay, expiração, substituição, concorrência, login com senha antiga recusado, login com senha nova aceito e `GET /auth/me` de sessões anteriores retornando `401`; o reset não emite cookies de uma nova sessão nem altera contas de outro usuário ou estabelecimento.
- **Dependências:** subtarefas 1 e 2.

### 4. Solicitar recuperação pela interface

- **O que:** ligar a tela de login a uma rota pública de solicitação de recuperação, criar formulário com e-mail e cliente tipado para o novo endpoint e apresentar o mesmo sucesso genérico para contas elegíveis ou não. Usar os componentes existentes de `frontend/components/ui` e atualizar a referência do design system somente se surgir um estado ou variante visual novo.
- **Motivação:** permitir que o proprietário inicie o fluxo sem expor o estado da conta e com feedback claro para validação, limite e falha temporária.
- **Onde:** `frontend/app/(session)/login/page.tsx`, `frontend/components/auth/login-form.tsx`, nova rota `frontend/app/(session)/recuperar-senha/`, novo wrapper `frontend/lib/auth/password-recovery-api.ts`, testes Vitest correspondentes, `docs/product.md` e `docs/architecture.md`.
- **Base documental:** `docs/product.md`, seções “Cadastro, usuários e permissões” e “Experiência de uso”; `docs/architecture.md`, seção “Frontend”; `docs/implementation-scope.md`, item 4; decisão confirmada nesta conversa de restringir a recuperação ao proprietário com e-mail confirmado. A interface não deve prometer que o link foi enviado para uma conta específica.
- **Verificação:** Vitest cobre entrada pelo login, validação de conveniência, envio único, sucesso genérico, `429`, erro de rede e navegação por teclado; ESLint, Prettier e tipos passam. A atualização documental descreve a interface como entregue somente após esta subtarefa.
- **Dependências:** subtarefa 2.

### 5. Redefinir senha pela interface

- **O que:** criar a página pública do link de recuperação fora do layout que consulta `/auth/me`, capturar o token do fragmento e limpar fragmento e query antes de qualquer requisição ou navegação; enviar token e nova senha somente no corpo da API. Mostrar estados de link inválido ou expirado, senha inválida ou repetida, processamento e sucesso; depois do sucesso, orientar o login com a nova senha, sem autenticação automática. Reutilizar o design system e atualizar sua referência se necessário.
- **Motivação:** concluir o fluxo sem vazar o token por URL, `Referer` ou logs e refletir que as sessões anteriores foram revogadas.
- **Onde:** nova rota `frontend/app/redefinir-senha/`, novos componentes em `frontend/components/auth/`, `frontend/lib/auth/password-recovery-api.ts`, testes Vitest, `docs/product.md`, `docs/architecture.md`, `docs/operations.md`, `README.md` e `docs/implementation-scope.md`.
- **Base documental:** `AGENTS.md`, regra universal dos links transacionais; `docs/architecture.md`, seções “Autenticação” e “Frontend”; `docs/operations.md`, seções de Mailpit e validação local; `docs/implementation-scope.md`, itens 5, 21 e 22; decisões confirmadas nesta conversa sobre retorno ao login e ausência de sessão nova.
- **Verificação:** Vitest cobre limpeza da URL antes da chamada à API, token ausente, submissão única, erros de token e senha, sucesso sem sessão automática e retorno ao login. A validação local com Mailpit percorre solicitação, link, redefinição e novo login; atualizar somente os checkboxes cujos fluxos completos foram entregues.
- **Dependências:** subtarefas 3 e 4.

## Verificação conjunta

- Executar `make generate-check`, `make test-unit`, `make test-integration`, `make check` e `make build`; revisar `gofmt`, `go vet -tags=integration ./...`, ESLint, Prettier e TypeScript conforme `AGENTS.md`.
- Revisar o diff, o OpenAPI, os testes e os documentos afetados em conjunto; confirmar isolamento por papel e estabelecimento, revogação atômica, ausência de dados sensíveis em logs e URLs e crescimento da nova tabela registrado como pendência operacional de retenção.
- O provedor real de e-mail e a rotina de limpeza permanecem para seus escopos próprios, já pendentes em `docs/implementation-scope.md`; este plano usa o Mailpit local e não declara a recuperação pronta para produção antes dessas dependências operacionais.
