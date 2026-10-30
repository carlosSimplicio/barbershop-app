# Consulta, edição, desativação e reativação de profissionais

Permitir ao proprietário administrar os profissionais existentes pelo painel, preservando identidade, isolamento do estabelecimento e segurança das credenciais e sessões.

Decisões confirmadas nesta conversa: a reativação aceita qualquer senha temporária válida, inclusive com o mesmo valor da anterior; reativar um profissional já ativo retorna conflito sem alterar sua senha. Atualizar a arquitetura para permitir explicitamente essa reutilização, mantendo a invalidação das sessões e tokens anteriores e a troca obrigatória após o login. A edição abrange o nome; o e-mail permanece imutável conforme a arquitetura vigente.

O schema atual já possui os campos necessários em `users`, `user_credentials`, `auth_sessions` e `professionals`; não há mudança de schema prevista. Associações com serviços, comissões, jornada, agenda, contagem de agendamentos afetados e alertas continuam nos respectivos escopos futuros. Não oferecer exclusão física nem redefinição independente de senha de profissional ativo nesta entrega.

### 1. Informar o direito de escrita na listagem de profissionais

- **O que:** acrescentar `can_write` a `GET /api/v1/professionals`, calculado pelo componente central de direitos; atualizar OpenAPI e regenerar Go e TypeScript.
- **Motivação:** permitir que o painel bloqueie alterações antes do envio quando o estabelecimento estiver em leitura, seguindo o padrão já usado pelo catálogo.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/professionals/service.go`, `backend/internal/professionals/handler.go`, testes co-localizados, `backend/integration/professionals_list_test.go`, artefatos gerados, `docs/architecture.md` e `docs/implementation-scope.md`, itens 6 e 9.
- **Base documental:** `docs/product.md`, “Planos, assinatura e trial”; `docs/architecture.md`, “Catálogo de serviços” e “Assinaturas da plataforma”.
- **Requisitos de entrega:**
  - A resposta mantém registros, paginação e ordenação atuais e acrescenta `can_write` booleano obrigatório.
  - Trial ausente ou expirado retorna `can_write: false` sem impedir a leitura; trial ativo retorna `true` pela decisão central, sem cálculo de datas no slice.
  - Erro de infraestrutura na resolução do direito permanece erro, sem ser apresentado como expiração.
  - Contrato, consumidores existentes, fixtures e documentação permanecem consistentes; os checkboxes distinguem o direito informado na leitura das novas escritas ainda pendentes.
- **Verificação:** integração HTTP cobre leitura com trial ativo, ausente e expirado, preservando paginação e isolamento; teste unitário protege a distinção entre recusa de escrita e erro de infraestrutura.

### 2. Consultar individualmente profissional ativo ou inativo

- **O que:** implementar `GET /api/v1/professionals/{professional_id}`, com contrato, query `sqlc`, caso de uso, handler e delegação HTTP; regenerar os artefatos afetados.
- **Motivação:** abrir um cadastro persistido para edição e permitir conferir seu estado após uma mutação de resultado incerto.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/db/query/professionals.sql`, `backend/internal/professionals/`, `backend/internal/httpapi/handler.go`, novos testes em `backend/integration/`, artefatos gerados, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`, itens 9 e 22.
- **Base documental:** `docs/implementation-scope.md`, item 9; `docs/architecture.md`, “Multi-tenancy e página pública”, “Autenticação” e “API contract-first”.
- **Requisitos de entrega:**
  - Somente proprietário consulta o profissional, exigindo ID, estabelecimento derivado do `Principal` e papel `professional` na leitura persistida.
  - `200` contém apenas os campos de `ProfessionalResponse`: ID, nome, e-mail e estado ativo, sem credenciais nem informações de sessões.
  - Ativos e inativos são consultáveis, inclusive após expiração do trial.
  - UUID inválido retorna `400`; ID inexistente, usuário sem perfil profissional e ID de outro estabelecimento produzem o mesmo `404`; sessão ausente retorna `401` e papel indevido, `403`.
  - Documentação e checklist registram a consulta individual entregue, separando-a de edição e transições de estado.
- **Verificação:** integração HTTP cobre os dois estados, leitura após expiração, UUID inválido, recurso ausente, outro tenant e recusa de papel e sessão; conferir ausência de material de credencial na resposta.

### 3. Editar o nome preservando os demais dados do profissional

- **O que:** implementar `PATCH /api/v1/professionals/{professional_id}` para nome, compartilhando a validação pura com o cadastro; incluir contrato, SQL, autorização, handler e geração.
- **Motivação:** corrigir o cadastro sem recriar o usuário nem modificar e-mail, credencial ou estado ativo.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/professionals/`, `backend/db/query/professionals.sql`, `backend/internal/auth/middleware.go`, `backend/internal/httpapi/handler.go`, testes em `backend/integration/`, artefatos gerados e documentos de produto, arquitetura e escopo.
- **Base documental:** `docs/product.md`, “Cadastro, usuários e permissões”; `docs/architecture.md`, “Campos opcionais e atualizações parciais”, “Backend” e “Autenticação”; `AGENTS.md`, validações compartilhadas e isolamento.
- **Requisitos de entrega:**
  - A entrada aceita somente `name`; nome omitido, `null`, vazio após remoção de espaços nas extremidades ou com tipo incorreto retorna `400`. Campos desconhecidos são recusados.
  - Cadastro e edição compartilham o limite existente de 120 pontos de código e a remoção de espaços nas extremidades, preservando o contrato atual do cadastro. Caractere NUL é recusado antes do PostgreSQL com erro de campo `invalid_name`, incluindo regressão no cadastro.
  - Proprietário pode editar profissional ativo ou inativo do próprio estabelecimento; escrita exige origem confiável, CSRF na rota parametrizada e `RequireWrite` satisfeito.
  - SQL altera somente nome e data de atualização do perfil, filtrando tenant e papel; ID, e-mail, estado e credenciais permanecem preservados, inclusive sob mudança concorrente de estado.
  - `200` retorna o registro persistido; recusas de sessão, papel, recurso e acesso expirado seguem os códigos já adotados. OpenAPI e documentos descrevem o contrato e os checkboxes distinguem API e interface.
- **Verificação:** testes unitários cobrem limites Unicode e recusas do nome; integração HTTP cobre edição de ativo e inativo, NUL, campos não permitidos, CSRF parametrizado, trial expirado, outro tenant e preservação de credenciais e estado.
- **Dependências:** subtarefa 2.

### 4. Impedir criação de sessão com estado ou credencial obsoletos

- **O que:** fazer a persistência do login revalidar o usuário sob bloqueio transacional antes de criar a sessão, recebendo a versão de credencial usada na verificação da senha.
- **Motivação:** hoje o login lê o usuário e cria a sessão em operações separadas; uma desativação ou troca de credencial nesse intervalo pode terminar antes da gravação de uma nova sessão pelo login antigo.
- **Onde:** `backend/internal/auth/login.go`, `backend/internal/auth/store.go`, `backend/db/query/authentication.sql`, testes unitários de autenticação, novos testes em `backend/integration/`, código `sqlc` gerado e `docs/architecture.md`, “Contrato de sessão”.
- **Base documental:** `docs/architecture.md`, “Autenticação” e “Contrato de sessão”; `docs/implementation-scope.md`, item 6, revogação imediata; `AGENTS.md`, transações e regressões.
- **Requisitos de entrega:**
  - A criação da sessão bloqueia a linha do usuário e revalida estado ativo, versão da credencial e identidade usada para emitir as claims, além da elegibilidade de confirmação do proprietário.
  - Invalidação concorrente da credencial ou desativação recusa o login com a resposta genérica de credenciais inválidas e não persiste a sessão nem envia cookies de sucesso.
  - Login, futura desativação, reativação e troca obrigatória usam uma ordem compatível de bloqueios, começando pelo usuário; nenhum fluxo mantém uma leitura obsoleta para autorizar a escrita.
  - Login comum de proprietário confirmado e de profissional com senha temporária mantém prazo, cookies e restrições existentes; falha de infraestrutura continua distinta de credencial inválida.
  - A arquitetura descreve a revalidação atômica efetivamente entregue.
- **Verificação:** teste unitário protege a tradução da recusa de persistência; regressão de integração reproduz por HTTP a mudança de estado ou versão entre verificação e gravação, com coordenação determinística e sem substituir autenticação ou persistência reais. Confirmar ausência de sessão criada pelo login recusado e preservar os fluxos existentes de login.

### 5. Desativar profissional e revogar seu acesso atomicamente

- **O que:** implementar `POST /api/v1/professionals/{professional_id}/deactivation`, sem corpo, com bloqueio do usuário, alteração de estado, incremento de versão e revogação de sessões na mesma transação.
- **Motivação:** retirar o acesso do profissional preservando sua identidade e as referências atuais e futuras.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/professionals/`, `backend/db/query/professionals.sql`, queries compartilhadas de autenticação quando aplicável, `backend/internal/auth/middleware.go`, `backend/internal/httpapi/handler.go`, testes em `backend/integration/`, artefatos gerados e documentos de produto, arquitetura e escopo.
- **Base documental:** `docs/product.md`, “Serviços e profissionais inativos”; `docs/architecture.md`, “Autenticação”; `docs/implementation-scope.md`, itens 6, 9 e 22.
- **Requisitos de entrega:**
  - Somente proprietário com direito de escrita, origem confiável e CSRF válido desativa profissional do próprio estabelecimento; ID de proprietário nunca pode ser usado como alvo.
  - A transição ativo → inativo altera `users.active`, incrementa `credential_version` e revoga todas as sessões do profissional atomicamente; falha em qualquer escrita reverte a operação.
  - Repetir a desativação de um inativo retorna seu registro inativo com `200`, sem novo incremento de versão ou substituição de credencial.
  - Após o commit, os access tokens anteriores são rejeitados e novos logins são recusados; login e troca de senha concorrentes não deixam acesso válido após a desativação.
  - Perfil, e-mail e hash armazenado são preservados, sem exclusão ou cascata; a versão e o estado impedem que credenciais emitidas anteriormente recuperem acesso.
  - OpenAPI documenta a repetição segura e os erros; o checklist marca revogação e preservação dos registros existentes, mantendo pendentes contagem, alertas e referências dos futuros domínios de agenda e financeiro.
- **Verificação:** integração HTTP cobre múltiplas sessões, login negado, repetição, rollback e disputas com login e troca obrigatória; verificar isolamento, papel, CSRF parametrizado e `read_only` sem alteração persistida.
- **Dependências:** subtarefas 2 e 4.

### 6. Reativar com senha temporária e troca obrigatória

- **O que:** implementar `POST /api/v1/professionals/{professional_id}/reactivation` recebendo `temporary_password`, com nova credencial Argon2id e transição atômica.
- **Motivação:** devolver o acesso sem recuperar sessões antigas ou permitir que o profissional pule a definição da senha definitiva.
- **Onde:** `backend/openapi/openapi.yaml`, `backend/internal/professionals/`, `backend/db/query/professionals.sql`, `backend/internal/auth/middleware.go`, `backend/internal/httpapi/handler.go`, composição em `backend/cmd/api/main.go` e `backend/integration/suite_test.go` se necessária, testes em `backend/integration/`, artefatos gerados e documentos de produto, arquitetura e escopo.
- **Base documental:** `docs/architecture.md`, “Proprietário e profissional”; `docs/implementation-scope.md`, itens 5 e 9; decisões confirmadas nesta conversa sobre senha e conflito ao reativar ativo.
- **Requisitos de entrega:**
  - Somente proprietário com origem, CSRF e direito de escrita válidos reativa profissional inativo do seu estabelecimento.
  - Senha é obrigatória e segue `auth/security`: de 8 a 32 pontos de código, sem espaços ou caracteres proibidos e sem normalização. Ausência, `null`, tipo incorreto e campos desconhecidos são recusados.
  - Aceitar o mesmo valor da senha anterior conforme decisão confirmada, gerando novo hash com salt próprio. Não comparar a entrada para exigir diferença nem manter histórico de senhas.
  - Sob bloqueio iniciado pelo usuário, revalidar estado e tenant; substituir hash, ativar usuário, definir `password_change_required=true`, incrementar versão e revogar sessões anteriores na mesma transação.
  - Alvo já ativo retorna `409` com código estável `professional_already_active`, sem mudar senha, versão ou sessões. Entre duas reativações concorrentes do mesmo inativo, somente uma grava a credencial; a outra recebe conflito.
  - `200` retorna apenas `ProfessionalResponse`; não autentica o profissional nem devolve senha ou hash. A senha temporária permite login restrito, e a troca obrigatória existente continua exigindo senha diferente da temporária.
  - Atualizar no mesmo trabalho produto e arquitetura, substituindo a proibição de reutilizar o valor anterior pela permissão explícita e preservando a invalidação de sessões e tokens. Separar os checkboxes de backend e painel até a interface ser entregue.
- **Verificação:** unitários cobrem validação e recusas; integração HTTP cobre senha igual à anterior aceita, senha inválida, ativo sem efeitos, rollback, duas reativações concorrentes, sessões antigas recusadas, login com a senha aplicada e troca obrigatória. Verificar papel, tenant, CSRF e expiração.
- **Dependências:** subtarefa 5.

### 7. Consultar e editar pelo painel com estado compartilhado consistente

- **O que:** adicionar editor de nome à lista, cliente tipado e coordenação das operações do workspace, consumindo `can_write` e os endpoints de consulta e edição.
- **Motivação:** permitir a manutenção do cadastro pela página existente sem divergência entre lista, editor e registro persistido.
- **Onde:** `frontend/lib/professionals/professionals-api.ts`, módulos auxiliares previstos em `frontend/lib/professionals/`, `frontend/components/professionals/professionals-workspace/`, `professionals-list-panel/`, `create-professional-form/`, novo `edit-professional-panel/`, testes co-localizados, `frontend/app/(session)/painel/profissionais/page.test.tsx`, documentos de produto, arquitetura, operação e checklist.
- **Base documental:** `docs/architecture.md`, “Frontend”, “Estado” e “Ciclo das mutações”; `docs/product.md`, “Administração do catálogo”, como padrão existente de editor; `AGENTS.md`, design system.
- **Requisitos de entrega:**
  - Ação de editar abre consulta individual, inclusive para inativo, mostra e-mail somente para leitura e permite alterar apenas nome. Cancelar não grava e devolve foco à origem ou à lista.
  - Formulário compartilha a validação de conveniência do nome com o cadastro, mostra limites e envia PATCH somente quando o nome foi alterado.
  - `can_write: false`, consulta de direitos pendente ou falha nessa consulta bloqueiam escritas e preservam leitura; `read_only` recebido numa mutação atualiza esse estado na página, inclusive no cadastro existente.
  - Operações conflitantes, cadastro, edição, paginação e recarga são coordenados; respostas obsoletas por seleção ou identidade são descartadas e `401` remove conteúdo protegido, sem consulta redundante de sessão.
  - Sucesso confirmado atualiza a lista com o registro persistido. Falha de atualização da lista não desfaz sucesso; resultado incerto conserva o rascunho e exige consulta antes de novo envio, sem propagar rascunho à lista.
  - Reutilizar `RecordList`, `TextField`, `Button` e `Feedback`, seguindo o padrão de editor na própria página do catálogo. Qualquer novo estado ou variante visual necessário entra na referência do design system no mesmo trabalho.
  - Documentos descrevem edição entregue e roteiro manual com teclado, temas e tela estreita; checklist mantém transições de estado pendentes até suas interfaces serem concluídas.
- **Verificação:** Vitest cobre seleção e consulta, edição e cancelamento, erros de campo, expiração, envio único, resposta obsoleta, `401`, resultado incerto e sincronização da lista sem perder rascunho. Conferir teclado, foco e apresentação responsiva no ambiente local.
- **Dependências:** subtarefas 1, 2 e 3.

### 8. Desativar pelo painel com confirmação de perda de acesso

- **O que:** adicionar confirmação de desativação na própria página e cliente tipado para a operação.
- **Motivação:** tornar explícito que a ação encerra o acesso, preservando o cadastro e evitando envio acidental ou repetido após falha incerta.
- **Onde:** novo componente em `frontend/components/professionals/`, workspace, lista e editor de profissionais, `frontend/lib/professionals/`, testes co-localizados, documentos de produto, arquitetura, operação e checklist.
- **Base documental:** `docs/product.md`, “Serviços e profissionais inativos”; `docs/architecture.md`, “Ciclo das mutações”; `AGENTS.md`, design system.
- **Requisitos de entrega:**
  - Confirmação identifica o profissional, explica o encerramento de suas sessões e oferece cancelamento sem escrita.
  - Envio usa CSRF e coordenação do workspace; sucesso confirmado mostra inativo na lista e no editor sem apagar o registro.
  - Resultado incerto exige consulta individual antes de outra ação; a consulta atualiza apenas o estado confirmado, sem repetição automática da desativação.
  - Sessão ausente, recurso não encontrado, acesso expirado e falhas recebem tratamento por códigos locais, sem mensagens brutas do servidor.
  - Não exibir contagens fictícias de agendamentos; documentação e checklist distinguem desativação entregue dos alertas ainda dependentes da agenda.
- **Verificação:** Vitest cobre confirmação, cancelamento, envio único, sincronização do editor, resultado incerto e `read_only`; roteiro manual mantém uma sessão do profissional aberta para confirmar sua recusa após desativação.
- **Dependências:** subtarefas 5 e 7.

### 9. Reativar pelo painel sem expor a senha temporária

- **O que:** adicionar formulário de reativação com `PasswordField`, confirmação e cliente tipado, concluindo a gestão de estado na página existente.
- **Motivação:** permitir que o proprietário devolva acesso e forneça a senha temporária pelo mesmo padrão seguro do cadastro.
- **Onde:** novo componente em `frontend/components/professionals/`, workspace, lista e editor, `frontend/lib/professionals/`, testes co-localizados, referência do design system quando aplicável, documentos de produto, arquitetura, operação e checklist.
- **Base documental:** `docs/architecture.md`, “Proprietário e profissional” e “Ciclo das mutações”; decisões confirmadas de senha reutilizável e conflito em alvo ativo; `docs/implementation-scope.md`, itens 5, 6 e 9.
- **Requisitos de entrega:**
  - Oferecer reativação somente para inativo, coletando senha temporária com as mesmas regras e textos compartilhados do cadastro, sem exigir que difira da anterior.
  - Confirmar somente após `200`, atualizar lista e editor e informar a troca obrigatória; limpar senha após sucesso, cancelamento e saída do formulário. Nunca colocá-la em URL, logs, resposta, armazenamento persistente ou listagem.
  - `professional_already_active` não vira sucesso da credencial; orientar consulta e manter a garantia de que a recusa não alterou senha ou sessões.
  - Resultado incerto bloqueia reenvio e oferece consulta. A consulta confirma apenas o estado do cadastro, sem afirmar qual senha foi aplicada; se ativo, explicar que o acesso com a senha informada ainda precisa ser confirmado, sem repetir reativação ou redefinir senha automaticamente. Conservar a entrada somente em memória enquanto a reconciliação estiver pendente e limpar ao encerrá-la.
  - Sessão, erros, bloqueios e coordenação seguem o workspace; não permitir que paginação ou outro envio desmontem uma operação pendente.
  - Atualizar roteiro operacional para desativar, verificar revogação, reativar, entrar com a senha temporária e concluir troca. Marcar consulta, edição, ativação e desativação no painel somente após entrega; preservar pendências de associação, comissões e agenda.
- **Verificação:** Vitest cobre validação, cancelamento, limpeza da senha, envio único, conflito, leitura após resultado incerto, troca de identidade, expiração e sincronização das representações do registro. Validação manual percorre reativação com valor diferente e igual ao anterior, login restrito e troca obrigatória.
- **Dependências:** subtarefas 6 e 8.

## Verificação conjunta

- Executar `make check` e `make build`; `make check` inclui geração verificável, `gofmt`, `go vet -tags=integration ./...`, testes unitários e integração HTTP com PostgreSQL real, Prettier, ESLint e tipos. Não adicionar E2E ou integração de frontend.
- Revisar os fluxos de edição, desativação, reativação, login e troca obrigatória em conjunto, com atenção a bloqueios, rollback, tenant, papel, CSRF parametrizado, versão de credencial e preservação da identidade.
- Comparar OpenAPI, artefatos gerados, código, testes e documentos. Conferir que a permissão de reutilizar a senha anterior não restaura sessões nem tokens antigos, e que os checkboxes marcam somente capacidades efetivamente entregues.
