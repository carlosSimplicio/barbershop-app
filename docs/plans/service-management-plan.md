# Cadastro e gestão de serviços

Permitir ao proprietário cadastrar, consultar, editar, ativar e desativar serviços pelo painel, com preço exato em BRL, isolamento por estabelecimento e leitura após expiração do trial.

### 1. Persistir o catálogo de cada estabelecimento

- **O que:** criar uma migration Goose nova para `services` e as queries compartilhadas do catálogo; regenerar `sqlc`.
- **Motivação:** armazenar serviços com identidade estável e preço e duração próprios, preparando referências históricas sem exclusão física.
- **Onde:** nova migration em `backend/db/migrations/`, novo `backend/db/query/services.sql`, `backend/db/generated/`, novos testes em `backend/integration/`, `docs/architecture.md` e `docs/implementation-scope.md`, item 3.
- **Base documental:** `docs/architecture.md`, “Banco de dados”; `docs/product.md`, “Localização”, “Agenda e disponibilidade” e “Serviços e profissionais inativos”.
- **Requisitos de entrega:**
  - `services` persiste UUID aleatório, `establishment_id`, `name`, `price_cents`, `duration_minutes`, `active`, `created_at` e `updated_at`; preço usa inteiro de centavos, sem ponto flutuante.
  - A FK para estabelecimento impede registros órfãos e não usa exclusão em cascata; não existe unicidade por nome.
  - Consultas e alterações por ID também exigem `establishment_id`; a listagem possui índice por estabelecimento, criação e ID.
  - Limites de nome, preço e duração ficam no backend Go, conforme a arquitetura, sem novos `CHECK` constraints para essas regras de produto.
  - A documentação registra a persistência entregue; associações profissional-serviço, agenda e financeiro continuam pendentes. Cadastro de serviço não preenche o marcador de início dos dados operacionais.
- **Verificação:** integração com migrations reais verifica persistência exata de centavos, vínculo obrigatório ao estabelecimento e consultas restritas ao tenant; conferir geração do `sqlc`.

### 2. Validar os dados do serviço no domínio

- **O que:** criar validações puras compartilhadas por cadastro e edição no novo slice `backend/internal/services/`.
- **Motivação:** aplicar as decisões confirmadas nesta conversa de forma uniforme, sem depender da interface ou de PostgreSQL.
- **Onde:** novo `backend/internal/services/`, testes Go co-localizados e `docs/product.md`.
- **Base documental:** `docs/product.md`, “Agenda e disponibilidade”; `docs/architecture.md`, “Backend”; decisões confirmadas pelo usuário sobre nome, preço e duração. Incorporar essas decisões ao produto nesta subtarefa.
- **Requisitos de entrega:**
  - Nome obrigatório após remover espaços nas extremidades, com até 120 pontos de código Unicode no valor normalizado; nomes repetidos são permitidos.
  - Preço obrigatório entre 0 e 9.999.999 centavos, equivalente a R$ 99.999,99; entradas fracionárias em centavos são recusadas, sem arredondamento silencioso.
  - Duração aceita inteiros de 1 a 1.440 minutos, inclusive valores que não sejam múltiplos de 15; ausência de duração no cadastro usa 30 minutos.
  - As validações produzem erros de campo estáveis e permanecem testáveis sem Gin, rede ou banco.
- **Verificação:** testes unitários cobrem nome vazio e limites com Unicode, preço zero e limites, duração padrão, duração de 17 minutos e recusas fora dos intervalos.

### 3. Cadastrar serviço pela API

- **O que:** definir `POST /api/v1/services` no OpenAPI e implementar cadastro no slice, com handler, persistência e wiring na aplicação; regenerar Go e TypeScript.
- **Motivação:** permitir ao proprietário criar um serviço ativo no próprio estabelecimento, sob a política centralizada de escrita.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/services/`, `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/internal/auth/middleware.go`, `backend/integration/suite_test.go`, novos testes em `backend/integration/`, artefatos gerados, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/product.md`, “Cadastro, usuários e permissões”, “Planos, assinatura e trial” e “Agenda e disponibilidade”; checklist, itens 6, 10 e 22; decisões confirmadas sobre os campos.
- **Requisitos de entrega:**
  - A entrada contém `name`, `price_cents` e duração opcional `duration_minutes`; duração omitida ou `null` usa 30 minutos. Resposta `201` contém ID, nome normalizado, preço em centavos, duração efetiva e `active: true`.
  - Apenas `owner` autenticado com origem confiável, CSRF válido e `entitlements.Service.RequireWrite` satisfeito cria serviço; tenant deriva exclusivamente do `Principal`.
  - Entrada inválida produz `400` com erros de campo; sessão ausente produz `401`, papel indevido produz `403` e trial ausente ou expirado produz `403` com `read_only`, sem gravação.
  - Repetir um nome é permitido, inclusive dentro do mesmo estabelecimento; não há associação automática a profissionais nem publicação do catálogo nesta entrega.
  - Contrato e documentação registram o endpoint entregue; dividir o item misto de gestão do checklist para distinguir API e painel, sem marcar o fluxo completo antes da interface.
- **Verificação:** unitários protegem validação e recusas; integração HTTP cobre criação, valores gratuitos, duração padrão, nomes repetidos, isolamento entre duas sessões de proprietários, profissional com troca pendente e após troca, CSRF, origem e trial expirado. Conferir ausência de dados do corpo em logs.
- **Dependências:** subtarefas 1 e 2.

### 4. Listar o catálogo com paginação

- **O que:** definir e implementar `GET /api/v1/services`, exclusivo do proprietário, com paginação limitada e indicação de escrita disponível; regenerar os contratos.
- **Motivação:** permitir consultar serviços ativos e inativos sem carregar todo o catálogo em memória.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/services/`, `backend/db/query/services.sql`, `backend/internal/httpapi/handler.go`, artefatos gerados, novos testes em `backend/integration/`, `docs/product.md`, `docs/architecture.md` e checklist, itens 10 e 22.
- **Base documental:** `docs/architecture.md`, “Multi-tenancy e página pública” e padrão de paginação de “Proprietário e profissional”; `docs/product.md`, “Planos, assinatura e trial”.
- **Requisitos de entrega:**
  - Listagem filtra pelo tenant do `Principal`, inclui ativos e inativos e ordena por criação decrescente, com desempate pelo ID.
  - `limit` tem padrão 20 e intervalo de 1 a 100; `offset` tem padrão 0 e intervalo de 0 a 2.147.483.647; a query lê no máximo `limit + 1` registros.
  - Resposta contém `items`, `limit`, `offset`, `has_more` e `can_write`; cada item expõe ID, nome, preço em centavos, duração e estado ativo. Lista vazia é `[]`.
  - `can_write` reutiliza o componente central de direitos; trial ausente ou expirado retorna `false` e preserva a leitura. Falhas de infraestrutura não são apresentadas como expiração.
  - Sessão ausente e papel profissional são recusados; contrato, documentação e checklist distinguem listagem da API e interface pendente.
- **Verificação:** integração HTTP cobre vazio, várias páginas, empate na ordenação, itens inativos, paginação inválida, dois tenants, recusa por papel e leitura com trial expirado; unitários cobrem a tradução do direito de escrita e falhas.
- **Dependências:** subtarefa 3.

### 5. Consultar um serviço para edição

- **O que:** definir e implementar `GET /api/v1/services/{service_id}` e regenerar os contratos.
- **Motivação:** carregar o estado persistido do serviço selecionado, inclusive para reconciliação após falha de atualização.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/services/`, `backend/db/query/services.sql`, `backend/internal/httpapi/handler.go`, artefatos gerados, novos testes em `backend/integration/`, `docs/product.md`, `docs/architecture.md` e checklist, itens 10 e 22.
- **Base documental:** checklist, item 10, “consultar”; `AGENTS.md`, isolamento por estabelecimento; `docs/product.md`, leitura após expiração.
- **Requisitos de entrega:**
  - Proprietário consulta serviço ativo ou inativo do próprio tenant e recebe os mesmos campos de registro da listagem.
  - UUID malformado produz `400`; ID inexistente ou pertencente a outro tenant produz a mesma resposta `404`, sem revelar a existência externa.
  - Leitura permanece disponível sem trial ativo; ausência de sessão e papel indevido mantêm as recusas documentadas.
  - Contrato e documentos descrevem a consulta entregue, sem declarar a edição como concluída.
- **Verificação:** integração HTTP cobre ativo, inativo, inexistente, UUID inválido, recurso de outro tenant, sessão ausente, profissional e trial expirado.
- **Dependências:** subtarefa 3.

### 6. Editar os dados de um serviço pela API

- **O que:** definir e implementar `PATCH /api/v1/services/{service_id}` para nome, preço e duração, incluindo proteção CSRF do caminho parametrizado; regenerar os contratos.
- **Motivação:** atualizar o catálogo sem substituir identidade ou alterar registros operacionais históricos.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/services/`, `backend/db/query/services.sql`, `backend/internal/auth/middleware.go` e seus testes, `backend/internal/httpapi/handler.go`, artefatos gerados, novos testes em `backend/integration/`, `docs/product.md`, `docs/architecture.md` e checklist, item 10.
- **Base documental:** `docs/product.md`, “Agenda e disponibilidade”, alterações do catálogo afetam apenas novos agendamentos; `docs/architecture.md`, autenticação, CSRF e snapshots; decisões confirmadas sobre os campos.
- **Requisitos de entrega:**
  - Apenas o proprietário com origem, CSRF e direito de escrita válidos edita um serviço do próprio tenant, inclusive inativo; resposta `200` devolve o registro atualizado.
  - Campos omitidos permanecem inalterados; `duration_minutes: null` aplica 30 minutos, correspondendo ao campo vazio; `null` em nome ou preço é inválido e corpo sem campos editáveis é recusado.
  - Atualização inválida não persiste nenhum dos campos recebidos; nome repetido é aceito. ID e estabelecimento são imutáveis e `active` é administrado pela operação específica.
  - Uma atualização SQL atômica altera somente os campos enviados; não sobrescreve mudanças concorrentes em outros campos nem modifica `created_at` ou registros históricos.
  - Recurso externo ou inexistente recebe `404`; escrita sem direito recebe `read_only`. O middleware exige CSRF também quando o caminho contém um UUID real.
  - Documentação registra as regras do PATCH e mantém pendente a proteção de snapshots nos futuros agendamentos, que ainda não existem.
- **Verificação:** unitários cobrem omissão, `null` e limites; integração HTTP cobre atualização parcial, ausência de gravação em erro, edição de inativo, tenant externo, trial expirado, origem e CSRF ausentes ou inválidos no caminho parametrizado.
- **Dependências:** subtarefas 2 e 5.

### 7. Ativar e desativar preservando o serviço

- **O que:** definir e implementar `PUT /api/v1/services/{service_id}/status` com estado desejado explícito, regenerando os contratos e protegendo a rota com CSRF.
- **Motivação:** administrar a oferta sem exclusão física ou perda de identidade e futuras referências históricas.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/services/`, `backend/db/query/services.sql`, `backend/internal/auth/middleware.go` e seus testes, `backend/internal/httpapi/handler.go`, artefatos gerados, novos testes em `backend/integration/`, `docs/product.md`, `docs/architecture.md` e checklist, item 10.
- **Base documental:** `docs/product.md`, “Serviços e profissionais inativos”; decisão confirmada de entregar ativação e desativação agora e integrar contagem e alertas quando existir agenda. Registrar essa divisão nos documentos, preservando a regra final do MVP.
- **Requisitos de entrega:**
  - Proprietário com origem, CSRF e direito de escrita válidos envia `{ active: boolean }`; resposta `200` devolve o registro com o estado persistido.
  - Repetir o mesmo estado é idempotente; ativar ou desativar preserva ID, tenant, nome, preço, duração e data de criação, sem sobrescrever edições concorrentes desses campos.
  - Recurso externo e inexistente retornam `404`; escrita após expiração retorna `read_only`; rotas parametrizadas exigem CSRF válido.
  - Não há endpoint de exclusão física. Reativação de serviço preserva sua identidade e não envolve credenciais de usuários.
  - Produto e checklist registram que contagem prévia, alertas e bloqueio de novos agendamentos com serviço inativo serão integrados à agenda; esses itens permanecem desmarcados. Nenhuma contagem fictícia é exposta.
- **Verificação:** integração HTTP cobre desativação, consulta posterior, reativação, repetição do mesmo estado, preservação dos demais campos, isolamento, papel, origem, CSRF e trial expirado.
- **Dependências:** subtarefa 5.

### 8. Consultar serviços pelo painel

- **O que:** criar `/painel/servicos` com guarda de proprietário, casca existente, navegação e listagem paginada pelo cliente tipado.
- **Motivação:** disponibilizar o catálogo no painel com os padrões visuais e de sessão já entregues.
- **Onde:** nova rota em `frontend/app/(session)/painel/servicos/`, novos componentes em `frontend/components/services/`, novo `frontend/lib/services/services-api.ts`, `frontend/components/session/owner-panel-shell/`, testes Vitest co-localizados, `docs/product.md`, `docs/architecture.md`, `docs/operations.md`, `README.md` e checklist, itens 4 e 10.
- **Base documental:** `docs/architecture.md`, “Frontend”, “Renderização e API” e “Estado”; `AGENTS.md`, design system; checklist, item 10, listagem do catálogo.
- **Requisitos de entrega:**
  - Página reutiliza `SessionGuard`, `OwnerPanelShell`, `OwnerPanelPage`, `RecordList`, `Button` e `Feedback`; navegação inclui “Serviços” com `aria-current` correto, acessível por teclado e em telas estreitas.
  - Lista mostra nome, preço em BRL, duração em minutos e estado “Ativo” ou “Inativo”, com paginação, carregamento, vazio, falha e nova tentativa.
  - Preços são formatados a partir de centavos exatos, sem cálculos monetários em ponto flutuante; consultas usam `apiFetch` e tipos gerados.
  - Troca de identidade ou página cancela consultas obsoletas; respostas antigas não substituem dados recentes; `401` remove conteúdo protegido e erros temporários não declaram logout.
  - `can_write: false` mantém a consulta e apresenta o modo somente leitura; falha de consulta não presume direito de escrita. A entrada `/painel` mantém seus destinos atuais.
  - Documentos e checklist descrevem somente a leitura entregue; validação manual inclui o novo link, isolamento por papel, responsividade e leitura após expiração.
- **Verificação:** Vitest cobre dados, vazio, paginação, falha com nova tentativa, `401`, mudança de identidade, resposta obsoleta, guarda de proprietário e indicação de página ativa; conferir teclado e temas claro e escuro manualmente.
- **Dependências:** subtarefa 4.

### 9. Cadastrar serviço pelo painel

- **O que:** incorporar formulário de cadastro à página, usando os componentes existentes e conversão exata entre preço digitado e centavos.
- **Motivação:** concluir o cadastro pelo proprietário sem exigir conhecimento da representação técnica da API.
- **Onde:** novos componentes em `frontend/components/services/`, modelo puro do formulário em `frontend/lib/services/`, `frontend/lib/services/services-api.ts`, `frontend/app/design-system/`, testes Vitest co-localizados, `docs/product.md`, `docs/architecture.md`, `docs/operations.md` e checklist, item 10.
- **Base documental:** decisões confirmadas de nome, preço e duração; `docs/product.md`, “Localização” e “Agenda e disponibilidade”; `AGENTS.md`, design system e validação autoritativa no backend.
- **Requisitos de entrega:**
  - Formulário coleta nome, preço e duração, apresenta os limites confirmados e explica o padrão de 30 minutos quando a duração estiver vazia; preço vazio não significa gratuito.
  - Preço usa composição de `Field` e `AffixedInput` com prefixo `R$` e teclado decimal; a referência do design system documenta essa composição e seus estados. Demais campos e ações reutilizam as primitivas existentes.
  - A conversão de texto decimal em centavos usa partes inteiras e no máximo duas casas, sem `parseFloat`, multiplicação de frações binárias ou arredondamento; entradas inválidas permanecem disponíveis para correção.
  - Envio usa CSRF e impede submissões simultâneas; erros de campo recebem mensagens locais em português e foco no primeiro campo inválido, sem apresentar texto bruto do servidor.
  - Após `201`, a lista é atualizada. Falha ao recarregar a lista não transforma criação confirmada em falha de cadastro; resposta ambígua não dispara reenvio automático nem anuncia sucesso sem confirmação.
  - Modo somente leitura desabilita cadastro; `read_only` recebido durante envio atualiza a explicação. Documentos e checklist registram o cadastro completo somente nesta entrega.
- **Verificação:** Vitest cobre conversão de R$ 0,00, R$ 0,01 e limite máximo, excesso de casas, obrigatoriedade, duração vazia, erros de campo, envio único, sucesso com atualização da lista, falha de recarga e expiração; conferir composição do campo no design system.
- **Dependências:** subtarefas 3 e 8.

### 10. Editar o serviço selecionado no painel

- **O que:** oferecer edição na própria página, carregando o registro pelo endpoint individual e reutilizando o modelo e os campos do cadastro.
- **Motivação:** permitir corrigir dados e atualizar o catálogo sem criar outro serviço.
- **Onde:** `frontend/components/services/`, `frontend/lib/services/`, testes Vitest co-localizados, `docs/product.md`, `docs/architecture.md`, `docs/operations.md` e checklist, item 10.
- **Base documental:** checklist, item 10, consulta e edição; `docs/product.md`, alterações de catálogo preservam os dados já registrados; `docs/architecture.md`, estado e respostas obsoletas.
- **Requisitos de entrega:**
  - Ação “Editar” consulta o serviço selecionado e oferece carregamento, erro, nova tentativa e cancelamento; troca de seleção descarta respostas antigas.
  - Formulário apresenta preço persistido em BRL, permite editar serviços inativos e envia somente campos alterados; limpar duração envia `null` para aplicar 30 minutos.
  - Cancelar não grava; `200` atualiza formulário e lista com os dados confirmados; falha de validação preserva a edição e destaca os campos afetados.
  - Falha ambígua de atualização oferece recarregar o registro antes de novo envio, sem sobrescrever silenciosamente o rascunho; `404`, `401` e `read_only` têm estados apropriados.
  - Edição fica indisponível no modo somente leitura; documentação e checklist refletem consulta e edição completas.
- **Verificação:** Vitest cobre carregamento do selecionado, edição parcial, duração limpa, cancelamento, troca de seleção com resposta obsoleta, erro de campo, registro ausente, falha ambígua, sucesso e expiração.
- **Dependências:** subtarefas 5, 6 e 9.

### 11. Ativar e desativar pelo painel

- **O que:** adicionar ações de ativação e desativação aos registros, com confirmação na própria página usando os componentes existentes.
- **Motivação:** completar a gestão de estado do catálogo com informação clara sobre preservação do serviço.
- **Onde:** `frontend/components/services/`, `frontend/lib/services/services-api.ts`, testes Vitest co-localizados, `docs/product.md`, `docs/architecture.md`, `docs/operations.md`, `README.md` e checklist, item 10.
- **Base documental:** `docs/product.md`, “Serviços e profissionais inativos”; decisão confirmada de integrar contagem e alertas apenas quando existir agenda; `AGENTS.md`, preservação histórica e design system.
- **Requisitos de entrega:**
  - Serviço ativo oferece “Desativar” e inativo oferece “Ativar”; confirmação identifica o serviço e permite cancelar antes do envio.
  - Envio informa o estado desejado explicitamente, usa CSRF e bloqueia ações simultâneas no registro; sucesso atualiza o estado persistido na lista e no editor aberto.
  - Falha ambígua permite consultar o registro antes de repetir a ação; não usa inversão cega do estado nem apresenta sucesso antes da resposta confirmada.
  - Modo somente leitura bloqueia as ações, preservando a consulta; falhas de papel, sessão ou recurso inexistente recebem tratamento consistente com a página.
  - Documentos e checklist marcam ativação/desativação do catálogo como entregues e mantêm pendentes contagem, alertas, aplicação aos agendamentos e proteção de seus snapshots. Associação a profissionais e comissões continuam nos respectivos escopos futuros.
- **Verificação:** Vitest cobre confirmação, cancelamento, ativação, desativação, envio único, falha ambígua e expiração; validação manual percorre cadastro, edição, desativação e reativação com teclado, em desktop e tela estreita.
- **Dependências:** subtarefas 7 e 10.

## Verificação conjunta

- Executar `make check` e `make build`; `make check` já inclui `make generate-check`, `gofmt`, `go vet -tags=integration ./...`, testes unitários, integração HTTP do backend com Testcontainers e PostgreSQL real, ESLint, Prettier e checagem de tipos. Não adicionar E2E ou integração do frontend.
- Revisar o diff e os fluxos completos comparando OpenAPI, código gerado, migration, testes, design system e documentos. Confirmar isolamento, papel, proteção CSRF em caminhos parametrizados, preço exato, leitura após expiração e preservação do ID ao editar ou desativar.
- Conferir a divisão dos checkboxes mistos: catálogo entregue é distinto de associação profissional-serviço, agenda, snapshots de agendamentos, contagem prévia e alertas futuros. A documentação deve descrever cada etapa efetivamente implementada, sem apresentar essas dependências como disponíveis.
