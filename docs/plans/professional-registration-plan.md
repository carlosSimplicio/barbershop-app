# Cadastro inicial de profissionais no painel

Permitir que o proprietário cadastre profissionais com nome, e-mail e senha temporária, consulte uma lista básica do próprio estabelecimento e entre no novo painel em `/painel/profissionais`. O painel pertence ao proprietário do estabelecimento. Decisões confirmadas nesta conversa: nome obrigatório, não vazio após remover espaços e com até 120 caracteres; cadastro e lista na página real do painel; proprietário segue para o painel após login ou confirmação, enquanto o profissional com senha temporária continua em `/testes`.

### 1. Persistir o perfil e as consultas de profissionais

- **O que:** criar uma migration Goose aditiva para `professionals`, com `user_id` como identidade do profissional, `establishment_id`, nome e vínculo composto `(user_id, establishment_id)` a `users`. Adicionar a `backend/db/query/professionals.sql` as escritas de usuário, credencial e perfil e a leitura filtrada por estabelecimento; regenerar `sqlc`.
- **Motivação:** guardar o nome que identificará o profissional no painel e na agenda futura, preservando a unicidade global do e-mail já aplicada em `users` e impedindo vínculo entre tenants distintos.
- **Onde:** nova `backend/db/migrations/00005_professionals.sql`, novo `backend/db/query/professionals.sql`, `backend/db/generated/`, `backend/integration/`.
- **Base documental:** `docs/architecture.md`, seções “Banco de dados” e “Multi-tenancy e página pública”; `docs/implementation-scope.md`, itens 3, 6 e 9; decisão desta conversa de exigir nome. Atualizar a modelagem descrita em `docs/architecture.md` junto com a implementação.
- **Requisitos de entrega:**
  - A migration nova cria um perfil com `user_id` único, nome obrigatório e `establishment_id` para cada profissional.
  - O vínculo composto rejeita um perfil cujo estabelecimento difira do estabelecimento do usuário.
  - As queries geradas permitem inserir usuário, credencial e perfil na mesma transação e listar perfis por estabelecimento com limite e ordem estável.
  - `docs/architecture.md` descreve a modelagem entregue; as migrations anteriores permanecem intactas.
- **Verificação:** aplicar as migrations em PostgreSQL descartável, verificar a rejeição do vínculo cruzado, regenerar `sqlc` e conferir a documentação. As subtarefas 3 e 4 exercitam as queries pelos fluxos HTTP.

### 2. Centralizar o direito de escrita do estabelecimento

- **O que:** criar um componente de autorização de escrita no backend que consulte o estado persistido do estabelecimento e conceda escrita durante o trial ativo. Sem períodos de assinatura implementados, trial ausente ou expirado resulta em leitura somente; a resolução fica em um lugar para receber assinaturas na fase comercial. A consulta da lista permanece disponível após expiração. Regenerar `sqlc` para a consulta adicionada.
- **Motivação:** impedir que o primeiro endpoint autenticado de mutação de domínio ignore a regra de acesso por trial ou assinatura.
- **Onde:** novo pacote em `backend/internal/` para direitos de acesso, query em `backend/db/query/`, `backend/db/generated/` e testes unitários do pacote.
- **Base documental:** `docs/product.md`, seção “Planos, assinatura e trial”; `docs/architecture.md`, seção “Assinaturas da plataforma”; `docs/implementation-scope.md`, itens 6 e 20. Documentar em `docs/architecture.md` que a fonte implementada nesta etapa é somente o trial, sem declarar assinaturas como entregues.
- **Requisitos de entrega:**
  - O direito de escrita é concedido apenas no intervalo do trial ativo, incluindo o início e excluindo o instante final.
  - Trial ausente ou expirado produz recusa de escrita sem impedir leitura autenticada.
  - A decisão fica centralizada em um componente reutilizável; a implementação não declara suporte a períodos de assinatura ainda inexistentes.
  - `docs/architecture.md` distingue o direito já implementado pelo trial da extensão futura para assinaturas.
- **Verificação:** testes unitários avaliam ausência de trial e as fronteiras temporais; as subtarefas 3 e 4 verificam por HTTP a recusa de escrita e a leitura após expiração.

### 3. Cadastrar profissional pela API

- **O que:** adicionar `POST /api/v1/professionals` ao OpenAPI e implementar validação backend de nome, e-mail e `temporary_password`, usando as regras de e-mail e senha existentes. Exigir sessão de `owner`, `Origin` e CSRF válidos e direito de escrita; derivar o tenant do `Principal`. Criar usuário ativo com papel `professional`, `password_change_required=true`, credencial Argon2id e perfil em uma transação. Responder `201` com dados não secretos; tratar e-mail globalmente duplicado como conflito de campo. Regenerar os tipos Go e TypeScript.
- **Motivação:** substituir a criação direta de profissionais no banco por um fluxo autenticado e permitir o primeiro login com troca obrigatória já implementada.
- **Onde:** `backend/openapi/openapi.yaml`, novo slice em `backend/internal/professionals/`, `backend/internal/auth/middleware.go`, `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/db/query/professionals.sql`, `backend/integration/`, `backend/openapi/generated/`, `frontend/types/generated/api.ts`.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Autenticação” e “Proprietário e profissional”; `docs/implementation-scope.md`, itens 5, 6, 9 e 22; decisões desta conversa sobre os três campos e a regra do nome. Atualizar `docs/product.md`, `docs/architecture.md` e dividir os itens mistos do checklist para marcar apenas o cadastro entregue.
- **Requisitos de entrega:**
  - `POST /api/v1/professionals` aceita nome não vazio após remover espaços e com até 120 caracteres, e-mail válido e senha temporária com a regra de senha existente; devolve erros de campo para entradas inválidas.
  - Somente o `owner` autenticado, com `Origin` e CSRF válidos e direito de escrita ativo, cria profissional no estabelecimento derivado da sessão.
  - Uma solicitação válida cria usuário ativo, credencial Argon2id e perfil na mesma transação, com `password_change_required=true`, e responde `201` sem devolver a senha.
  - E-mail já cadastrado em qualquer estabelecimento gera conflito de campo; falha em qualquer escrita não deixa cadastro parcial.
  - O profissional criado consegue entrar com a senha temporária e permanece restrito à troca obrigatória antes de acessar recursos de domínio.
  - OpenAPI, artefatos gerados, `docs/product.md`, `docs/architecture.md` e os itens concluídos do checklist refletem o endpoint entregue sem expor senha em logs.
- **Verificação:** testes unitários cobrem validação e autorização; integração HTTP com migrations reais cobre criação, rollback, duplicidade, `401`, `403`, CSRF, trial expirado, login e troca de senha. Conferir contrato gerado, resposta e logs em busca de dados sensíveis.
- **Dependências:** subtarefas 1 e 2.

### 4. Listar somente os profissionais do proprietário

- **O que:** adicionar `GET /api/v1/professionals` ao OpenAPI e implementar leitura autenticada exclusiva do proprietário, filtrada pelo `establishment_id` do `Principal`, com ordem estável e paginação limitada. Retornar apenas identificador, nome, e-mail e estado ativo; nunca retornar hash, senha ou dados de outro estabelecimento. Regenerar os tipos Go e TypeScript.
- **Motivação:** permitir reencontrar os cadastros na página administrativa sem fazer uma consulta sem limite nem expor profissionais de outro tenant.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/professionals/`, `backend/internal/httpapi/handler.go`, `backend/db/query/professionals.sql`, `backend/integration/`, `backend/openapi/generated/`, `frontend/types/generated/api.ts`.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Multi-tenancy e página pública” e “Assinaturas da plataforma”; `docs/implementation-scope.md`, itens 6 e 9; decisão desta conversa de incluir lista básica. Atualizar contrato e documentação na mesma entrega.
- **Requisitos de entrega:**
  - `GET /api/v1/professionals` permite somente ao proprietário consultar profissionais do estabelecimento derivado da sessão.
  - A resposta inclui apenas identificador, nome, e-mail e estado ativo, com ordem estável e paginação limitada.
  - A consulta retorna lista vazia quando não há profissionais e continua disponível com trial expirado.
  - Contrato, tipos gerados e documentação descrevem a leitura entregue sem expor hash ou senha.
- **Verificação:** integração HTTP cobre lista vazia, mais de uma página, dois estabelecimentos, recusa a profissional, trial expirado e ausência de material de credencial na resposta.
- **Dependências:** subtarefa 1.

### 5. Incorporar uma lista com estado vazio ao design system

- **O que:** criar um padrão reutilizável de lista de registros com estado vazio em `frontend/components/ui`, com semântica e foco acessíveis, variantes claro e escuro e apresentação responsiva. Documentá-lo na referência React do design system.
- **Motivação:** a nova página precisa apresentar uma coleção e seu estado vazio sem estabelecer um padrão visual isolado do restante do produto.
- **Onde:** novo componente em `frontend/components/ui/`, `frontend/components/ui/index.ts`, `frontend/app/design-system/` e testes Vitest co-localizados.
- **Base documental:** `AGENTS.md`, regra universal do design system; `docs/architecture.md`, seção “Frontend”; `docs/implementation-scope.md`, item 4. Marcar o padrão de estado vazio somente quando estiver entregue e documentado.
- **Requisitos de entrega:**
  - Um componente reutilizável apresenta cada registro como item de uma lista sem depender de dados específicos de profissionais.
  - Quando não há registros, o componente anuncia uma mensagem de estado vazio e não renderiza itens fictícios.
  - Lista e estado vazio usam os tokens existentes, permanecem legíveis nos temas claro e escuro e cabem em telas estreitas sem rolagem horizontal.
  - A referência React do design system mostra lista preenchida e vazia; o checklist marca apenas os padrões concluídos.
- **Verificação:** Vitest cobre conteúdo e estado vazio com consultas por papel e rótulo; conferir a referência do design system em ambos os temas e em viewport estreita.

### 6. Abrir a área real do painel para o proprietário

- **O que:** criar a casca reutilizável e a rota `/painel/profissionais` no App Router, protegidas por sessão e papel de `owner`, usando o `AuthProvider` existente. Direcionar proprietário autenticado para essa rota após login e confirmação; manter profissional com senha temporária em `/testes` e impedir que acesse o painel. Preservar `/testes` como área temporária de diagnóstico acessível pela navegação apropriada.
- **Motivação:** fazer do cadastro uma função do painel administrativo do estabelecimento, com caminho real de entrada após autenticação.
- **Onde:** nova rota em `frontend/app/(session)/painel/`, guarda em `frontend/components/session/`, `frontend/components/auth/login-form.tsx`, `frontend/components/auth/email-verification.tsx`, `frontend/components/session/session-guard.tsx`, testes Vitest, `docs/product.md`, `docs/architecture.md`, `docs/operations.md` e `README.md`.
- **Base documental:** `docs/product.md`, seção “Experiência de uso”; `docs/architecture.md`, seções “Autenticação” e “Frontend”; decisão desta conversa que substitui o destino `/testes` documentado para o proprietário e estabelece `/painel/profissionais`. Atualizar os documentos e `docs/operations.md` no mesmo trabalho.
- **Requisitos de entrega:**
  - `/painel/profissionais` existe dentro de uma casca reutilizável do painel do proprietário, com link de navegação alcançável por teclado em desktop e tela menor.
  - Sessão ausente leva ao login; usuário com papel `professional` não vê o conteúdo do painel.
  - Login e confirmação do proprietário levam ao painel; profissional com senha temporária continua em `/testes`.
  - `/testes` permanece acessível para diagnóstico, e produto, arquitetura, operação e README descrevem os destinos reais.
- **Verificação:** Vitest cobre os destinos após login e confirmação e as guardas por sessão e papel; conferir navegação por teclado em desktop e tela menor.

### 7. Mostrar a lista básica no painel

- **O que:** conectar a página ao `GET /professionals` pelo cliente tipado e mostrar nome, e-mail e estado ativo, com carregamento, vazio, erro e paginação. Cancelar consultas obsoletas e impedir que dados da sessão anterior apareçam após troca de identidade.
- **Motivação:** permitir ao proprietário verificar e reencontrar os profissionais já cadastrados.
- **Onde:** `frontend/app/(session)/painel/profissionais/`, novo cliente em `frontend/lib/professionals/`, testes Vitest, `docs/product.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, seções “Frontend” e “Estado”; `docs/implementation-scope.md`, itens 4 e 9; decisão desta conversa de incluir lista básica. Atualizar a documentação do estado real da página.
- **Requisitos de entrega:**
  - A página consulta o endpoint tipado e apresenta nome, e-mail e estado ativo dos profissionais retornados, com navegação entre páginas.
  - Carregamento, lista vazia e falha de consulta possuem estados visuais distintos e acessíveis.
  - Uma resposta antiga não substitui dados de consulta ou identidade mais recentes; `401` remove o conteúdo protegido.
  - A lista continua legível quando o trial expira, e a documentação descreve essa leitura.
- **Verificação:** Vitest cobre dados, vazio, paginação, falha de rede, `401`, trial expirado e descarte de resposta obsoleta após troca de identidade.
- **Dependências:** subtarefas 4, 5 e 6.

### 8. Cadastrar pela página do painel

- **O que:** adicionar formulário de nome, e-mail e senha temporária à página, com cliente tipado para `POST /professionals`, validação de conveniência, erros de campo, bloqueio de envio duplicado e atualização da lista após `201`. Limpar a senha do estado da interface após sucesso e informar que ela não poderá ser recuperada. Usar os componentes do design system; documentar nele qualquer estado novo que se mostrar necessário.
- **Motivação:** completar o fluxo do proprietário sem expor a senha na URL, em logs, na resposta da API ou na listagem.
- **Onde:** `frontend/app/(session)/painel/profissionais/`, novo formulário em `frontend/components/professionals/`, `frontend/lib/professionals/`, testes Vitest, `docs/product.md`, `docs/operations.md`, `README.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/product.md`, seções “Cadastro, usuários e permissões” e “Experiência de uso”; `docs/architecture.md`, seções “Proprietário e profissional” e “Frontend”; `docs/implementation-scope.md`, itens 5 e 9; decisões desta conversa sobre campos e página. Atualizar a validação manual em `docs/operations.md` para usar o fluxo real em vez de preparar uma conta no banco.
- **Requisitos de entrega:**
  - O formulário coleta nome, e-mail e senha temporária e mostra erros de conveniência sem substituir a validação do backend.
  - O envio usa o cliente tipado com CSRF, impede submissões simultâneas e apresenta conflito de e-mail e falhas temporárias sem mensagens brutas do servidor.
  - Após `201`, o novo profissional aparece na lista; a senha sai do estado da interface e não aparece na URL, na resposta nem na lista.
  - Com trial expirado, o proprietário pode ler a lista, mas não consegue concluir o cadastro; a interface explica a recusa.
  - `docs/operations.md` e `README.md` descrevem o fluxo real de cadastro, login do profissional e troca obrigatória; os checkboxes marcam somente o comportamento entregue.
- **Verificação:** Vitest cobre validação, conflito, falha de rede, trial expirado, envio único e sucesso com atualização da lista e limpeza da senha. Validação manual local percorre cadastro, login e troca obrigatória.
- **Dependências:** subtarefas 3, 6 e 7.

## Verificação conjunta

- Executar `make generate-check`, `make check` e `make build`; `make check` inclui `gofmt`, `go vet -tags=integration ./...`, testes unitários, integração do backend, ESLint, Prettier e tipos.
- Revisar diff, OpenAPI, migrations, testes e documentos em conjunto; confirmar isolamento, papel, CSRF, trial expirado, atomicidade e ausência de senha em respostas, logs e fixtures públicas.
- Dividir ou atualizar os checkboxes mistos de `docs/implementation-scope.md` conforme o comportamento realmente entregue. Administração completa, reativação, comissões e associação com serviços continuam nas fases posteriores; não marcar esses itens como concluídos.
