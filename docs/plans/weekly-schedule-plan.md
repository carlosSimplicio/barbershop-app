# Jornada semanal dos profissionais

Permitir que o proprietário configure a jornada de qualquer profissional do próprio estabelecimento e que cada profissional configure a sua em “Minha jornada”. A jornada semanal é a base para as próximas etapas de bloqueios pontuais, disponibilidade e agendamento público.

### 1. Persistir períodos semanais por profissional

- **O que** — Criar a tabela `professional_work_periods` e a consulta dos períodos semanais.
- **Motivação** — Guardar horários locais recorrentes e garantir que cada período pertença ao mesmo estabelecimento que o profissional.
- **Onde** — Nova migration em `backend/db/migrations/`, novo arquivo em `backend/db/query/`, `backend/db/generated/`, `backend/integration/` e documentação de banco e persistência.
- **Base documental** — `docs/product.md`, “Localização” e “Agenda e disponibilidade”. `docs/architecture.md`, “Banco de dados”. A jornada inicialmente vazia foi confirmada no planejamento.
- **Requisitos de entrega**
  - FK composta para `professionals (user_id, establishment_id)` recusa períodos entre estabelecimentos. A migration preserva os cadastros e vínculos existentes.
  - Cada período guarda dia da semana e minutos locais de início e fim. A consulta filtra por estabelecimento e profissional, ordena por dia e início e usa colunas explícitas.
  - Profissionais novos e existentes começam com jornada vazia. A jornada recorrente mantém a alteração do fuso sujeita às regras vigentes de início de dados operacionais.
  - A arquitetura registra a persistência entregue. O item 3 de `docs/implementation-scope.md` distingue jornada persistida das demais persistências de agenda.
- **Verificação** — Testes de integração com migrations reais para jornada vazia em cadastro preexistente e recusa de FK cruzada. Regenerar as queries com sqlc.

### 2. Consultar a jornada pela API

- **O que** — Entregar `GET /professionals/{professional_id}/weekly-schedule` em uma fatia própria de jornada.
- **Motivação** — Consultar a jornada e os direitos de edição a partir da sessão, inclusive com o estabelecimento em modo somente leitura.
- **Onde** — `backend/openapi/openapi.yaml`, novo slice previsto em `backend/internal/professionals/schedule/`, `backend/internal/httpapi/handler.go`, composições em `backend/cmd/api/main.go` e `backend/integration/suite_test.go`, artefatos gerados e `docs/architecture.md`.
- **Base documental** — `docs/product.md`, “Cadastro, usuários e permissões” e “Planos, assinatura e trial”. Foi confirmado que o proprietário pode consultar e editar a jornada de profissionais inativos.
- **Requisitos de entrega**
  - O proprietário pode consultar ativos e inativos do próprio estabelecimento. O profissional pode consultar a própria jornada, após a troca obrigatória de senha quando aplicável.
  - A API deriva o estabelecimento da sessão. IDs válidos fora do escopo do usuário e IDs sem perfil profissional recebem `404`. UUID inválido recebe `400`.
  - A resposta traz o ID do profissional, fuso IANA do estabelecimento, `can_write` e `periods` em ordem canônica ou `[]`. Cada período usa `weekday` de 1 (segunda-feira) a 7 (domingo), `start_time` e `end_time` em `HH:mm`.
  - O direito central de escrita produz `can_write: false` em modo somente leitura. Uma falha do serviço de direitos permanece um erro da consulta.
- **Verificação** — Integração HTTP com Testcontainers para proprietário e profissional, alvos externos e ausentes, troca obrigatória de senha e trial expirado. Regenerar OpenAPI para Go e TypeScript.
- **Dependências** — 1.

### 3. Salvar a semana inteira pela API

- **O que** — Entregar `PUT /professionals/{professional_id}/weekly-schedule` com validação da semana e substituição transacional.
- **Motivação** — Manter a jornada anterior quando uma gravação falhar e garantir que escritas concorrentes produzam uma semana completa.
- **Onde** — `backend/openapi/openapi.yaml`, slice de jornada, queries de jornada e de profissionais, `backend/internal/auth/middleware.go`, composições da API e da suíte de integração, artefatos gerados, `docs/product.md` e `docs/architecture.md`.
- **Base documental** — `docs/product.md`, “Localização”, “Agenda e disponibilidade” e “Planos, assinatura e trial”. Foram confirmados o uso de minutos, a limpeza da semana, a divisão à meia-noite e o salvamento integral com prevalência da última gravação aceita.
- **Requisitos de entrega**
  - O `PUT` exige origem confiável, CSRF, escopo equivalente ao da consulta e direito central de escrita. O proprietário pode salvar a jornada de profissionais inativos.
  - Na autoedição, a transação confirma sob bloqueio que o profissional segue ativo e que a versão da credencial no `Principal` autenticado corresponde à versão atual. Uma requisição autenticada antes da desativação continua recusada após a reativação e preserva a jornada gravada.
  - O corpo exige `periods` como array e usa `[]` para limpar a semana. Campos desconhecidos, valores nulos em campos obrigatórios e formatos inválidos recebem erro com código estável.
  - Cada período tem início anterior ao fim, precisão de minuto e pertence a um único dia. `24:00` é aceito apenas como fim. Períodos adjacentes são válidos e qualquer sobreposição no mesmo dia recusa a semana inteira. Para trabalhar após a meia-noite, cadastre um período até `24:00` e outro a partir de `00:00` no dia seguinte.
  - O `PUT` tem um limite de corpo próprio no domínio de jornada. O limite comporta a semana de 10.080 períodos adjacentes de um minuto, que ocupa 544.333 bytes no JSON compacto do contrato. Documentar o teto aplicado e a resposta a excesso no OpenAPI.
  - A escrita serializa substituições para o mesmo profissional usando o bloqueio do usuário filtrado por tenant. Leitores observam uma semana completa. A última gravação aceita prevalece e repetir a mesma semana mantém o mesmo estado.
  - O sucesso retorna `200` com a jornada confirmada e ordenada. Gravar ou limpar períodos mantém o fuso e o marcador operacional do estabelecimento.
- **Verificação**
  - Testes Go para formatos e fronteiras, períodos adjacentes, cruzamento de dia e sobreposição.
  - Integração HTTP para autorização, CSRF, recusa de escrita após expiração, limpeza, recusa de semana inválida sem escrita parcial, rollback durante a substituição e gravações concorrentes com semanas completas.
  - Aceitar e persistir a semana de 10.080 períodos. Recusar um corpo acima do teto e confirmar que a jornada gravada foi preservada.
  - Pausar um `PUT` de autoedição após a autenticação, desativar e reativar o profissional, liberar a requisição e confirmar a recusa e a preservação da semana. Usar como referência `backend/integration/professional_password_change_inflight_test.go`.
  - Verificar que salvar a jornada mantém a possibilidade de mudar o fuso quando o marcador operacional está vazio. Regenerar OpenAPI e sqlc.
- **Dependências** — 2.

### 4. Criar o editor semanal reutilizável

- **O que** — Criar o editor de jornada e conectar a consulta e o salvamento ao cliente tipado.
- **Motivação** — Aplicar as mesmas regras de edição, sessão e feedback nas duas interfaces.
- **Onde** — Novos módulos previstos em `frontend/lib/professionals/` e `frontend/components/professionals/`, cliente em `frontend/lib/api/client.ts`, `useMutationRequest` e componentes existentes de `frontend/components/ui/`. Atualizar a referência React em `frontend/app/design-system` se o editor exigir uma nova primitiva, variante ou estado visual.
- **Base documental** — `AGENTS.md`, “Regras universais”. `docs/architecture.md`, “Frontend”, “Estado” e “Ciclo das mutações”. `docs/product.md`, “Agenda e disponibilidade” e “Experiência de uso”.
- **Requisitos de entrega**
  - O editor exibe os sete dias e o fuso. Permite adicionar, editar e remover vários períodos por dia, representar o fim à meia-noite e salvar uma semana vazia. Campos têm rótulos acessíveis e as mensagens aparecem em português.
  - A interface indica carregamento, erro e somente leitura. A API permanece autoritativa, e a validação local aponta períodos inválidos ou sobrepostos antes do envio.
  - Cada salvamento envia a semana completa com CSRF. A interface impede envios duplicados.
  - Somente um `200` válido para o profissional atual, com os mesmos períodos da semana enviada após ordenar por dia e início, confirma o sucesso e atualiza a semana gravada no editor. Se os períodos divergirem, o editor apresenta erro, preserva o rascunho e permite nova tentativa explícita.
  - Falhas temporárias preservam o rascunho para nova tentativa explícita. `read_only` retira a permissão de salvar e `401` retira o conteúdo protegido. Trocas de alvo ou sessão descartam respostas obsoletas.
- **Verificação** — Vitest para adição e remoção de períodos, meia-noite, limpeza, erros de intervalo, somente leitura, tentativa manual após falha e invalidação de sessão ou de resposta obsoleta. Confirmar o sucesso quando a API devolve os períodos do envio ordenados e tratar como erro um `200` com o ID correto e outra semana, preservando o rascunho.
- **Dependências** — 3.

### 5. Abrir “Jornada” na edição do profissional

- **O que** — Integrar o editor em uma modal própria acessada pelo proprietário em `/painel/profissionais`.
- **Motivação** — Tornar a configuração acessível a partir do perfil do profissional e preservar o rascunho da edição cadastral.
- **Onde** — `frontend/components/professionals/professionals-workspace/`, `edit-professional-panel/`, infraestrutura de `Modal` e `ModalActions`, `docs/product.md` e `docs/architecture.md`.
- **Base documental** — `docs/product.md`, “Administração do catálogo”. `AGENTS.md`, “Regras universais”. A abertura pela ação “Jornada” na edição foi confirmada no planejamento.
- **Requisitos de entrega**
  - “Jornada” abre a modal para ativos e inativos. A modal preserva o rascunho cadastral, tem ações compactas à direita em `ModalActions` e usa botão secundário para cancelar.
  - Jornada, vínculos de serviços e mudança de estado ocupam a edição uma operação por vez. Um envio em curso impede fechamento ou operações concorrentes no painel. Ao fechar, o foco retorna à ação “Jornada”.
  - Com acesso expirado, o proprietário pode abrir e consultar a jornada. A escrita exige direitos confirmados na lista e no editor. Uma recusa `read_only` sincroniza o modo somente leitura com o restante do painel.
- **Verificação** — Vitest para abrir a jornada do profissional escolhido, preservar e retomar o rascunho com foco, coordenar as modais e abrir para consulta em modo somente leitura.
- **Dependências** — 4.

### 6. Entregar “Minha jornada” ao profissional

- **O que** — Disponibilizar o mesmo editor em `/painel/minha-jornada`, com link em `/testes`.
- **Motivação** — Permitir a autoedição sem acesso à administração dos demais profissionais.
- **Onde** — Nova rota em `frontend/app/(session)/painel/minha-jornada/`, `frontend/app/(session)/testes/page.tsx`, guarda e sessão em `frontend/components/session/`, `docs/product.md`, `docs/architecture.md`, `docs/implementation-scope.md` e `docs/local-validation-checklist.md`.
- **Base documental** — `docs/product.md`, “Cadastro, usuários e permissões” e “Agenda e disponibilidade”. `docs/implementation-scope.md`, item 12. A página própria acessada por `/testes` foi confirmada no planejamento.
- **Requisitos de entrega**
  - O link em `/testes` e o acesso direto a `/painel/minha-jornada` exigem uma sessão autenticada com papel `professional` e `password_change_required: false`. A guarda impede que sessões de proprietário, sessões com troca obrigatória pendente e visitantes montem o editor ou façam chamadas de jornada por essa rota.
  - O `professional_id` do `GET` e do `PUT` vem de `principal.user_id`.
  - O profissional consulta a própria semana em modo somente leitura quando o trial expira. Em tela estreita, os sete dias e os controles permanecem utilizáveis por toque e teclado.
  - O produto e a arquitetura descrevem os fluxos entregues. Marcar no item 12 a configuração por ambos os papéis e os múltiplos períodos por dia, após validar API e painel. Acrescentar o roteiro de jornada ao checklist local.
- **Verificação** — Vitest para visibilidade do link, consultas e salvamentos com `principal.user_id` e acesso direto com quatro estados de sessão: profissional elegível, profissional com troca pendente, proprietário autenticado e visitante. Nos três últimos casos, verificar a ausência do editor e de chamadas de jornada. Validação local em tela estreita com proprietário e profissional editando a mesma jornada.
- **Dependências** — 4 e 5 para concluir os checkboxes compartilhados.

## Verificação conjunta

Executar `make generate`, `make check` e `make build`. Conferir o diff, a jornada nos dois acessos, o contrato OpenAPI e os documentos afetados antes de marcar a entrega como concluída.
