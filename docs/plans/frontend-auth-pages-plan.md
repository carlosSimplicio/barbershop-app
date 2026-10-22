# Primeiras páginas e sessão do frontend

**Registro histórico:** as etapas de refresh, recuperação automática e Web Lock deste plano foram substituídas por [access-csrf-only-auth-plan.md](access-csrf-only-auth-plan.md). O frontend atual envia os cookies e encerra a sessão local ao receber `401`, conforme `docs/architecture.md`. As etapas abaixo permanecem como registro da implementação original, não como instruções para novas mudanças.

Entregar a home deslogada, o fluxo completo de cadastro e confirmação de e-mail, o login e uma área autenticada de testes que permita validar todos os endpoints atuais de sessão e estabelecimento.

### 1. Criar o padrão compartilhado de feedback do design system

- **O que:** adicionar um componente acessível de feedback para mensagens de sucesso, aviso e erro, com API pública, estilos nos temas claro e escuro, teste unitário e exemplos na referência React. O componente deve comportar mensagens gerais e ações relacionadas sem substituir os erros associados diretamente aos campos.
- **Motivação:** cadastro, confirmação, login e ações da tela de testes precisam comunicar estados assíncronos de forma consistente; o checklist ainda registra feedback compartilhado como pendente.
- **Onde:** novo componente em `frontend/components/ui/feedback`; barrel `frontend/components/ui/index.ts`; referência em `frontend/app/design-system`.
- **Base documental:** `AGENTS.md`, regra universal do design system; `docs/architecture.md`, seção “Frontend”; `docs/implementation-scope.md`, seção 4, itens de padrões compartilhados de feedback, acessibilidade e adesão ao design system.
- **Verificação:** testes Vitest comprovam semântica acessível, associação de título e mensagem, variantes e ação opcional; a referência exibe os estados nos dois temas e em largura reduzida.

### 2. Tornar o transporte da API consciente de sessão e CSRF

- **O que:** manter `client.ts` como entrada pequena, separar transporte, leitura do cookie CSRF e interpretação de respostas em `transport.ts`, e concentrar a recuperação de leituras em `session.ts`. Preservar status e erros tipados sem expor access ou refresh tokens ao JavaScript. Somente `GET`/`HEAD` autenticados que retornarem `401` entram na recuperação automática: uma promessa compartilhada evita refresh duplicado na aba e o Web Lock coordena abas. Dentro do lock, consultar `/auth/me` antes de renovar; se outra aba já autenticou, repetir automaticamente apenas `/auth/me`, para que leituras de domínio antigas não recebam dados de outra conta. Após refresh `204`, repetir a leitura uma vez. `401` definitivo notifica sessão inválida; falha indeterminada não reutiliza automaticamente o token nessa aba. Mutações, inclusive login, logout, troca de senha e refresh manual, seguem diretamente ao transporte nesta etapa e terão tratamento explícito no `AuthProvider` e nas ações futuras.
- **Motivação:** os cookies são rotativos e o refresh token é de uso único; renovações concorrentes independentes poderiam ser interpretadas como reutilização e revogar a família da sessão. As páginas também precisam consumir o contrato OpenAPI sem duplicar serialização, erros, CSRF ou política de repetição.
- **Onde:** `frontend/lib/api/client.ts`, `frontend/lib/api/transport.ts`, `frontend/lib/api/session.ts` e testes; tipos de `frontend/types/generated/api.ts` apenas como dependência gerada, sem edição manual.
- **Base documental:** `docs/architecture.md`, seções “API contract-first”, “Erros”, “Autenticação”, “Contrato de sessão”, “Frontend”, “Renderização e API” e “Estado”; `docs/implementation-scope.md`, seções 4, 5 e 22; contrato vigente em `backend/openapi/openapi.yaml`.
- **Verificação:** testes Vitest cobrem resultados tipados, envio de credenciais, inclusão de CSRF apenas quando exigido, uma única renovação para leituras concorrentes na mesma aba, parada após falha, aquisição do Web Lock, revalidação que evita segunda renovação entre abas, ausência de retry automático em mutações e em leituras de domínio após mudança externa de sessão, fallback sem Web Locks e repetição única. Confirmar que nenhum token sensível é persistido em `localStorage`, `sessionStorage`, URL ou logs.

### 3. Centralizar o estado da sessão e proteger a navegação

- **O que:** criar o cliente tipado de `GET /auth/me` e `POST /auth/refresh` e um único Context global de sessão, restrito a identidade e estado transversal, que registre uma vez a reação a uma sessão inválida. Representar carregamento, sessão autenticada e sessão ausente; inicializar a identidade com recuperação automática da subtarefa 2 e expor operações explícitas para sincronizar login, confirmação, troca de senha, refresh manual e logout, inclusive reiniciar a recuperação após uma nova sessão. Coordenar no Provider essas transições com uma renovação automática em andamento antes de reiniciar o estado, para que uma tentativa antiga não altere uma sessão nova; o transporte não serializa mutações. Criar guardas reutilizáveis que enviem visitantes sem sessão de `/testes` para `/login` e usuários autenticados que abrirem `/login` ou `/cadastro` para `/testes`, usando substituição de histórico e uma tela de carregamento sem piscar conteúdo protegido.
- **Motivação:** os cookies `HttpOnly` não podem ser usados diretamente pelo React para decidir a identidade, e cada página não deve implementar sua própria descoberta ou recuperação de sessão.
- **Onde:** componentes e hooks previstos sob `frontend/components/session` e `frontend/lib/auth`; composição no menor layout necessário sob `frontend/app`, sem transformar componentes estáticos em Client Components sem necessidade.
- **Base documental:** `docs/architecture.md`, seções “Frontend”, “Renderização e API” e “Estado”; `docs/implementation-scope.md`, seções 4 e 5; decisão confirmada pelo usuário sobre guardas e uma tentativa automática de refresh.
- **Verificação:** testes Vitest cobrem inicialização autenticada, recuperação após access token expirado, sessão ausente, redirecionamentos, atualização do principal e logout; durante o carregamento, nenhum conteúdo protegido ou formulário público incompatível é exibido.
- **Dependências:** subtarefa 2.

### 4. Substituir a página provisória pela home deslogada

- **O que:** implementar `/` como uma landing page responsiva em `pt-BR`, com proposta da plataforma e chamadas claras para `/cadastro` e `/login`, reutilizando tokens e componentes do design system. Remover a apresentação provisória do health check da home, sem remover o endpoint ou seu cliente se ainda tiver uso técnico legítimo.
- **Motivação:** oferecer uma entrada pública coerente para proprietários antes da autenticação e conectar os dois fluxos solicitados.
- **Onde:** `frontend/app/page.tsx`, CSS Module co-localizado previsto para a página e metadados em `frontend/app/layout.tsx` ou na própria rota.
- **Base documental:** `docs/product.md`, seções “Visão e públicos” e “Experiência de uso”; `docs/architecture.md`, seções “Frontend” e “Renderização e API”; `docs/implementation-scope.md`, seção 4; decisão do usuário de criar uma home deslogada.
- **Verificação:** teste Vitest comprova título, conteúdo principal e destinos dos CTAs; revisão manual confirma teclado, foco, contraste e adaptação a larguras móveis e desktop, sem chamadas desnecessárias à API.
- **Dependências:** subtarefa 1.

### 5. Entregar cadastro com estado de confirmação pendente

- **O que:** transformar o protótipo existente em `/cadastro`, usando os componentes reais de formulário e um cliente tipado de `POST /auth/register`. O formulário recebe nome do estabelecimento, slug, fuso IANA, e-mail e senha; sugere o slug a partir do nome até edição manual; detecta o fuso do dispositivo, usa `America/Sao_Paulo` como fallback e permite alterá-lo antes do envio. Aplicar validação de conveniência no cliente sem substituir o backend, mapear erros gerais e por campo pelo código estável e, após `202`, substituir o formulário por um estado persistente que oriente a conferir o e-mail.
- **Motivação:** tornar funcional o cadastro público já entregue no backend e conduzir o usuário até a etapa obrigatória de ativação sem iniciar o trial prematuramente.
- **Onde:** rota e estilos previstos em `frontend/app/cadastro`; componentes específicos previstos em `frontend/components/auth`; referência visual em `frontend/prototypes/cadastro.html` e `frontend/prototypes/cadastro.css`, adaptada aos tokens e componentes atuais, sem copiar seus controles isolados; cliente de onboarding da subtarefa 2.
- **Base documental:** `docs/product.md`, seções “Cadastro, usuários e permissões” e “Planos, assinatura e trial”; `docs/architecture.md`, seções “Erros”, “Contrato de sessão” e “Frontend”; `docs/implementation-scope.md`, seções 4, 5, 7, 21 e 22; decisão confirmada pelo usuário de incluir estado pós-cadastro e reenvio.
- **Verificação:** testes Vitest cobrem preenchimento e envio, detecção e fallback do fuso, sugestão e edição do slug, validações relevantes, foco no primeiro campo inválido, conflitos específicos de e-mail e slug, rate limit/erro geral e estado `202`. Confirmar que o trial não é apresentado como iniciado antes da confirmação.
- **Dependências:** subtarefas 1 e 2.

### 6. Permitir reenvio da confirmação pendente

- **O que:** criar o cliente tipado de `POST /auth/email-confirmation` e acrescentar ao estado pós-cadastro uma ação de reenvio para o mesmo e-mail, com carregamento independente e resposta de sucesso genérica. Disponibilizar a mesma ação a partir do login quando o backend retornar `email_confirmation_required`, sem revelar se a conta existe, já foi confirmada ou recebeu um novo token.
- **Motivação:** recuperar confirmação não recebida ou expirada preservando a proteção anti-enumeração do backend.
- **Onde:** cliente previsto sob `frontend/lib/auth`; componentes compartilhados sob `frontend/components/auth`; estados consumidores em `frontend/app/cadastro` e, após a subtarefa 8, `frontend/app/login`.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seção “Contrato de sessão”; `docs/implementation-scope.md`, seções 5, 21 e 22; decisão confirmada pelo usuário de incluir reenvio.
- **Verificação:** testes Vitest cobrem payload de e-mail, `202`, bloqueio de envio duplicado, rate limit, falha inesperada e a mesma mensagem de sucesso independentemente do estado real da conta.
- **Dependências:** subtarefas 1, 2 e 5.

### 7. Confirmar o e-mail pelo fragmento e iniciar a sessão

- **O que:** implementar `/confirmar-email` como fronteira interativa mínima que extrai `token` de `window.location.hash`, remove imediatamente o fragmento com `history.replaceState` antes de qualquer navegação ou chamada de API e envia o valor somente no corpo de `POST /auth/email-confirmation/verify`. Exibir estados de processamento, sucesso e link inválido/expirado; no sucesso, atualizar a sessão e substituir a rota por `/testes`. O token nunca deve aparecer em UI, logs, atributos, path, query ou estado persistente.
- **Motivação:** completar com segurança o cadastro, iniciar o único trial e aproveitar a autenticação automática já retornada pelo backend.
- **Onde:** rota e estilos previstos em `frontend/app/confirmar-email`; componente específico sob `frontend/components/auth`; cliente e Context das subtarefas 2 e 3.
- **Base documental:** `AGENTS.md`, regra universal para links transacionais; `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Contrato de sessão” e “Frontend”; `docs/implementation-scope.md`, seções 5, 7 e 22; decisão confirmada pelo usuário de entregar a confirmação funcional.
- **Verificação:** testes Vitest cobrem fragmento válido, remoção anterior à requisição, token ausente, resposta inválida/expirada, erro inesperado e sucesso com sincronização explícita do Provider e redirecionamento. Inspeção confirma ausência do token em URL após a captura, mensagens, armazenamento e logs.
- **Dependências:** subtarefas 1–3.

### 8. Entregar login para proprietário e profissional

- **O que:** criar `/login` e o cliente tipado de `POST /auth/login`, com e-mail e senha, preenchimento automático apropriado, estados de envio, erros por campo e mensagens gerais decididas por código. Após sucesso, registrar o principal no Context e substituir a rota por `/testes`, preservando `password_change_required` para que a área autenticada apresente a troca obrigatória. Para `email_confirmation_required`, orientar o proprietário e reutilizar a solicitação de nova confirmação da subtarefa 6.
- **Motivação:** permitir que os dois papéis usem as sessões existentes e alcancem a tela técnica solicitada, inclusive o fluxo especial de senha temporária.
- **Onde:** rota e estilos previstos em `frontend/app/login`; componentes específicos sob `frontend/components/auth`; clientes e Context das subtarefas 2 e 3.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Autenticação”, “Contrato de sessão” e “Proprietário e profissional”; `docs/implementation-scope.md`, seções 4 e 5; decisão do usuário de redirecionar o login para `/testes`.
- **Verificação:** testes Vitest cobrem login válido dos dois papéis, credenciais inválidas genéricas, confirmação obrigatória e reenvio, rate limit, erro inesperado, bloqueio de envio duplicado, sincronização explícita do Provider e redirecionamento sem manter `/login` no histórico.
- **Dependências:** subtarefas 1–3 e 6.

### 9. Criar a área autenticada e permitir nova consulta da sessão

- **O que:** implementar `/testes` como página explicitamente temporária e protegida, exibir somente os dados não secretos do `AuthPrincipal` mantido no Context e oferecer uma ação para executar novamente `GET /auth/me`. Preparar seções independentes para as demais capacidades, sem mostrar cookies ou tokens.
- **Motivação:** fornecer a casca controlada da área técnica e tornar observável o endpoint que identifica a sessão atual.
- **Onde:** rota e estilos previstos em `frontend/app/testes`; componentes previstos em `frontend/components/testing`; clientes da subtarefa 2 e guarda/Context da subtarefa 3.
- **Base documental:** `docs/architecture.md`, seções “Contrato de sessão”, “Frontend” e “Estado”; `docs/implementation-scope.md`, seções 4 e 5; decisão confirmada pelo usuário de expor todos os endpoints autenticados atuais na tela de testes.
- **Verificação:** testes Vitest cobrem principal inicial, nova consulta bem-sucedida, erro da ação, sessão perdida com redirecionamento para `/login` e ausência de dados sensíveis.
- **Dependências:** subtarefas 1–3 e 8.

### 10. Consultar o estabelecimento na área de testes

- **O que:** criar o cliente tipado de `GET /establishment` e uma seção em `/testes` que apresente nome, slug, fuso IANA e trial ausente, ativo ou expirado. Manter a ação indisponível enquanto `password_change_required` for verdadeiro, refletindo a autorização do backend.
- **Motivação:** validar o primeiro endpoint autenticado de domínio e tornar observável que o tenant vem exclusivamente da sessão.
- **Onde:** módulo previsto em `frontend/lib/establishment`; componentes de `frontend/components/testing`; página `frontend/app/testes`.
- **Base documental:** `docs/product.md`, seções “Cadastro, usuários e permissões” e “Planos, assinatura e trial”; `docs/architecture.md`, seções “Multi-tenancy e página pública” e “Contrato de sessão”; `docs/implementation-scope.md`, seções 6, 7 e 20.
- **Verificação:** testes Vitest cobrem sucesso, trial nulo/ativo/expirado, erro, prevenção de resposta obsoleta e indisponibilidade durante troca obrigatória de senha; nenhum identificador de estabelecimento é enviado como autorização.
- **Dependências:** subtarefa 9.

### 11. Expor a renovação manual da sessão

- **O que:** adicionar a `/testes` uma ação explícita de `POST /auth/refresh`, coordenada pelo Provider com eventual renovação automática em andamento e com os mesmos cookies e CSRF do transporte. Após sucesso, reconsultar ou preservar coerentemente o principal; após falha de sessão, limpar o estado e redirecionar para `/login`.
- **Motivação:** permitir testar diretamente a rotação do refresh token sem criar uma segunda implementação concorrente da operação.
- **Onde:** cliente em `frontend/lib/auth`; Context da subtarefa 3; seção de ação em `frontend/components/testing` e `frontend/app/testes`.
- **Base documental:** `docs/architecture.md`, seções “Autenticação” e “Contrato de sessão”; `docs/implementation-scope.md`, seção 5; decisão confirmada pelo usuário de testar refresh manual e automático.
- **Verificação:** testes Vitest cobrem sucesso, prevenção de cliques concorrentes, erro recuperável e sessão inválida; confirmar que a ação não lê nem exibe o refresh token.
- **Dependências:** subtarefas 2, 3 e 9.

### 12. Permitir a troca obrigatória da senha temporária

- **O que:** criar o cliente tipado de `POST /auth/password/change` e mostrar em `/testes`, somente quando `password_change_required` for verdadeiro, um formulário para nova senha. Mapear os erros de senha inválida ou inalterada, sincronizar os cookies reemitidos e atualizar o principal após o sucesso, liberando as ações de domínio.
- **Motivação:** profissionais com senha temporária não podem usar o restante do produto até concluir essa operação, embora possam manter ou encerrar a sessão.
- **Onde:** cliente e estado em `frontend/lib/auth`; formulário em `frontend/components/testing`; página `frontend/app/testes`.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Contrato de sessão” e “Proprietário e profissional”; `docs/implementation-scope.md`, seção 5.
- **Verificação:** testes Vitest cobrem visibilidade condicional, validação, senha igual à temporária, erro geral, sincronização explícita do Provider após sucesso com atualização de `password_change_required` e liberação da consulta do estabelecimento.
- **Dependências:** subtarefas 3, 9 e 10.

### 13. Encerrar a sessão pela área de testes

- **O que:** criar o cliente tipado de `POST /auth/logout` e uma ação em `/testes` que encerre idempotentemente a sessão atual, limpe o Context mesmo após resposta vazia válida e substitua a rota por `/login`. Em falha interna, preservar o estado coerente e oferecer nova tentativa sem indicar que cookies foram removidos quando não foram.
- **Motivação:** completar o ciclo de sessão e permitir validar a revogação e remoção de cookies pelo backend.
- **Onde:** cliente e estado em `frontend/lib/auth`; ação em `frontend/components/testing`; página `frontend/app/testes`.
- **Base documental:** `docs/architecture.md`, seção “Contrato de sessão”; `docs/implementation-scope.md`, seção 5; decisão confirmada pelo usuário de testar logout.
- **Verificação:** testes Vitest cobrem `204`, repetição segura, prevenção de cliques duplicados, falha interna, limpeza explícita do principal no Provider e redirecionamento com substituição de histórico.
- **Dependências:** subtarefas 2, 3 e 9.

### 14. Sincronizar documentação, checklist e qualidade da entrega

- **O que:** atualizar a documentação para substituir as afirmações de que onboarding e consumo de sessão ainda são manuais, registrar as rotas e o comportamento real de sessão do frontend e marcar ou dividir somente os checkboxes efetivamente concluídos. Revisar todo o diff, fluxos públicos e autenticados, responsividade, acessibilidade, tratamento de erros e aderência ao OpenAPI; não alterar o contrato ou artefatos gerados se a implementação confirmar que os endpoints atuais são suficientes.
- **Motivação:** a entrega torna funcional uma capacidade que os documentos registram explicitamente como pendente e precisa manter código, testes, contrato e escopo coerentes.
- **Onde:** `docs/product.md`, seções “Cadastro, usuários e permissões” e “Experiência de uso”; `docs/architecture.md`, seções “Contrato de sessão” e “Frontend”; `docs/implementation-scope.md`, seções 4, 5, 7, 22, 26 e definição de conclusão; `backend/openapi/openapi.yaml` apenas para conferência; este plano para registrar o estado após implementação, se o processo vigente exigir.
- **Base documental:** `AGENTS.md`, regras de documentação viva, verificação e revisão final; os mesmos documentos e contrato citados nas subtarefas anteriores.
- **Verificação:** comparar todos os payloads e códigos consumidos com o OpenAPI e os handlers; executar `npm test`, `npm run lint`, `npm run format:check`, `npm run typecheck` e `npm run build` em `frontend`; revisar a aplicação por teclado, em tema claro e escuro e em larguras móvel e desktop; confirmar que os protótipos não se tornaram uma segunda fonte de componentes.
- **Dependências:** subtarefas 1–13.

## Verificação conjunta

- Executar o fluxo manual por `https://localhost:8443`: home → cadastro → inspeção do e-mail no Mailpit → confirmação → `/testes` → consulta da sessão e estabelecimento → refresh → logout → login. Repetir o trecho login → `/testes` com profissional de senha temporária para validar o bloqueio e a troca obrigatória.
- Confirmar no navegador que cookies `Secure`, `HttpOnly`, `SameSite=Strict` e seus `Path` permanecem conforme o backend; somente o cookie CSRF é lido pelo frontend e nenhum token aparece em URL, logs ou armazenamento web.
- Não adicionar E2E ou testes de integração do frontend; a validação automatizada permanece em Vitest, conforme a decisão arquitetural vigente.

## Registro da verificação

- As subtarefas 1–13 foram implementadas em commits separados. A jornada do proprietário foi percorrida no navegador local: home, cadastro, confirmação pelo Mailpit, consultas em `/testes`, renovação, logout e novo login. O fragmento desapareceu da URL antes da verificação; os três cookies apresentaram os atributos e caminhos previstos. A inspeção exploratória cobriu largura móvel e desktop, foco por teclado e tema claro e escuro.
- O trecho manual com profissional de senha temporária não foi executado: a criação do profissional e a definição dessa senha ainda não foram entregues, como registra `docs/implementation-scope.md`. A troca e o bloqueio de domínio permanecem cobertos por testes Vitest e de integração do backend; a verificação manual depende de uma conta de teste preparada separadamente.
