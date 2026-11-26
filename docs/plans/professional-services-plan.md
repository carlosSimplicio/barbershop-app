# Associação de serviços aos profissionais

Permitir ao proprietário consultar e configurar quais serviços cada profissional oferece, com vínculos explícitos, isolamento por estabelecimento e preservação dos registros.

Decisões confirmadas pelo usuário: gestão em modal própria, aberta por “Serviços” dentro da edição do profissional; configuração permitida para profissionais e serviços ativos ou inativos; cada ação “Associar” ou “Remover” salva imediatamente um único vínculo. Inatividade continua impedindo a elegibilidade para novos agendamentos. Essas decisões complementam a documentação e devem entrar nela nas entregas correspondentes. Comissões, jornada, disponibilidade, agendamento público e alertas de impacto futuro permanecem fora deste plano.

## Progresso

- [x] 1. Persistência e integridade dos vínculos.
- [x] 2. Consulta paginada pela API.
- [x] 3. Mutação idempotente pela API.
- [ ] 4. Consulta na modal do profissional.
- [ ] 5. Associação e remoção pelo painel.

### 1. Persistir vínculos sem cruzar estabelecimentos

- **O que:** criar a estrutura relacional das associações, com estado ativo e identidade estável do par profissional-serviço.
- **Motivação:** representar a oferta explícita sem conceder automaticamente todos os serviços a todos os profissionais nem destruir referências ao remover um vínculo.
- **Onde:** nova migration prevista `backend/db/migrations/00010_professional_services.sql`, `backend/db/generated/`, `backend/integration/`, `docs/architecture.md` em “Banco de dados” e `docs/implementation-scope.md`, item 3.
- **Base documental:** `docs/product.md`, “Agenda e disponibilidade” e “Serviços e profissionais inativos”; `docs/architecture.md`, “Banco de dados”; `AGENTS.md`, integridade estrutural e migrations novas.
- **Requisitos de entrega:**
  - A tabela `professional_services` contém `establishment_id`, `professional_id`, `service_id`, estado `active` e datas; há no máximo um registro por par no estabelecimento.
  - FKs compostas para `professionals (user_id, establishment_id)` e `services (id, establishment_id)` impedem vínculos entre tenants mesmo por SQL direto. A migration acrescenta a chave única necessária ao perfil profissional, sem reescrever migrations anteriores.
  - Não existem preço, duração ou configuração de comissão na associação; preço e duração continuam pertencendo a `services`.
  - Remoção lógica preserva a identidade do par; referências usam `ON DELETE RESTRICT`, sem cascatas destrutivas.
  - A migration não cria associações para o catálogo existente, não muda estados dos cadastros e não preenche o marcador de início dos dados operacionais.
  - Artefatos afetados pelo schema são regenerados com `sqlc`, sem edição manual.
  - Arquitetura descreve o schema entregue; o checkbox de persistência de associações permanece pendente até a integração das queries na próxima etapa.
- **Verificação:** integração com PostgreSQL real aplica as migrations e protege unicidade, referências inexistentes, rejeição de cada lado de uma associação cruzada e ausência de associações automáticas. Executar as verificações de backend aplicáveis.

### 2. Consultar o catálogo com os vínculos do profissional

- **O que:** entregar `GET /api/v1/professionals/{professional_id}/services` como consulta paginada dos serviços do estabelecimento, indicando quais estão associados ao profissional.
- **Motivação:** apresentar opções e vínculos em uma única leitura limitada, incluindo inativos, sem carregar todo o catálogo no navegador ou consultar cada serviço separadamente.
- **Onde:** `backend/openapi/openapi.yaml`; novo slice previsto em `backend/internal/professionals/offerings/`, com caso de uso, handler e store próprios; novo arquivo `backend/db/query/professional_services.sql`; `backend/internal/httpapi/handler.go`; composição em `backend/cmd/api/main.go` e `backend/integration/suite_test.go`; artefatos gerados; documentos de produto, arquitetura e escopo.
- **Base documental:** `docs/product.md`, permissões, oferta explícita e leitura após expiração; `docs/architecture.md`, “API contract-first”, “Backend” e “Banco de dados”; decisão confirmada sobre consulta de ativos e inativos.
- **Requisitos de entrega:**
  - Somente proprietário consulta profissional do próprio estabelecimento. Profissional inexistente, de outro tenant ou ID de proprietário recebe o mesmo `404`; UUID inválido recebe `400`.
  - A resposta inclui identificação mínima do profissional (`id`, nome e estado), `items`, `limit`, `offset`, `has_more` e `can_write`. Cada item apresenta os dados do serviço já expostos pelo catálogo e `associated` booleano, distinto do estado ativo do serviço.
  - Serviços ativos e inativos aparecem, associados ou não; ausência de registro ou vínculo inativo produz `associated: false`. Serviços e profissionais recém-criados não adquirem vínculos automaticamente.
  - Paginação e ordenação seguem o catálogo existente: padrão 20, limite de 1 a 100, offset de 0 a 2.147.483.647 e criação/ID decrescentes. A query busca somente `limit + 1`, com colunas enumeradas e filtros explícitos de tenant.
  - Trial ausente ou expirado preserva consulta e retorna `can_write: false`; falha na resolução dos direitos continua sendo erro, sem converter indisponibilidade em leitura autorizada para escrita.
  - O slice não amplia o `Store` legado de credenciais e gestão de profissionais com responsabilidades de associação. Stores, casos de uso e handlers novos são compostos explicitamente em produção e na integração.
  - OpenAPI e tipos Go/TypeScript são regenerados juntos. Documentos distinguem consulta entregue de alterações ainda pendentes; o item 3 do escopo marca a persistência integrada sem antecipar a conclusão da interface.
- **Verificação:** unitários cobrem autorização, paginação e resolução dos direitos; integração HTTP cobre catálogo vazio, vínculos ativos/inativos, paginação, recursos ausentes, isolamento, recusa de profissional e leitura após expiração. Executar verificações de backend e checagens do frontend afetadas pela geração.
- **Dependências:** subtarefa 1.

### 3. Definir um vínculo de forma idempotente pela API

- **O que:** entregar `PUT /api/v1/professionals/{professional_id}/services/{service_id}`, com corpo `{ "associated": true | false }`, alterando apenas o par solicitado.
- **Motivação:** associar e remover com repetição segura, sem substituir a coleção inteira nem sobrescrever vínculos alterados em outra operação.
- **Onde:** OpenAPI; `backend/internal/professionals/offerings/`; `backend/db/query/professional_services.sql`; `backend/internal/httpapi/handler.go`; `backend/internal/auth/middleware.go` e testes; `backend/integration/`; artefatos gerados; documentos de produto, arquitetura e escopo.
- **Base documental:** `docs/product.md`, oferta explícita e preservação ao remover associação; `docs/architecture.md`, autenticação, SQL explícito e ciclo das mutações; decisões confirmadas de configuração de inativos e gravação por vínculo.
- **Requisitos de entrega:**
  - Somente proprietário com direito centralizado de escrita, origem confiável e CSRF válido altera pares do próprio estabelecimento. O matcher CSRF protege o caminho real com os dois IDs; acesso expirado recusa a escrita sem mudar o vínculo.
  - Profissional e serviço são validados no tenant, inclusive ao remover. IDs inexistentes ou externos não revelam dados de outro estabelecimento; UUIDs inválidos recebem `400` e alvos indisponíveis recebem `404`.
  - `associated` é booleano obrigatório; ausência, `null`, tipos incorretos, campos desconhecidos e JSON inválido são recusados. O corpo segue o limite existente de 8 KiB das mutações de profissionais.
  - Associação e remoção são permitidas mesmo com profissional ou serviço inativo, conforme decisão confirmada. Isso não ativa os cadastros nem concede elegibilidade enquanto profissional, serviço e vínculo não estiverem ativos.
  - Repetir o mesmo estado retorna `200` sem duplicar registros ou alterar datas de uma transição que não ocorreu. Remover um vínculo ainda ausente retorna desassociado; reassociar reutiliza o mesmo par preservado.
  - A escrita é atômica e usa unicidade/upsert ou transação conforme necessário. Criações concorrentes não duplicam pares; alterações de pares diferentes não se sobrescrevem. Disputas no mesmo par preservam um único estado persistido.
  - A resposta `200` retorna o item confirmado no mesmo formato usado pela consulta, com `associated` refletindo a operação. Nome do profissional, credenciais, estado dos cadastros, preço e duração não são alterados.
  - Desativar ou reativar profissional/serviço preserva os vínculos; remover associação não apaga cadastros nem cria efeitos sobre futuros históricos de agenda e financeiro. Não implementar esses domínios antecipadamente.
  - Produto e arquitetura registram configuração de inativos, idempotência e contrato entregue. O checklist distingue API pronta de painel pendente, mantendo comissões e validação de disponibilidade nos respectivos escopos futuros.
- **Verificação:** unitários cobrem recusas de domínio e validação HTTP; integração HTTP cobre associar/remover/reassociar, repetição, remoção de vínculo ausente, inativos, isolamento, papel, origem, CSRF e `read_only`, além de escritas concorrentes no mesmo par e em pares diferentes. Verificar preservação de preço, duração, credenciais e vínculos após mudanças de estado dos cadastros. Executar as checagens de backend e geração.
- **Dependências:** subtarefa 2.

### 4. Consultar serviços em uma modal própria do profissional

- **O que:** adicionar a ação “Serviços” dentro da edição do profissional e abrir uma modal secundária com o catálogo paginado e a situação dos vínculos, inicialmente somente para consulta.
- **Motivação:** oferecer um lugar próprio para a gestão sem misturar associações com o salvamento do nome ou perder o rascunho do editor.
- **Onde:** `frontend/components/professionals/professionals-workspace/`, `edit-professional-panel/`, novo componente previsto `professional-services-panel/`; novo cliente tipado em `frontend/lib/professionals/professional-services-api.ts` e módulo de interpretação das respostas no mesmo domínio; `frontend/components/ui` e `frontend/app/design-system` somente se houver necessidade de nova variante; documentos de produto, arquitetura e operação.
- **Base documental:** decisão confirmada da modal própria aberta na edição; `docs/architecture.md`, “Frontend”, “Estado” e composição de modais; `AGENTS.md`, design system.
- **Requisitos de entrega:**
  - “Serviços” fica disponível após a consulta do profissional, inclusive para inativo e em modo somente leitura; abre a modal identificando o profissional e consulta somente a página solicitada.
  - A lista distingue serviço ativo/inativo e associado/não associado, exibe preço BRL e duração do catálogo e explica que vínculo com cadastro inativo não habilita novos agendamentos.
  - Reutilizar `Modal`, `ModalActions`, `RecordListPanel`, `RecordSummary`, `Button` e `Feedback`; oferecer carregamento, vazio, erro com nova tentativa, atualização, paginação e ação “Fechar”. Não adicionar dependências nem carregar antecipadamente todas as páginas.
  - Ajustar o controle atual `confirming` do workspace para distinguir a gestão de serviços das confirmações de estado. As modais secundárias são mutuamente exclusivas; edição e rascunho de nome permanecem montados, sem permitir ações conflitantes no editor sob a modal aberta.
  - Fechar ou pressionar Escape atua somente na modal superior e retorna o foco a “Serviços”; fechar a edição mantém o retorno ao lápis ou à lista. Não criar formulários aninhados.
  - Cliente usa `apiFetch` e tipos gerados; aborta consultas ao desmontar e ignora respostas obsoletas por página, profissional ou identidade. `401` remove conteúdo protegido e respostas atrasadas não o restauram; erros temporários permitem nova tentativa sem declarar logout.
  - Documentos registram a consulta entregue e o roteiro manual; o checkbox de associação no painel permanece pendente até as mutações da próxima etapa.
- **Verificação:** Vitest cobre abertura, carregamento paginado, vazio, falha e nova tentativa, estados dos itens, cancelamento de consultas, respostas obsoletas, sessão ausente e preservação do rascunho do nome. Regressões existentes de edição/desativação/reativação continuam passando. Conferir no navegador teclado, modais sobrepostas, foco, temas e tela estreita; executar testes, ESLint, Prettier e tipos.
- **Dependências:** subtarefa 2.

### 5. Associar e remover pelo painel com confirmação do resultado

- **O que:** acrescentar ações imediatas “Associar” e “Remover” por serviço na modal, usando o endpoint idempotente sem etapa adicional de confirmação.
- **Motivação:** concluir a configuração dos serviços oferecidos com feedback de cada operação e sem gravar o rascunho do nome.
- **Onde:** `frontend/components/professionals/professional-services-panel/`, workspace e testes relacionados; `frontend/lib/professionals/professional-services-api.ts` e interpretação de respostas; `docs/product.md`, `docs/architecture.md`, `docs/operations.md` e `docs/implementation-scope.md`, itens 3, 9 e 10.
- **Base documental:** decisões confirmadas de salvar imediatamente cada vínculo e configurar inativos; `docs/architecture.md`, “Ciclo das mutações”; `docs/product.md`, preço/duração compartilhados e preservação dos vínculos.
- **Requisitos de entrega:**
  - Cada ação envia o estado desejado para um único serviço; não usa toggle no backend, não substitui a coleção e não modifica nome, preço, duração ou estado dos cadastros. Inativos continuam configuráveis conforme a decisão confirmada.
  - Escrita depende de direitos confirmados no workspace e na consulta da modal. `can_write: false` bloqueia os botões de mutação, preservando leitura; uma recusa `read_only` propaga o bloqueio aos demais formulários da página.
  - Reutilizar `useMutationRequest` para impedir envio duplicado, cancelar ao desmontar e liberar atividade. Enquanto há envio, bloquear ações concorrentes da modal, paginação, atualização e fechamento; os controles são liberados ao terminar, inclusive em falha.
  - Somente resposta válida `200`, correspondente ao serviço, ao estado solicitado e à seleção ainda vigente, confirma sucesso e atualiza diretamente o item da página; não há GET adicional obrigatório nem atualização otimista.
  - Rede, `5xx` e resposta fora do contrato exibem mensagem local segura e permitem fechar ou tentar novamente explicitamente, sem repetição automática, consulta de confirmação ou afirmação de rollback. Repetir manualmente mantém o mesmo estado desejado da tentativa que falhou.
  - Recurso ausente, sessão ausente e acesso recusado têm tratamento local por códigos/status, sem mensagens brutas. Sucesso e falha são anunciados de forma acessível; alterações anteriores confirmadas permanecem salvas ao fechar.
  - A lista principal e o editor do profissional preservam seus dados e rascunho; somente o vínculo confirmado é atualizado. A associação não preenche o marcador operacional nem altera direitos do estabelecimento.
  - Documentos descrevem o fluxo completo e seu roteiro manual. O checklist conclui a associação explícita e preço/duração compartilhados, mantendo pendentes comissões específicas, agenda, disponibilidade, snapshots e alertas; não marcar como entregues integrações históricas que ainda não existem.
- **Verificação:** Vitest cobre associar, remover, repetição manual após erro com o mesmo estado desejado, envio único, bloqueios durante envio, inativos, `read_only` compartilhado, `401`, recurso ausente e preservação do nome editado. Verificação manual percorre associação/remoção, reabertura, dois profissionais com conjuntos distintos, expiração e desativação/reativação dos cadastros. Executar todas as checagens de frontend.
- **Dependências:** subtarefas 3 e 4.

## Verificação conjunta

- Executar `make check` e `make build`; incluem geração verificável, formato Go, `go vet -tags=integration ./...`, unitários, integração do backend com Testcontainers/PostgreSQL real, Prettier, ESLint, tipos e builds. Registrar qualquer impedimento de execução. Não introduzir E2E ou integração de frontend.
- Revisar o diff e os fluxos de catálogo, edição de nome, desativação/reativação e associações, conferindo isolamento, CSRF do caminho com dois IDs, concorrência, direitos de escrita e preservação dos registros.
- Comparar OpenAPI, artefatos gerados, migrations, código, testes, documentos e checklists; manter explícito que a elegibilidade de agenda será aplicada quando os fluxos de disponibilidade e agendamento forem entregues.
