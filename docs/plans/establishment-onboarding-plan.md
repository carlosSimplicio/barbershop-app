# Conclusão do onboarding do estabelecimento

Permitir ao proprietário configurar o estabelecimento no painel, enviar seu logo e publicar uma página informativa em `/{slug}`, preservando o acesso ao restante do painel durante o preenchimento.

### 1. Persistir os dados e limites configuráveis

- **O que:** adicionar persistência para dados públicos opcionais e limites de agenda do estabelecimento existente.
- **Motivação:** completar o perfil sem recriar contas, alterar slugs ou reiniciar trials.
- **Onde:** novas migrations em `backend/db/migrations/`, queries em `backend/db/query/`, artefatos `sqlc` em `backend/db/generated/`, `backend/integration/`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/product.md`, “Limites configuráveis” e “Experiência de uso”; escopos 3 e 7; campos e unidades confirmados nesta conversa.
- **Requisitos de entrega:**
  - Migrations novas e aditivas preservam estabelecimentos, usuários, slugs, fusos e trials existentes; os novos dados começam ausentes.
  - Descrição, endereço, telefone e lista ordenada de links possuem representação persistida e queries limitadas explicitamente ao estabelecimento.
  - Os três limites opcionais possuem representação inteira; `NULL` significa ausência de restrição, sem substituir valores ausentes por zero.
- **Verificação:** integração aplica migrations reais, preserva um cadastro anterior à nova migration e verifica leitura, nulabilidade e isolamento das queries; regenerar `sqlc`.

### 2. Preparar o bloqueio permanente de fuso

- **O que:** persistir o início dos dados operacionais e disponibilizar sua ativação dentro de uma transação existente.
- **Motivação:** impedir futuras alterações de fuso depois do primeiro agendamento, bloqueio ou lançamento financeiro, inclusive sob concorrência.
- **Onde:** nova migration, queries em `backend/db/query/`, `backend/internal/establishment/`, `backend/integration/`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/product.md`, “Localização”; escopo 7; decisão confirmada de preparar o bloqueio antes dos fluxos operacionais.
- **Requisitos de entrega:**
  - Um marcador persistido começa ausente e registra o início dos dados operacionais; sua ativação é idempotente e não apaga nem substitui o primeiro instante registrado.
  - A troca de fuso e a ativação do marcador utilizam o mesmo bloqueio transacional do estabelecimento; configurações não oferecem operação de limpar o marcador.
  - A arquitetura exige que os futuros primeiros agendamentos, bloqueios e lançamentos financeiros ativem o marcador na mesma transação de sua criação; essa integração continua explicitamente pendente no checklist.
- **Verificação:** integração com transações reais verifica ativação idempotente, rollback e serialização pelo bloqueio compartilhado; a recusa HTTP de troca será verificada na subtarefa 9. Regenerar `sqlc`.

### 3. Persistir metadados de logos com vínculo ao tenant

- **O que:** criar a estrutura e as queries para registrar arquivos e associar o logo atual ao estabelecimento.
- **Motivação:** manter no banco a autoridade sobre a imagem publicada, sem armazenar caminhos absolutos.
- **Onde:** nova migration, queries em `backend/db/query/`, `backend/db/generated/`, testes em `backend/integration/`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, “Banco de dados” e “Arquivos públicos”; escopos 3 e 8; decisão confirmada de manter referências compatíveis com futura substituição do armazenamento local por S3, a registrar na arquitetura nesta entrega.
- **Requisitos de entrega:**
  - Metadados registram identificador aleatório, estabelecimento, chave lógica estável do objeto, tipo validado, tamanho, dimensões e hash do conteúdo; estabelecimentos existentes começam sem logo.
  - A chave é gerada pela aplicação e funciona como referência relativa no adaptador local e como chave de objeto em um futuro bucket. Caminhos absolutos, nome do bucket, URLs do provedor e URLs assinadas não são persistidos como identidade do arquivo.
  - A associação do logo atual não pode apontar para arquivo de outro tenant, inclusive por gravação direta no banco.
  - Queries permitem associar e consultar o arquivo atual sob transação, sem aceitar caminhos ou tenant fornecidos como autoridade pelo cliente.
- **Verificação:** integração verifica associação, substituição transacional e rejeição estrutural de referência cruzada entre tenants; regenerar `sqlc`.

### 4. Reservar slugs que colidem com a plataforma

- **O que:** ampliar a reserva de slugs e identificar cadastros legados que não podem receber a página canônica.
- **Motivação:** impedir que o endereço de um estabelecimento coincida com cadastro, login ou navegação administrativa.
- **Onde:** `backend/internal/onboarding/validation.go`, seus testes, `backend/integration/onboarding_test.go`, `backend/internal/establishment/`, `docs/architecture.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, “Multi-tenancy e página pública”; escopo 7; decisão confirmada de ampliar a reserva, preservar slugs existentes e resolver conflitos explicitamente antes da publicação.
- **Requisitos de entrega:**
  - A reserva cobre `admin`, `api`, `assets`, `cadastro`, `login`, `confirmar-email`, `recuperar-senha`, `redefinir-senha`, `painel`, `testes`, `design-system` e os prefixos técnicos aplicáveis às rotas existentes.
  - Novos cadastros com slug reservado recebem o erro de campo já adotado, sem transformação silenciosa do valor.
  - Cadastros legados conflitantes são preservados; a regra de elegibilidade sinaliza o conflito para a futura consulta de configurações e impede a publicação naquele caminho; a operação possui procedimento de diagnóstico sem renomeação automática nem exposição de dados em logs.
- **Verificação:** testes unitários de reserva e integração HTTP de recusa no cadastro; conferir a lista contra as rotas reais de `frontend/app/` e do Caddy.

### 5. Isolar o armazenamento em uma interface com adaptador local

- **O que:** definir uma interface pequena de armazenamento de objetos e implementar seu adaptador local, com escrita, leitura e remoção por chave lógica.
- **Motivação:** permitir substituir o volume local por S3 preservando os casos de uso, as validações, os metadados e os contratos públicos.
- **Onde:** interface consumida em `backend/internal/establishment/`, novo adaptador previsto em `backend/internal/platform/storage/`, seus testes, `docs/architecture.md`, “Arquivos públicos”, e `docs/implementation-scope.md`, escopo 8.
- **Base documental:** escopo 8; `docs/architecture.md`, “Arquivos públicos”; `docs/operations.md`, “Persistência”; solicitação do usuário de preparar a substituição futura por S3. Registrar a interface e o adaptador local como implementados na mesma entrega, mantendo S3 como evolução futura.
- **Requisitos de entrega:**
  - A interface recebe contexto e chaves opacas, transporta conteúdo por streams e define escrita, abertura para leitura e remoção. Arquivo ausente, falha de armazenamento e remoção de objeto já ausente têm semântica estável, independente do provedor.
  - Handlers e casos de uso dependem dessa interface; caminhos, `os.File`, operações de filesystem e tipos de um futuro SDK AWS ficam restritos aos adaptadores. Validação de imagem, autorização e associação transacional continuam nos casos de uso.
  - Objetos recebem chaves aleatórias geradas pela aplicação e são criados sem sobrescrever objetos existentes. No adaptador local, a conversão da chave em caminho é interna e fica confinada à raiz configurada, sem seguir referências que escapem dela.
  - Escritas incompletas não ficam disponíveis para leitura; falhas removem temporários e não modificam o objeto atualmente associado ao estabelecimento. A interface não exige renomeação de arquivo ou transação distribuída entre storage e PostgreSQL.
  - O adaptador entregue nesta etapa é o local. A futura implementação S3 utilizará o mesmo contrato; sua adoção exigirá copiar os objetos preservando as chaves e trocar o adaptador na composição da aplicação.
- **Verificação:** uma suíte reutilizável do contrato verifica escrita e leitura do conteúdo, ausência, remoção idempotente e preservação de objetos existentes, executada agora contra o adaptador local com diretório temporário real. Testes locais adicionais protegem falha de gravação e confinamento; revisar a ausência de dependências de filesystem nos casos de uso.

### 6. Disponibilizar o volume de logos ao processo da aplicação

- **O que:** configurar a raiz de armazenamento e preparar permissões do volume para o usuário não privilegiado do backend.
- **Motivação:** o volume `logos_data` já está montado, mas a aplicação ainda não o utiliza e a imagem executa como `app`.
- **Onde:** `backend/internal/config/config.go`, `backend/cmd/api/main.go`, `backend/Dockerfile`, `compose.yaml`, `.env.example`, testes de configuração, `docs/operations.md` e `README.md`.
- **Base documental:** `docs/operations.md`, “Infraestrutura”, “Configuração e segredos” e “Persistência”; escopos 8 e 24; decisão confirmada de selecionar o adaptador na composição da aplicação.
- **Requisitos de entrega:**
  - `LOGO_STORAGE_DIR` identifica a raiz interna; no Compose aponta para `/var/lib/barbershop/logos`, usando o volume persistente existente.
  - `backend/cmd/api/main.go` instancia o adaptador local e o injeta pela interface nos consumidores; a raiz local pertence à configuração desse adaptador e não aparece nos contratos HTTP ou nas regras de negócio.
  - A inicialização valida a configuração e a capacidade de acesso ao diretório; o backend continua executando sem privilégios de root.
  - Recriar o container preserva arquivos; a configuração não incorpora uploads à imagem nem publica o diretório diretamente no Caddy.
  - A operação documenta permissões, falhas de armazenamento e diagnóstico de resíduos de interrupções, com limpeza manual que verifica referências no banco antes de remover arquivos. A futura migração para S3 deve preservar as chaves e validar a cópia antes de mudar o adaptador ou remover os originais.
- **Verificação:** testes de configuração, `make compose-config`, build da imagem e validação local de escrita e persistência com o usuário real do container.
- **Dependências:** subtarefa 5.

### 7. Validar o conteúdo dos logos antes da gravação

- **O que:** implementar validação de conteúdo, tamanho e dimensões independente do handler HTTP.
- **Motivação:** aceitar os formatos documentados sem confiar em extensão ou `Content-Type` e limitar o custo de decodificação.
- **Onde:** novo validador em `backend/internal/establishment/`, testes unitários, `backend/go.mod` e `backend/go.sum` se necessário para WebP, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** `docs/architecture.md`, “Arquivos públicos”; escopo 8; decisão confirmada de limitar a 2 MiB e 4.096 × 4.096 pixels.
- **Requisitos de entrega:**
  - Somente PNG, JPEG e WebP decodificáveis são aceitos; SVG, conteúdo corrompido e assinatura incompatível são recusados.
  - O arquivo possui no máximo 2.097.152 bytes e dimensões positivas de até 4.096 pixels por eixo; o limite de bytes e a inspeção das dimensões precedem a decodificação completa.
  - Tipo e dimensões persistidos vêm do conteúdo validado; nome original e cabeçalhos do cliente não determinam o tipo nem o caminho final.
  - Eventual dependência de decodificação WebP tem versão fixa e necessidade registrada, seguindo os padrões do projeto.
- **Verificação:** testes unitários com imagens pequenas geradas ou fixtures não sensíveis cobrem formatos aceitos, limites, arquivo truncado e dimensões excessivas.

### 8. Consultar as configurações do próprio estabelecimento

- **O que:** entregar `GET /api/v1/establishment/settings` como leitura exclusiva do proprietário, mantendo a consulta básica existente compatível.
- **Motivação:** fornecer os dados necessários ao formulário e à orientação de conclusão sem ampliar a exposição aos profissionais.
- **Onde:** `backend/openapi/openapi.yaml` e gerados, `frontend/types/generated/api.ts`, `backend/internal/establishment/`, `backend/internal/httpapi/handler.go`, `backend/cmd/api/main.go`, `backend/integration/`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** escopos 6 e 7; `docs/product.md`, “Cadastro, usuários e permissões” e “Planos, assinatura e trial”; decisões confirmadas sobre configuração no painel e publicação após os dados obrigatórios.
- **Requisitos de entrega:**
  - A resposta inclui nome, slug imutável, fuso, dados opcionais, limites, metadados públicos do logo e situação de preenchimento/publicação, incluindo conflito de slug legado quando houver.
  - A resposta informa a possibilidade de editar o fuso e de realizar escritas a partir do marcador operacional e do componente central de direitos de acesso.
  - Somente o proprietário consulta essas configurações, com tenant derivado do `Principal`; a leitura continua disponível após expiração do trial.
  - Caminho interno, nome original do arquivo e dados de credenciais não aparecem na resposta; `GET /establishment` e a área `/testes` preservam o contrato existente.
- **Verificação:** testes unitários e integração HTTP cobrem proprietário, profissional, sessão ausente, tenant diferente, trial expirado e compatibilidade da consulta básica; regenerar os artefatos do OpenAPI.
- **Dependências:** subtarefas 1, 2, 3 e 4.

### 9. Atualizar dados públicos e limites de agenda

- **O que:** entregar `PATCH /api/v1/establishment/settings` com validação autoritativa e atualização transacional dos campos enviados.
- **Motivação:** permitir concluir e manter o perfil sem alterar a identidade pública ou regras já fixadas pelo primeiro dado operacional.
- **Onde:** OpenAPI e gerados Go/TypeScript, `backend/internal/establishment/`, `backend/internal/auth/middleware.go`, `backend/internal/httpapi/handler.go`, queries, testes unitários e de integração, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** escopo 7; `docs/product.md`, “Localização”, “Limites configuráveis” e “Experiência de uso”; decisões confirmadas sobre campos, limites e unidades.
- **Requisitos de entrega:**
  - Nome obrigatório mantém o limite de 120 caracteres; descrição e endereço são opcionais, com até 500 cada; telefone brasileiro é normalizado no backend.
  - São aceitos até cinco links ordenados, cada um com rótulo não vazio de até 60 caracteres e URL HTTPS de até 2.048 caracteres, sem credenciais; textos são tratados como texto, sem HTML executável.
  - Antecedência e prazo de cancelamento/reagendamento são minutos inteiros não negativos; horizonte é número inteiro positivo de dias. Ausência no PATCH preserva o valor; `null` remove uma restrição ou limpa um dado opcional. Lista vazia remove os links.
  - Slug e marcador operacional não podem ser alterados pelo corpo. O fuso deve ser IANA válido e só pode mudar enquanto o marcador estiver ausente, com bloqueio transacional que também será usado pelos futuros primeiros registros operacionais.
  - A operação exige proprietário, tenant da sessão, origem confiável, CSRF e direito de escrita; falhas não deixam alterações parciais e trial expirado resulta em `read_only`.
  - A documentação registra as unidades e limites escolhidos e diferencia a configuração entregue da aplicação futura desses limites no agendamento.
- **Verificação:** testes unitários protegem validações e semântica de campos ausentes/nulos; integração HTTP verifica persistência, recusas, isolamento e fuso bloqueado com marcador persistido. Regenerar OpenAPI e `sqlc` afetados.
- **Dependências:** subtarefas 1, 2 e 8.

### 10. Enviar e substituir o logo do estabelecimento

- **O que:** entregar `PUT /api/v1/establishment/logo` com upload multipart e associação segura do arquivo validado.
- **Motivação:** completar o último dado público obrigatório e permitir substituições sem perder o logo válido em caso de falha.
- **Onde:** OpenAPI e gerados, `backend/internal/establishment/`, middleware de autenticação/CSRF, wiring HTTP, queries, testes unitários e de integração, `docs/product.md`, `docs/architecture.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** escopos 7 e 8; decisões confirmadas de publicar ao completar os obrigatórios e permitir substituição, sem remoção após publicação.
- **Requisitos de entrega:**
  - O corpo é limitado antes de ser integralmente consumido, com margem delimitada para o envelope multipart; o arquivo continua limitado a 2 MiB e passa pelo validador de conteúdo.
  - Proprietário, tenant da sessão, origem, CSRF e direito de escrita são verificados; upload recusado não muda o perfil nem publica o estabelecimento.
  - O novo arquivo está íntegro antes da associação transacional dos metadados. Substituições concorrentes não apagam o arquivo atualmente referenciado, e falhas preservam o logo anterior.
  - Gravação e compensações usam somente a interface de storage. O caso de uso gera a chave e o hash do conteúdo, sem conhecer diretórios locais, buckets ou URLs do provedor.
  - Arquivos novos não associados são compensados quando a falha é confirmada; o arquivo anterior só é removido após a nova associação estar confirmada. Resultado incerto de commit exige conferir a referência antes de apagar arquivos.
  - Upload válido conclui o preenchimento obrigatório quando nome e slug utilizável já existem. Não há operação que remova o logo publicado; a resposta não expõe caminhos nem nome original.
  - Falhas de compensação geram evento operacional sem dados sensíveis e têm procedimento de recuperação documentado; o checklist distingue upload entregue de publicação efetivamente concluída.
- **Verificação:** integração usa PostgreSQL e diretório temporário reais para upload, troca, concorrência, conteúdo inválido, excesso de tamanho, CSRF, papel e trial expirado; testes unitários exercitam falhas relevantes de associação e compensação. Regenerar artefatos afetados.
- **Dependências:** subtarefas 3, 5, 6, 7 e 8.

### 11. Servir o logo público com cache revalidável

- **O que:** entregar `GET /api/v1/public/establishments/{slug}/logo` para o arquivo atualmente associado a um estabelecimento publicável.
- **Motivação:** disponibilizar imagens sem publicar o volume nem expor caminhos internos.
- **Onde:** OpenAPI e gerados, `backend/internal/establishment/`, `backend/internal/auth/middleware.go`, wiring HTTP, testes de integração, `docs/architecture.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** escopo 8; `docs/architecture.md`, “Arquivos públicos” e “Multi-tenancy e página pública”.
- **Requisitos de entrega:**
  - A resolução por slug é centralizada e a consulta vincula explicitamente logo e estabelecimento; a chave usada para leitura vem apenas do banco, e o conteúdo é aberto pela interface de storage.
  - O endpoint é público somente para leitura; a exceção no middleware não libera mutações nem outras rotas autenticadas.
  - A resposta usa tipo validado, tamanho, `nosniff`, `ETag` derivado do hash persistido do conteúdo e política de revalidação que permite perceber substituições; acesso condicional pode retornar `304`. O contrato não depende de metadados do filesystem nem do `ETag` de um provedor.
  - O endereço público continua sendo o endpoint da aplicação; clientes não recebem caminhos locais nem URLs do storage, permitindo a futura troca do adaptador sem alterar frontend ou OpenAPI.
  - Slug ausente, não publicável ou sem logo retorna `404`; indisponibilidade de armazenamento produz erro operacional sem revelar caminhos. A imagem publicada permanece acessível com trial expirado.
- **Verificação:** integração HTTP com arquivos reais verifica bytes, cabeçalhos, troca do logo, revalidação, tenant e ausência de autenticação; comparar erros com o contrato e regenerar os artefatos.
- **Dependências:** subtarefas 3, 4, 5, 6 e 10.

### 12. Expor somente o perfil público publicável

- **O que:** entregar `GET /api/v1/public/establishments/{slug}` com o perfil informativo mínimo.
- **Motivação:** fornecer a leitura server-side da página pública usando o backend como autoridade de publicação.
- **Onde:** OpenAPI e gerados, `backend/internal/establishment/`, middleware, wiring HTTP, queries, testes unitários e de integração, `docs/product.md`, `docs/architecture.md` e `docs/implementation-scope.md`.
- **Base documental:** escopos 6, 7 e 13; `docs/product.md`, “Experiência de uso” e “Planos, assinatura e trial”; decisão confirmada de incluir a página pública básica nesta entrega.
- **Requisitos de entrega:**
  - A consulta reutiliza a resolução centralizada por slug e retorna nome, slug, logo e apenas os dados opcionais de contato e apresentação preenchidos.
  - Nome, logo e slug utilizável determinam a publicação; estabelecimento incompleto, ausente ou com conflito de rota não tem perfil público e recebe `404`.
  - IDs de usuário, credenciais, marcador operacional, configuração de agenda e dados privados do trial não são expostos.
  - Trial expirado mantém a identificação pública. Esta resposta não oferece disponibilidade nem operações de agendamento.
- **Verificação:** testes unitários de elegibilidade e integração HTTP cobrem publicação após upload, campos ausentes, isolamento por slug, conflitos legados, trial expirado e payload público; regenerar artefatos afetados.
- **Dependências:** subtarefas 1, 3, 4 e 10.

### 13. Incorporar campo multilinha ao design system

- **O que:** criar a primitiva de textarea e seu wrapper de formulário com os estados existentes de `Field`.
- **Motivação:** oferecer edição acessível de descrição e endereço no padrão visual do projeto.
- **Onde:** novos componentes em `frontend/components/ui/`, exportações, testes Vitest e referência em `frontend/app/design-system/`.
- **Base documental:** `AGENTS.md`, regra universal do design system; escopo 4; `docs/product.md`, “Experiência de uso”.
- **Requisitos de entrega:**
  - O campo suporta rótulo, ajuda, erro, valor, limite e estado desabilitado, com associação acessível entre controle e mensagens.
  - Foco, contraste, tamanhos e temas reutilizam os tokens e estilos compartilhados; a referência apresenta os estados introduzidos.
- **Verificação:** Vitest protege associação de mensagens, edição e recusa quando desabilitado; conferir a referência nos temas claro e escuro e executar as checagens do frontend.

### 14. Incorporar seleção de imagem ao design system

- **O que:** criar um controle acessível de arquivo adequado ao upload de logo.
- **Motivação:** apresentar seleção, restrições e falhas de upload com os mesmos padrões dos demais formulários.
- **Onde:** novo componente em `frontend/components/ui/`, exportações, testes Vitest e referência em `frontend/app/design-system/`.
- **Base documental:** regra universal do design system em `AGENTS.md`; escopos 4 e 8; limites de imagem confirmados nesta conversa.
- **Requisitos de entrega:**
  - O controle permite selecionar PNG, JPEG ou WebP por teclado e expõe rótulo, ajuda, erro e estado desabilitado.
  - As restrições informadas usam 2 MiB e 4.096 × 4.096 pixels; a interface deixa a decisão final de aceitação ao backend.
  - Estados visuais novos entram na referência React no mesmo commit, usando tokens existentes.
- **Verificação:** Vitest cobre seleção, erro acessível e desabilitação; revisão visual da referência nos dois temas e checagens do frontend.

### 15. Editar o perfil e os limites no painel

- **O que:** criar `/painel/configuracoes` com consulta e formulário para os dados públicos e limites de agenda.
- **Motivação:** permitir ao proprietário concluir e manter os dados já iniciados no cadastro.
- **Onde:** nova página em `frontend/app/(session)/painel/configuracoes/`, componentes em diretório previsto `frontend/components/establishment/`, `frontend/lib/establishment/`, testes Vitest, `docs/product.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** escopo 7; decisões confirmadas de campos e limites; `docs/architecture.md`, “Frontend” e “Estado”.
- **Requisitos de entrega:**
  - A página usa `SessionGuard` de proprietário e `OwnerPanelShell`, carrega valores persistidos e mostra slug como somente leitura, com fuso editável conforme a resposta do backend.
  - Descrição, endereço, telefone, até cinco links e os três limites podem ser editados e limpos; campos de minutos/dias mostram suas unidades e vazio representa ausência de limite.
  - O cliente tipado envia PATCH com CSRF, representa erros por campo e falhas temporárias sem mensagens brutas; submissões simultâneas e respostas obsoletas são controladas.
  - Trial expirado mantém a consulta e explica a indisponibilidade de escritas; conflito de slug legado aparece com orientação compatível com a resolução operacional aprovada.
  - Navegação por teclado, foco e apresentação responsiva seguem o design system; o checklist só marca os comportamentos efetivamente entregues.
- **Verificação:** Vitest cobre carregamento, edição e limpeza, validação, falha de consulta com nova tentativa, erro por campo, bloqueio de fuso, `read_only` e descarte de respostas antigas; executar checagens do frontend.
- **Dependências:** subtarefas 8, 9 e 13.

### 16. Concluir a configuração com envio e troca de logo

- **O que:** adicionar à página de configurações o fluxo de seleção, envio e substituição do logo.
- **Motivação:** tornar acessível ao proprietário o requisito que habilita a publicação.
- **Onde:** `frontend/components/establishment/` e `/painel/configuracoes`, `frontend/lib/establishment/`, testes Vitest, `docs/product.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** escopos 7 e 8; decisões confirmadas sobre publicação automática ao completar os obrigatórios e substituição sem remoção.
- **Requisitos de entrega:**
  - O cliente reutiliza `apiFetch` com `FormData` e CSRF, deixando o navegador definir o boundary multipart.
  - A página informa o logo ausente, permite pré-visualização e envio, apresenta recusas do backend e mantém o logo anterior diante de falha.
  - Sucesso atualiza a consulta do estabelecimento e a imagem, limpa a seleção temporária e informa o endereço público quando disponível; URLs temporárias de pré-visualização são liberadas.
  - Não existe ação de remover logo publicado; trial expirado mantém a visualização e bloqueia envio e substituição.
- **Verificação:** Vitest cobre multipart/CSRF no cliente, falha e nova tentativa, substituição, limpeza de seleção e `read_only`; validação manual local do upload e do endereço público.
- **Dependências:** subtarefas 10, 11, 14 e 15.

### 17. Direcionar o proprietário à configuração pendente

- **O que:** centralizar a entrada do proprietário em `/painel`, resolver seu destino pelo preenchimento do logo e incluir Configurações na navegação.
- **Motivação:** orientar a conclusão após confirmação ou login sem impedir o uso do painel.
- **Onde:** nova página em `frontend/app/(session)/painel/`, `frontend/components/session/owner-panel-shell/`, `frontend/components/session/session-guard/`, `frontend/components/auth/login-form/`, `frontend/components/auth/email-verification/`, testes Vitest, `docs/product.md`, `docs/operations.md` e `docs/implementation-scope.md`.
- **Base documental:** decisão confirmada que substitui o redirecionamento atual para `/painel/profissionais`, descrito em `docs/product.md`, “Cadastro, usuários e permissões” e “Experiência de uso”, e em `docs/operations.md`, “Validação local do onboarding pela interface”.
- **Requisitos de entrega:**
  - Login, confirmação e redirecionamento de proprietário já autenticado usam a mesma entrada: sem logo seguem a Configurações; com logo seguem a Profissionais.
  - A ausência de logo não impede abrir diretamente Profissionais ou usar a navegação; erros ao consultar configurações têm estado de falha e nova tentativa, sem declarar a sessão ausente.
  - Profissionais continuam em `/testes` e mantêm a troca obrigatória; a política de acesso por papel não é ampliada.
  - A navegação oferece Configurações e identifica corretamente a página ativa, sem manter `aria-current` fixo em Profissionais.
- **Verificação:** Vitest percorre destinos com/sem logo, confirmação, falha temporária, acesso direto ao painel e restrição de papel; validar manualmente foco e navegação.
- **Dependências:** subtarefas 8, 15 e 16.

### 18. Publicar a página informativa do estabelecimento

- **O que:** criar a página canônica server-side `/{slug}` consumindo diretamente a API Go.
- **Motivação:** concluir o onboarding com um endereço público utilizável e dados reais do estabelecimento.
- **Onde:** nova rota prevista `frontend/app/[establishmentSlug]/page.tsx`, cliente server-side previsto em `frontend/lib/establishment/`, configuração server-side e `compose.yaml` quando necessário, testes Vitest de decisões de apresentação, `docs/product.md`, `docs/architecture.md`, `docs/operations.md`, `README.md` e `docs/implementation-scope.md`.
- **Base documental:** escopos 7 e 13; `docs/architecture.md`, “Multi-tenancy e página pública” e “Frontend”; decisão confirmada de incluir página pública básica nesta entrega.
- **Requisitos de entrega:**
  - O Server Component consulta o backend por `API_INTERNAL_URL`, sem BFF, Route Handler ou Server Action. A origem canônica vem de configuração server-side validada, reutilizando `PUBLIC_APP_URL` e documentando seu fornecimento ao frontend.
  - Nome, logo e metadados canônicos são renderizados no servidor; descrição, endereço, telefone e links só aparecem quando preenchidos, com alt do logo baseado no nome.
  - A leitura evita cache que perpetue dados substituídos ou página ainda incompleta. `404` da API resulta em página ausente; falhas temporárias não são transformadas em estabelecimento inexistente.
  - A página é mobile-first, segue o design system e conserva a identificação pública após expiração do trial. Nenhum controle de disponibilidade ou agendamento é apresentado nesta etapa.
  - Rotas estáticas da plataforma continuam acessíveis; configurações e clientes server-side não são importados pelo código enviado ao navegador.
  - Produto, operação e checklist descrevem exatamente o fluxo entregue: cadastro → confirmação → configurações quando falta logo → publicação; itens de agendamento permanecem pendentes.
- **Verificação:** Vitest protege decisões de apresentação e tratamento das respostas; checagem de tipos e build de produção verificam a rota. Validação manual local percorre publicação, dados opcionais, substituição do logo, trial expirado e acesso às rotas existentes, sem adicionar E2E nem integração de frontend.
- **Dependências:** subtarefas 11, 12 e 16.

## Verificação conjunta

- Executar `make check`, `make build` e `make compose-config`; as tarefas cobrem geração, `gofmt`, `go vet -tags=integration ./...`, testes unitários, integração HTTP do backend com Testcontainers/PostgreSQL real, ESLint, Prettier e tipos. Integrações de arquivos usam diretório temporário real; apenas dependências externas, como e-mail, são substituídas.
- Revisar o diff completo contra OpenAPI, migrations, testes e documentos, verificando papel, tenant, CSRF, acesso expirado, concorrência na troca do logo e fuso, legados com conflito e ausência de dados sensíveis em logs e erros.
- Conferir que escrita, leitura e compensações passam pela interface de storage, que o banco usa chaves portáveis e que a documentação distingue o adaptador local entregue do futuro adaptador S3. A troca futura preserva URLs públicas e regras; a migração dos objetos exige cópia e validação antes do corte.
- Percorrer o fluxo local no Compose, incluindo reinício sem perda do logo. Registrar verificações impedidas e limitações reais de recuperação de arquivos; preservar os volumes existentes.
- Atualizar os checkboxes nas subtarefas que concluem cada comportamento. Nos itens mistos, separar a configuração entregue da aplicação futura das regras: integrar o bloqueio de fuso aos primeiros dados operacionais e aplicar limites ao agendamento continuam pendentes.
- Este plano conclui o onboarding e a página informativa, sem declarar toda a Fase 3 concluída: seeds de categorias, catálogo de planos/preços e ciclo comercial permanecem nos respectivos escopos financeiros e de assinatura.
