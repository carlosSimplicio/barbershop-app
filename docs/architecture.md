# Arquitetura

## Monorepo e responsabilidades

- Use `frontend/` e `backend/` como aplicações separadas no monorepo.
- O frontend consome a API REST sem depender de detalhes internos do backend.
- O backend concentra regras, validação, autorização e persistência.
- Mudanças de endpoint devem manter OpenAPI, implementação Go e consumo TypeScript alinhados.

## API contract-first

- A API usa o prefixo `/api/v1`.
- `backend/openapi/openapi.yaml` é a fonte da verdade.
- Altere primeiro o OpenAPI, regenere os artefatos e depois ajuste implementação e consumidores.
- Use `oapi-codegen` para modelos, interfaces e boilerplate Gin.
- Use `openapi-typescript` para os tipos do frontend.
- Mantenha um wrapper pequeno sobre `fetch`; não adicione outro cliente HTTP sem necessidade demonstrada.
- Código gerado não é editado manualmente. A automação deve detectar artefatos desatualizados.
- Geração não substitui validação em runtime, autenticação, autorização ou regras de negócio.

### Erros

- Use um formato único com código estável, mensagem em `pt-BR` e `request_id`.
- Erros de validação incluem detalhes estruturados por campo.
- O frontend decide pelo código, não pelo texto traduzido.
- Use status HTTP coerentes e documente respostas no OpenAPI.
- Não exponha stack traces, SQL, segredos, nomes internos ou infraestrutura; detalhes técnicos ficam nos logs correlacionados.

## Backend

- Use Go, Gin, PostgreSQL, `pgx/v5`, `sqlc` e Goose.
- Organize por Vertical Slice Architecture, agrupando cada capacidade ou caso de uso.
- Um slice pode conter handler, entrada/saída, validação, caso de uso e persistência necessários ao fluxo.
- Evite grandes diretórios horizontais globais de handlers, services e repositories.
- Mantenha regras testáveis sem Gin, `pgx`, rede ou banco reais.
- Gin fica na borda HTTP; `pgx` e `sqlc`, na borda de persistência.
- Compartilhe apenas abstrações de domínio estáveis ou necessidades transversais reais; evite utilitários genéricos.
- Não adote CQRS formal, command/query bus, mediator ou event bus sem necessidade concreta.

### Estratégia de testes

- Mantenha testes unitários rápidos para regras de negócio, validações e recusas que possam ser verificadas sem infraestrutura real.
- Cubra os fluxos entregues do backend também com testes de integração black-box na fronteira HTTP, atravessando roteamento Gin, middleware, serviços, código `sqlc`, transações e constraints do PostgreSQL.
- Execute a integração em PostgreSQL descartável provisionado por Testcontainers, usando a mesma versão fixada para o projeto e aplicando as migrations Goose reais. Cada teste deve começar com dados isolados e determinísticos.
- Use `httptest` com TLS para preservar o comportamento dos cookies `Secure`. Substitua somente dependências externas ao backend, como o transporte de e-mail, por fakes locais controlados; não substitua banco, stores ou casos de uso nos testes de integração.
- Mantenha a suíte de integração sob a build tag `integration`. Testes unitários não exigem Docker; a suíte de integração e o gate de pull request exigem Docker disponível.
- Não adote testes E2E nem integração de frontend sem nova decisão. Evite duplicar nos testes de integração todas as combinações já protegidas unitariamente; priorize jornadas reais, contratos entre camadas, persistência, autorização e invariantes transacionais.

## Banco de dados

- Escreva SQL nos arquivos de entrada do `sqlc` e regenere o código Go após alterações.
- Não introduza ORM ou outro driver PostgreSQL em paralelo.
- Modele a oferta de serviços como uma associação explícita entre profissional e serviço, limitada ao mesmo estabelecimento.
- Mantenha preço e duração no serviço, não na associação com o profissional; somente a comissão pode possuir uma regra específica para esse par.
- Persista snapshots de preço e duração no agendamento; alterações posteriores no catálogo não recalculam agendamentos existentes.
- Valide a associação ativa ao consultar disponibilidade e novamente ao criar, reagendar ou registrar um atendimento sem agendamento; não confie apenas nas opções exibidas pelo frontend.
- Use UUIDs aleatórios e não sequenciais; gere-os com fonte segura e trate-os como opacos.
- UUID não substitui autorização ou isolamento de tenant.
- Toda alteração de schema usa migration Goose nova, pequena e determinística.
- Não modifique migrations que possam ter sido executadas em outro ambiente.
- Use transações quando uma operação precisar preservar invariantes entre múltiplas escritas.
- Use constraints do PostgreSQL para integridade estrutural, como nulabilidade, referências, unicidade, cardinalidade, isolamento relacional e ordenação temporal genérica. Regras de negócio que podem evoluir — formatos e palavras reservadas, limites de produto, papéis permitidos, durações exatas e transições ou imutabilidade de domínio — pertencem ao backend Go e aos seus testes, não a `CHECK` constraints ou triggers.

## Arquivos públicos

- Armazene logos em um volume persistente local da VPS no MVP.
- O backend controla upload, validação e associação do arquivo ao estabelecimento.
- Aceite somente PNG, JPEG e WebP com tamanho máximo de 2 MB; rejeite SVG no MVP.
- Verifique assinatura e decodificação do conteúdo real do arquivo, não apenas extensão ou `Content-Type` informado pelo cliente.
- Gere nomes internos não previsíveis; nunca use diretamente o nome enviado pelo usuário como caminho.
- Armazene no PostgreSQL somente metadados e uma referência relativa estável, não caminhos absolutos da VPS.
- Não permita que caminhos fornecidos pelo cliente determinem onde um arquivo será lido ou gravado.

## Multi-tenancy e página pública

- No MVP, a URL canônica é `/{establishment_slug}` no domínio principal.
- O slug é único e fica armazenado no estabelecimento; não armazene a URL completa.
- O proprietário informa o slug durante o cadastro. Valide formato, unicidade e palavras reservadas no backend antes de criar o estabelecimento.
- Aceite somente letras minúsculas ASCII, números e hífens, com 3 a 63 caracteres e sem hífen no início ou no fim.
- Valide o valor informado em vez de transformá-lo silenciosamente; o frontend deve explicar o formato antes do envio.
- Trate o slug como imutável no MVP e não ofereça endpoint ou ação de interface para alterá-lo.
- Reserve ao menos `admin`, `api`, `login` e `assets`.
- Centralize a resolução do estabelecimento; domínio e consultas não dependem diretamente do formato da URL.
- Prepare o resolvedor para futuramente aceitar `{establishment_slug}.barbershop-app.com`.
- Não publique caminho e subdomínio simultaneamente como canônicos. Em migração futura, redirecione permanentemente os caminhos antigos.
- Toda consulta autenticada ou pública inclui explicitamente o estabelecimento resolvido.
- `GET /establishment` deriva o estabelecimento exclusivamente do `Principal` autenticado e não aceita identificador de tenant como autoridade em path, query ou corpo. A resposta limita-se a identificador, nome, slug, fuso IANA e trial; seus limites são instantes UTC, enquanto o fuso persistido continua sendo a autoridade para regras e apresentação locais.

## Autenticação

- Proprietários e profissionais usam e-mail e senha; a API usa JWT.
- O e-mail de login é único em toda a plataforma, comparado de forma normalizada e sem diferenciar maiúsculas de minúsculas; não use o estabelecimento para desambiguar contas.
- O e-mail de login é imutável no MVP; não ofereça endpoint ou ação de interface para alterá-lo.
- Nunca armazene senha em texto puro nem registre senhas ou tokens.
- Senhas devem ter entre 8 e 32 caracteres e não podem conter espaços ou outros caracteres de whitespace.
- Não exija combinações obrigatórias de maiúsculas, minúsculas, números ou símbolos e não imponha troca periódica.
- Não consulte listas de senhas comuns ou credenciais comprometidas no MVP.
- Armazene senhas com Argon2id, salt único por senha e parâmetros centralizados e ajustáveis; nunca use hash rápido ou criptografia reversível.
- Aplique limitação progressiva às tentativas de login usando, no mínimo, a combinação de IP e e-mail normalizado.
- Não bloqueie a conta permanentemente. Use janelas temporárias, responda de forma genérica e não revele se o e-mail está cadastrado.
- Retorne `429` e `Retry-After` quando aplicável e estenda a mesma proteção aos fluxos de confirmação e recuperação.
- Não implemente autenticação em dois fatores no MVP; preserve o modelo de autenticação de forma que ela possa ser adicionada posteriormente sem alterar as regras de domínio.
- Use access JWT de 15 minutos e refresh token rotativo com prazo deslizante de 7 dias de inatividade.
- Transporte-os em cookies `HttpOnly`, `Secure` e `SameSite`, com `Domain` e `Path` mínimos.
- Não use `localStorage` ou `sessionStorage` para tokens.
- Refresh tokens são revogáveis e protegidos no backend; cada uso invalida o anterior e reutilização deve ser detectada.
- Proteja mutações contra CSRF e valide origem e autorização.
- JWT contém apenas claims necessárias: usuário, estabelecimento, papel, expiração e versão de credencial.
- Reset de senha invalida credenciais previamente emitidas.
- Desativar um usuário revoga seus refresh tokens, incrementa sua versão de credencial e faz o backend rejeitar imediatamente access tokens ainda não expirados.

### Contrato de sessão

- A API expõe `POST /auth/register`, `POST /auth/email-confirmation`, `POST /auth/email-confirmation/verify`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `POST /auth/password/change` e `GET /establishment` sob `/api/v1`.
- `POST /auth/register` é público, mas mantém a allowlist estrita de `Origin`. Ele valida no backend nome, slug, fuso IANA, e-mail e senha; cria estabelecimento, usuário `owner`, credencial e primeira confirmação em uma única transação; e não cria trial.
- O cadastro responde `202 Accepted` depois do commit. O envio da confirmação ocorre diretamente pelo serviço interno após a transação; falha de transporte é registrada apenas como evento operacional, sem e-mail, token ou link, e não transforma o cadastro persistido em erro HTTP.
- Conflitos de e-mail global e slug único retornam `409` com erro associado ao campo correspondente. A limitação progressiva do cadastro usa escopo próprio e chaves HMAC separadas para IP e e-mail normalizado, sem persistir esses identificadores em claro.
- A reserva dos eventos de rate limiting pertence à infraestrutura compartilhada em `internal/platform/ratelimit`; autenticação e onboarding recebem esse store explicitamente e não reutilizam stores de outro slice. O store serializa concorrentemente cada chave HMAC com advisory locks transacionais, verifica todas as dimensões e grava seus eventos na mesma transação, impedindo que requisições simultâneas ultrapassem o limite. No login, a reserva é removida quando a tentativa não representa credencial inválida, preservando a contagem exclusiva de falhas.
- `POST /auth/email-confirmation` é público, mantém validação estrita de `Origin` e recebe o e-mail somente no corpo. Para qualquer e-mail válido, conta inexistente, owner já confirmado e nova confirmação emitida recebem o mesmo `202 Accepted` sem corpo.
- Quando há owner ativo ainda não confirmado, a substituição bloqueia e revalida sua elegibilidade, invalida tokens ativos e persiste um novo token de 24 horas antes do envio. Mudança concorrente de elegibilidade encerra a solicitação sem novo token. Falha de transporte após a persistência não altera o `202` e gera somente evento operacional sem e-mail ou token.
- A mensagem usa `/confirmar-email#token=...`: o fragmento não participa da requisição HTTP. A interface extrai o token, remove o fragmento e a query do histórico antes de qualquer navegação ou chamada de API e o envia exclusivamente no corpo de `POST /auth/email-confirmation/verify`. A verificação de uso único é iniciada uma vez mesmo sob Strict Mode; token ausente ou resposta inválida/expirada recebe mensagem própria, sem expor o valor.
- O rate limiting da solicitação de confirmação usa escopo próprio, IP e e-mail normalizado e é registrado antes de consultar a conta, preservando o mesmo tratamento de abuso independentemente de sua existência.
- A proteção anti-enumeração atual uniformiza status e corpo, mas não garante tempo constante: o SMTP síncrono pode tornar uma solicitação elegível mais lenta. Esse risco é aceito nesta etapa; não há outbox nem sessão restrita antes da confirmação, e eventual endurecimento de latência exige uma decisão arquitetural posterior.
- `POST /auth/email-confirmation/verify` é público, exige `Origin` confiável e recebe o token somente no corpo. A transação bloqueia token e owner, valida expiração e estado na aplicação, confirma o e-mail, consome o token, cria o único trial exato de 14 dias e persiste sessão e refresh token. Replay, token substituído, expirado ou inexistente produzem o mesmo erro; access JWT e CSRF são preparados antes do commit, de modo que uma falha de emissão reverta a transação, e então são enviados nos cookies seguros existentes.
- Confirmações persistem apenas o hash SHA-256 do token aleatório de 256 bits, sua expiração e estados de consumo ou invalidação. Trials ficam separados de assinaturas e preservam proprietário, estabelecimento e a janela imutável `[confirmed_at, confirmed_at + 14×24h)` com unicidade para ambos.
- Access JWTs usam Ed25519/EdDSA, incluem `kid` no header e são validados por um keyring de chaves públicas. A chave privada ativa e as chaves públicas anteriores são configuração exclusiva do backend.
- As claims são limitadas a usuário, estabelecimento, papel, expiração e versão da credencial. Em cada requisição autenticada, o backend também consulta o usuário atual e compara estado, tenant, papel e versão para permitir revogação imediata.
- A configuração inicial do Argon2id usa `m=19456 KiB`, `t=2`, `p=1`, salt aleatório de 16 bytes e saída de 32 bytes em formato PHC. Os limites configuráveis não podem ficar abaixo desse perfil.
- Os cookies são host-only e `SameSite=Strict`: `__Secure-barbershop_access` usa `Path=/api/v1`, `__Secure-barbershop_refresh` usa `Path=/api/v1/auth` e ambos são `HttpOnly`; `__Secure-barbershop_csrf` usa `Path=/` e pode ser lido pelo frontend somente para envio no header `X-CSRF-Token`. Todos usam `Secure`, inclusive no desenvolvimento local.
- Mutações validam uma allowlist exata de `Origin`. Refresh, logout e troca de senha também exigem igualdade entre cookie e header CSRF e assinatura válida do token.
- O rate limiting de login usa chaves HMAC separadas para IP e e-mail normalizado, sem persistir esses valores na tabela de tentativas. Depois de 5 falhas em 15 minutos, 10 em 1 hora ou 20 em 24 horas, novas tentativas recebem `429` e `Retry-After` pela maior janela ativa.
- Várias sessões por usuário são permitidas, sem prazo absoluto. Cada refresh estende a inatividade por 7 dias, consome definitivamente o token anterior e revoga a família da sessão se houver reutilização.
- `client.ts` é a entrada pequena do transporte HTTP, `transport.ts` envia requisições e trata CSRF e respostas, e `session.ts` recupera somente leituras autenticadas (`GET`/`HEAD`) após `401`. Uma promessa compartilhada evita duas renovações simultâneas na aba. O Web Lock `barbershop-auth-refresh` serializa a renovação entre abas; dentro dele, `GET /auth/me` verifica se outra aba já atualizou os cookies. Quando a rechecagem retorna `200`, somente `/auth/me` é repetido automaticamente, para descobrir a identidade atual sem entregar dados de outra conta a um consumidor antigo. Sem cookie CSRF, o refresh exigido pelo backend não pode funcionar: após a rechecagem, o frontend não o envia e mantém o `401` como sessão ausente, permitindo as páginas públicas. Após refresh `204`, a leitura original é repetida uma única vez. `401` do refresh ou do retry notifica sessão inválida; resposta indeterminada (`5xx`/rede) impede outra tentativa automática na mesma aba e é apresentada como indisponibilidade, não como logout. Sem Web Locks, resta a deduplicação da aba atual. Nenhum token ou estado é persistido em Web Storage. Limites conhecidos: outra aba pode renovar após falha indeterminada, e a versão atual do backend pode já ter consumido o token antes de responder `500`; a correção definitiva exige uma mudança transacional no backend ou sinal entre abas.
- Nas rotas comuns de sessão, o `AuthProvider` fica no layout do grupo `(session)`, preservando a home estática sem consulta de autenticação. `/confirmar-email` é uma rota de bootstrap fora desse grupo: só monta uma instância do mesmo Provider depois de remover o fragmento e a query da URL, impedindo que a consulta inicial de `/auth/me` preceda a limpeza. O Provider consulta `/auth/me`, mantém somente identidade e estado transversal, distingue sessão ausente de erro temporário e expõe recarga e uma ação explícita de transição para login, confirmação, troca de senha, refresh manual e logout. Quando a ação retorna `AuthPrincipal`, ele aplica diretamente essa identidade sem uma segunda chamada a `/auth/me`; ações sem principal continuam reconsultando a sessão. A transição aguarda qualquer refresh pendente, bloqueia nova recuperação automática durante a ação e usa o mesmo Web Lock quando disponível; só uma nova sessão bem-sucedida reabilita a recuperação. Desmontar o Provider não reinicia esse estado; uma leitura de domínio antiga não é repetida após troca explícita de sessão, e um retry que termina depois da troca é descartado. As guardas reutilizáveis ocultam conteúdo incompatível durante a descoberta da sessão e usam substituição de histórico para `/login` ou `/testes`. `/cadastro` usa o layout e a guarda de visitantes; `/confirmar-email` sincroniza a nova sessão antes de substituir a rota por `/testes`.
- Um profissional com senha temporária pode consultar e manter sua sessão, trocar a senha ou sair, mas não acessar recursos de domínio. A troca rejeita a mesma senha, revoga todas as sessões, incrementa a versão da credencial e reemite somente a sessão atual.

### Retenção dos dados de autenticação

- Access JWTs e tokens CSRF são stateless e não geram registros persistidos. Cada login cria uma linha em `auth_sessions` e uma em `refresh_tokens`; cada renovação acrescenta outro refresh token à mesma sessão; cada cadastro ou substituição elegível cria uma linha em `email_confirmations`; e cada falha de login, tentativa válida de cadastro e solicitação válida de confirmação cria eventos separados por dimensão em `auth_login_attempts`, com escopos independentes.
- A implementação atual não executa limpeza de sessões, refresh tokens, confirmações de e-mail ou eventos de rate limiting. A query de exclusão de eventos antigos existe, mas ainda não está ligada a uma rotina operacional. Portanto, essas quatro tabelas têm crescimento ilimitado até a implementação da retenção e não devem ser consideradas prontas para operação prolongada em produção.
- Antes da produção, defina e documente a janela durante a qual a reutilização de um refresh token consumido ainda revoga sua família. Depois que o token estiver expirado e ultrapassar essa margem, remover seu hash pode transformar uma reutilização tardia em sessão inválida genérica, mas nunca pode tornar o token utilizável novamente.
- A retenção deve remover eventos de rate limiting de todos os escopos depois da maior janela aplicável, atualmente 24 horas, acrescida de margem operacional; remover confirmações consumidas, invalidadas ou expiradas somente depois da janela operacional definida; remover refresh tokens consumidos ou revogados somente depois de expirados e da margem de detecção; e remover sessões expiradas ou revogadas somente depois de tratar seus tokens relacionados e cumprir a retenção operacional definida.
- Execute a limpeza em lotes limitados, com índices apropriados para os predicados de expiração e um único executor por vez. A exclusão precisa respeitar as chaves estrangeiras: primeiro refresh tokens elegíveis e depois sessões sem tokens retidos.
- Limite também a frequência de renovação por sessão. Sem esse controle, um cliente autenticado com defeito ou malicioso pode rotacionar tokens em sequência e acelerar o crescimento mesmo com a limpeza periódica.
- Monitore quantidade de linhas, tamanho total e idade do registro elegível mais antigo em `auth_sessions`, `refresh_tokens`, `email_confirmations` e `auth_login_attempts`. Confie no autovacuum para tornar espaço apagado reutilizável; não use `VACUUM FULL` como manutenção rotineira.

### Proprietário e profissional

- Proprietário confirma e-mail e recupera senha por link enviado por e-mail.
- Tokens de confirmação e recuperação são aleatórios, expiram, são de uso único e não ficam armazenados de forma utilizável após vazamento do banco.
- Respostas de recuperação não revelam se o e-mail existe; aplique rate limiting.
- Profissional não possui recuperação autônoma. O proprietário define uma senha temporária e a troca é obrigatória no próximo login.
- Ao reativar um profissional, o proprietário deve definir uma nova senha temporária; nunca restaure a validade da senha ou das sessões anteriores.
- Até a troca, o profissional só pode consultar ou manter a sessão, definir uma nova senha ou encerrar a sessão; recursos de domínio permanecem bloqueados.
- A senha temporária nunca é recuperável ou armazenada em texto puro.

### E-mail transacional

- O provedor ainda não foi definido.
- Isole envio atrás de uma interface pequena; não adicione SDK de SMTP, SES, Resend ou outro provedor antes da decisão.
- Em desenvolvimento local, use SMTP com Mailpit para inspecionar mensagens de confirmação e recuperação sem enviar e-mails reais.
- O Mailpit é uma dependência exclusiva de desenvolvimento e nunca é fallback para produção.
- Em produção, falhe de forma segura se o envio de e-mail necessário estiver habilitado sem um provedor real configurado.
- Não registre endereço completo, token, link de recuperação ou credenciais ao tratar falhas.
- Templates de confirmação e recuperação ficam versionados e em `pt-BR`.

## Assinaturas da plataforma

- Modele assinatura e trial separadamente dos recebimentos dos atendimentos do estabelecimento.
- O cadastro não depende de uma assinatura ativa.
- Persista a utilização do trial separadamente da assinatura atual; nunca derive elegibilidade apenas da ausência de uma assinatura ativa.
- O MVP possui um plano comercial, mas não codifique a suposição de plano único em tabelas, regras ou contratos.
- Separe o conceito de plano das opções de preço e período de cobrança, permitindo vários preços ou durações para o mesmo conjunto de funcionalidades.
- Cadastre o preço mensal inicial como `BRL 20,00` no catálogo persistido ou em dados de inicialização versionados; não o espalhe como constante no código.
- Crie plano e preço iniciais por migration/seed idempotente e versionado.
- Não implemente perfil ou painel de administrador da plataforma no MVP; mudanças de catálogo são alterações controladas de seed/migration.
- Versione preços e preserve na assinatura a opção contratada, para que reajustes futuros não alterem silenciosamente contratos existentes.
- Modele adicionais como itens opcionais da assinatura, com preço e direitos próprios, sem adicionar colunas específicas para cada produto futuro.
- Centralize a resolução de direitos de acesso; não espalhe comparações de nome de plano ou adicional pela aplicação.
- Preserve em cada assinatura os identificadores e termos necessários para interpretar historicamente o que foi contratado.
- Não acople o domínio a um gateway específico enquanto o provedor de cobrança não estiver decidido.
- Defina uma interface de provedor de cobrança e implemente inicialmente um adaptador mockado.
- O mock não movimenta dinheiro nem coleta dados reais de pagamento e deve ser identificado claramente na interface e nos logs.
- Habilite o mock somente por configuração explícita nos ambientes autorizados; falhe de forma segura se produção exigir um provedor real ainda não configurado.
- Faça o mock simular de forma determinística os estados necessários para desenvolver assinatura, renovação, falha e cancelamento.
- A ação mockada de assinatura é idempotente: repetição da mesma solicitação não cria assinaturas ou períodos duplicados.
- Ao aprovar, registre início e fim do período contratado e conceda os direitos do plano pelo domínio de assinaturas, não diretamente pelo handler.
- A renovação mockada é manual e idempotente. Período ativo é estendido a partir do fim atual; assinatura expirada cria novo período a partir da reativação.
- Modele cancelamento ao fim do período separadamente de expiração imediata; direitos permanecem ativos até `current_period_end`.
- Centralize a política de acesso por assinatura: trial ou período ativo concede escrita; expiração preserva leitura e bloqueia mutações, exceto autenticação e assinatura/renovação.
- Aplique a política no backend e não apenas na interface. Todos os endpoints públicos de mutação da agenda devem respeitar o direito de acesso do estabelecimento.
- Expiração não altera estados nem remove agendamentos; apenas bloqueia suas mutações até a reativação.
- Estados, webhooks, idempotência e acesso por plano devem ser definidos antes de implementar a integração de cobrança.

## Frontend

- A home deslogada em `/` é uma landing page estática, renderizada no servidor e sem chamadas à API. `/cadastro` usa o layout de sessão e a guarda de visitantes, envia `POST /auth/register` pelo cliente tipado e mantém um estado de confirmação pendente após `202`, sem criar sessão ou iniciar o trial. Nesse estado, um componente reutilizável permite solicitar `POST /auth/email-confirmation` para o mesmo e-mail; ele nunca mostra a mensagem bruta do backend e apresenta o mesmo sucesso genérico para qualquer `202`. `/confirmar-email` monta primeiro uma fronteira que lê e elimina o fragmento; somente depois monta o Provider, verifica o token e, após sucesso, substitui a rota por `/testes`. `/login` usa a guarda de visitantes e `POST /auth/login` pelo cliente tipado, trata credenciais inválidas sem detalhes, oferece o mesmo reenvio após `email_confirmation_required` e sincroniza a sessão antes de substituir a rota por `/testes`. `/testes` usa a guarda autenticada, apresenta papel, identificadores e `password_change_required` sem e-mail ou tokens, e permite repetir `GET /auth/me` pelo Provider; falha temporária mostra opção de nova tentativa, e sessão ausente substitui a rota por `/login`. Uma seção separada consulta `GET /establishment` sem identificador de tenant no pedido, apresenta nome, slug, fuso e trial ausente, ativo ou expirado, e fica indisponível durante troca obrigatória de senha. A seção descarta respostas antigas após troca de identidade. As demais ações técnicas permanecem pendentes. Os tipos TypeScript gerados do OpenAPI permanecem sincronizados.

- Use Next.js App Router, React, TypeScript estrito e npm, com `package-lock.json` versionado.
- Não adicione lockfiles de pnpm, Yarn ou Bun.
- Use ESLint e Prettier e forneça scripts `lint`, `format`, `format:check`, `typecheck`, `test` e `build`.
- Crie componentes e estilos próprios com CSS ou SCSS.
- Não use Tailwind CSS, shadcn/ui ou kits equivalentes.
- Trate os tokens em `frontend/styles/tokens.css` e os componentes em `frontend/components/ui` como fontes de verdade da identidade visual e dos padrões de interface. A referência React em `frontend/app/design-system` deve usar esses componentes reais, documentar seus estados e permanecer disponível somente em desenvolvimento; em produção, essa rota responde como não encontrada.
- Mantenha cada componente de `frontend/components/ui` em uma pasta própria, com implementação TypeScript React, teste e CSS Module co-localizados. Exponha a API pública pelo barrel do diretório e use a prop `ref` nativa do React 19, sem `forwardRef`.
- Sirva a tipografia Montserrat a partir de `frontend/public/fonts/montserrat`, junto de sua licença OFL, sem depender de download de fonte durante build ou execução.
- Reutilize componentes, variáveis e estilos existentes antes de criar novos padrões. Não crie variações locais ou componentes isolados que dupliquem uma solução já coberta pelo design system.
- Quando o produto realmente precisar de um componente, variante ou estado ainda inexistente, adicione-o ao design system no mesmo trabalho que o introduzir no frontend. A documentação deve incluir os temas claro e escuro, estados interativos, comportamento responsivo e requisitos de acessibilidade aplicáveis.
- Alterações visuais em componentes existentes devem atualizar simultaneamente o design system e todos os usos afetados, sem divergência conhecida entre documentação e implementação.

### Renderização e API

- Use estratégia híbrida; não force SSR dinâmico ou client-side para tudo.
- A página pública renderiza no servidor conteúdo inicial e metadados, como estabelecimento, serviços e profissionais.
- Agendamento e disponibilidade são Client Components interativos.
- Agenda, administração e financeiro são predominantemente client-side.
- Mantenha layouts e cascas estáticas como Server Components quando possível.
- Coloque `"use client"` na menor fronteira necessária; não marque o layout raiz sem necessidade.
- O navegador chama `/api/v1` diretamente; Caddy encaminha ao backend no mesmo domínio.
- Não crie BFF com Route Handlers nem use Server Actions para regras ou mutações.
- Server Components não chamam Route Handlers internos; uma leitura server-side legítima chama o backend Go diretamente.

### Estado

- Use estado, efeitos e Context nativos do React.
- Prefira estado local e eleve-o apenas ao ancestral necessário.
- Context serve somente a estado transversal, como sessão; não é cache genérico da API.
- Não adicione TanStack Query, Redux, Zustand ou equivalente sem nova decisão.
- O wrapper de `fetch` representa carregamento, sucesso, ausência e erro consistentemente.
- Cancele requisições obsoletas e impeça respostas antigas de sobrescrever estado recente.

## Convenções

- Código, arquivos, pacotes, tipos, variáveis, tabelas, colunas, endpoints, enums técnicos e códigos de erro usam inglês.
- Interface e mensagens ao usuário usam `pt-BR`; traduza valores técnicos na apresentação.
- Comentários técnicos usam inglês. Documentação de produto e operação pode usar português.
- A fundação usa Go `1.27.2`, Gin `1.12.0`, pgx `5.11.0`, sqlc `1.31.1`, Goose `3.29.0` e oapi-codegen `2.8.0` no backend.
- O frontend usa Node.js `24.21.0` LTS, Next.js `16.4.2`, React `19.3.1`, TypeScript `5.9.3` e Vitest `5.0.5`.
- PostgreSQL `18.6`, Caddy `2.11.7` e Mailpit `1.31.5` completam a base operacional. Atualizações são mudanças deliberadas e devem manter manifests, imagens, lockfiles e esta documentação alinhados.
