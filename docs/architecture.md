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

### Proprietário e profissional

- Proprietário confirma e-mail e recupera senha por link enviado por e-mail.
- Tokens de confirmação e recuperação são aleatórios, expiram, são de uso único e não ficam armazenados de forma utilizável após vazamento do banco.
- Respostas de recuperação não revelam se o e-mail existe; aplique rate limiting.
- Profissional não possui recuperação autônoma. O proprietário define uma senha temporária e a troca é obrigatória no próximo login.
- Ao reativar um profissional, o proprietário deve definir uma nova senha temporária; nunca restaure a validade da senha ou das sessões anteriores.
- Até a troca, o profissional só pode definir uma nova senha ou encerrar a sessão.
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

- Use Next.js App Router, React, TypeScript estrito e npm, com `package-lock.json` versionado.
- Não adicione lockfiles de pnpm, Yarn ou Bun.
- Use ESLint e Prettier e forneça scripts `lint`, `format`, `format:check`, `typecheck`, `test` e `build`.
- Crie componentes e estilos próprios com CSS ou SCSS.
- Não use Tailwind CSS, shadcn/ui ou kits equivalentes.
- Reutilize componentes, variáveis e estilos existentes antes de criar novos padrões.

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
