# Escopo de implementação

Este documento é o inventário do que precisa ser construído para entregar o MVP. As regras detalhadas continuam em `product.md`, `architecture.md` e `operations.md`; em caso de dúvida ou divergência, esses documentos devem ser consultados e alinhados antes da implementação.

## Ordem recomendada de implementação

As fases abaixo orientam a sequência de desenvolvimento. Os checklists detalhados permanecem organizados por domínio nas seções seguintes; uma fase pode usar apenas parte de um escopo e concluir o restante mais adiante.

### Fase 1 — Fundação de engenharia

Escopos principais: 1–4, 24–26.

- Criar monorepo, backend, frontend e convenções compartilhadas.
- Fixar versões, configurar dependências, geração de código e comandos locais.
- Subir PostgreSQL, backend, frontend, Caddy e Mailpit com Docker Compose de desenvolvimento.
- Criar OpenAPI inicial, formato de erros, conexão com PostgreSQL, Goose e `sqlc`.
- Configurar logs básicos, health checks, testes unitários, lint, formato, tipos e CI inicial.

Critério de saída: um checkout limpo sobe localmente, frontend e backend se comunicam por `/api/v1`, uma migration pode ser aplicada e todas as verificações iniciais passam.

### Fase 2 — Identidade, sessões e isolamento

Escopos principais: 5, 6, 21 e 22.

- Implementar usuários, senhas, login, cookies, access JWT e sessões revogáveis.
- Implementar confirmação de e-mail, recuperação do proprietário e senha temporária do profissional.
- Integrar Mailpit no ambiente local.
- Estabelecer papéis, contexto de estabelecimento, isolamento de tenant, CSRF e rate limiting.
- Criar a base da política centralizada de acesso por trial ou assinatura.

Critério de saída: proprietário e profissional autenticam com segurança, sessões expiram após 15 dias e podem ser revogadas, e nenhum usuário acessa dados de outro estabelecimento ou papel.

### Fase 3 — Onboarding e estabelecimento

Escopos principais: 7, 8 e parte do 20.

- Implementar cadastro atômico do proprietário e estabelecimento.
- Implementar confirmação que inicia um único trial de 14 dias.
- Criar slug, fuso, dados públicos, configurações de agenda e upload seguro de logo.
- Criar seeds de categorias, plano e preço inicial.
- Expor situação básica do trial e aplicar seus direitos de acesso.

Critério de saída: um novo proprietário cria e confirma a conta, entra no painel e possui estabelecimento público configurável com trial ativo.

### Fase 4 — Catálogo e disponibilidade interna

Escopos principais: 9, 10 e 12.

- Implementar profissionais, serviços e associação explícita entre eles.
- Implementar comissões padrão e específicas por serviço.
- Implementar jornada semanal, múltiplos períodos e bloqueios pontuais.
- Implementar ativação, desativação, preservação histórica e alertas de impacto futuro.
- Construir e testar o cálculo de disponibilidade e slots de 15 minutos.

Critério de saída: o proprietário consegue preparar todo o catálogo e a jornada, e o backend calcula slots válidos sem dupla ocupação do profissional.

### Fase 5 — Cliente e agendamento público

Escopos principais: 11 e 13, usando o 12.

- Implementar página pública renderizada pelo servidor e fluxo mobile-first de agendamento.
- Implementar clientes por telefone normalizado e regras de atualização de identidade.
- Criar agendamento idempotente com snapshots de preço e duração.
- Implementar consulta pública limitada a agendamentos futuros `scheduled`.
- Implementar cancelamento e reagendamento atômico pelo cliente.
- Aplicar proteção contra enumeração, abuso e concorrência.

Critério de saída: um cliente anônimo agenda, consulta, cancela e reagenda sem conta, respeitando disponibilidade, limites e isolamento do estabelecimento.

### Fase 6 — Agenda operacional

Escopo principal: 14.

- Criar agenda autenticada mobile-first para proprietário e profissional.
- Implementar criação e alterações internas de agendamentos.
- Implementar cancelamento, `no_show` e correções de estado sem efeito financeiro.
- Preparar a ação de conclusão na interface, mas só habilitá-la quando a geração atômica de receita e comissão estiver pronta na fase seguinte.
- Implementar justificativas, histórico e auditoria de estados e horários.
- Exibir alertas para itens inativos e agendamentos pendentes após reativação.

Critério de saída: proprietário e profissionais operam horários, cancelamentos e `no_show` dentro de suas permissões, com estados e histórico consistentes; conclusão permanece indisponível até a fase seguinte.

### Fase 7 — Recebimentos, atendimentos e comissões

Escopos principais: 15–17.

- Registrar Pix, cartão ou dinheiro somente após atendimento com valor positivo.
- Implementar e habilitar a transição para `completed` como parte da mesma operação financeira.
- Gerar receita e comissão atomicamente na conclusão.
- Implementar atendimentos sem agendamento e suas restrições por papel e data.
- Implementar ajustes, anulação, snapshots e auditoria financeira.
- Criar visão do profissional para seus atendimentos e comissões calculadas.

Critério de saída: atendimentos agendados ou avulsos produzem registros financeiros corretos, auditáveis e restritos ao papel apropriado.

### Fase 8 — Despesas e painel financeiro

Escopos principais: 18 e 19.

- Implementar despesas previstas, pagas, canceladas e atrasadas.
- Implementar categorias padrão e personalizadas.
- Implementar edição e cancelamento auditáveis.
- Construir indicadores, tabelas e filtros do proprietário.
- Finalizar visão financeira restrita do profissional.

Critério de saída: os indicadores seguem regime de caixa e produzem os mesmos resultados em cards, tabelas e filtros, sem vazamento entre profissionais ou estabelecimentos.

### Fase 9 — Fluxo comercial da assinatura

Escopo principal: restante do 20.

- Finalizar catálogo, versões de preço, períodos e estrutura para futuros adicionais.
- Implementar gateway mockado, assinatura, renovação manual e cancelamento ao fim do período.
- Implementar telas de contratação, situação, renovação e acesso expirado.
- Validar modo somente leitura, congelamento de agendamentos e reativação.
- Cobrir idempotência e transições da assinatura.

Critério de saída: todo o ciclo mockado de trial, contratação, renovação, cancelamento, expiração e reativação funciona sem perda ou mutação indevida de dados.

### Fase 10 — Preparação e validação de produção

Escopos principais: 23–26 e 28.

- Completar logs estruturados, métricas de duração por etapa, auditoria e rotação.
- Endurecer segurança, configuração, imagens, redes, volumes, Caddy e health checks.
- Finalizar GitHub Actions, releases SemVer, deploy por SSH e migrations seguras.
- Configurar o provedor real de e-mail.
- Executar todas as verificações e validar os critérios de conclusão do MVP.

Critério de saída: uma GitHub Release implanta exatamente o commit marcado na VPS, preserva dados e arquivos e disponibiliza o MVP com documentação consistente.

### Ordem interna de cada vertical slice

Dentro de cada fase, implemente capacidades completas em fatias verticais pequenas:

1. Confirmar a regra nos documentos e atualizar o OpenAPI.
2. Criar migration e SQL necessários.
3. Implementar regra de domínio e testes unitários; adicionar integração quando o fluxo cruzar HTTP, persistência ou PostgreSQL.
4. Implementar persistência e handler Gin.
5. Regenerar backend e tipos TypeScript.
6. Implementar interface e estados do frontend.
7. Executar verificações aplicáveis e atualizar este checklist.

## Uso do checklist

- `[ ]` significa não concluído.
- `[x]` significa implementado, verificado e documentado.
- Marque um item somente quando backend, frontend, contrato, migration e testes aplicáveis estiverem consistentes.
- Ao adicionar, remover ou alterar um requisito, atualize este inventário no mesmo trabalho.
- Itens fora do MVP ficam registrados ao final e não devem ser implementados sem nova decisão.

## 1. Fundação do monorepo

- [x] Criar `backend/` e `frontend/` como aplicações separadas no mesmo repositório.
- [x] Selecionar as versões estáveis atuais das tecnologias e fixá-las explicitamente, sem tags `latest` ou intervalos flutuantes.
- [x] Configurar arquivos ignorados, `.editorconfig` e comandos de desenvolvimento sem incluir segredos; proteger por regressão a exclusão de arquivos de ambiente dos contextos Docker.
- [x] Criar `.env.example` completo e validar toda configuração obrigatória na inicialização.
- [x] Documentar comandos locais para instalar, gerar código, migrar banco, testar, executar e construir o projeto.
- [x] Garantir que código, schema, endpoints e identificadores técnicos usem inglês e que a interface use `pt-BR`.

## 2. Backend e API

- [x] Inicializar o backend em Go com Gin e organização por Vertical Slice Architecture simples.
- [x] Configurar encerramento gracioso, timeouts HTTP, tratamento de panic e health checks.
- [x] Publicar a API REST sob `/api/v1`.
- [x] Criar `backend/openapi/openapi.yaml` como fonte da verdade do contrato.
- [x] Configurar `oapi-codegen` e impedir edição manual dos artefatos gerados.
- [x] Implementar formato uniforme de erro com código estável, mensagem em `pt-BR`, `request_id` e erros estruturados por campo.
- [ ] Implementar validação de entrada, autorização e regras no backend, independentemente das restrições do frontend.
- [ ] Criar middleware para contexto de requisição, estabelecimento, autenticação, autorização, assinatura e correlação de logs.
- [ ] Implementar idempotência nos casos de uso que possam duplicar agendamentos, assinaturas, renovações ou lançamentos.
- [ ] Manter regras de domínio testáveis sem Gin, rede ou PostgreSQL reais.

## 3. Banco de dados e persistência

- [x] Inicializar PostgreSQL com `pgx/v5`, SQL explícito, `sqlc` e Goose, sem ORM.
- [x] Configurar `sqlc` e geração verificável de código Go.
- [ ] Criar migrations pequenas, determinísticas e imutáveis depois de aplicadas.
- [ ] Usar UUIDs aleatórios e não sequenciais para identificadores expostos.
- [ ] Modelar todos os dados pertencentes a estabelecimento com isolamento explícito de tenant.
- [x] Criar persistência base para estabelecimentos, usuários, credenciais, sessões e tokens de autenticação.
- [x] Persistir descrição, endereço, telefone de contato, links sociais ordenados e limites opcionais de agenda por estabelecimento; aplicação das configurações aos fluxos de agenda permanece pendente no escopo 7.
- [x] Consultar perfil, logo e links das configurações no mesmo snapshot SQL, com regressão HTTP sob atualização concorrente.
- [x] Persistir o marcador imutável de início dos dados operacionais e a query que bloqueia a linha do estabelecimento; a alteração de fuso já usa esse bloqueio; a integração atômica com futuros agendamentos, bloqueios e lançamentos permanece pendente no escopo 7.
- [ ] Criar persistência para clientes.
- [x] Criar persistência para serviços, com preço exato em centavos, tenant e queries de catálogo; integração com API e painel consta no escopo 10.
- [x] Criar perfil persistido de profissionais com vínculo ao estabelecimento e queries de criação e listagem, integradas aos fluxos entregues da API.
- [ ] Criar persistência para associações profissional-serviço.
- [ ] Criar persistência para jornadas, períodos de trabalho, bloqueios, agendamentos e histórico de alterações.
- [ ] Criar persistência para atendimentos, receitas, comissões, despesas, categorias e auditoria financeira.
- [ ] Criar persistência para planos, preços, trials, assinaturas, períodos e futuros adicionais.
- [ ] Usar tipos monetários exatos e nunca ponto flutuante binário para valores financeiros.
- [ ] Persistir instantes de forma inequívoca e aplicar o fuso IANA do estabelecimento nas regras locais.
- [ ] Usar transações e proteção contra concorrência em toda operação com invariantes de múltiplas escritas.
- [ ] Criar seed idempotente e versionado para categorias padrão de despesas, plano inicial e preço mensal de `BRL 20,00`.

## 4. Frontend e experiência compartilhada

- [x] Inicializar Next.js App Router com React, TypeScript estrito, npm e `package-lock.json`.
- [x] Configurar ESLint, Prettier, Vitest e scripts `lint`, `format`, `format:check`, `typecheck`, `test` e `build`.
- [x] Configurar `openapi-typescript` e impedir edição manual dos tipos gerados.
- [x] Criar wrapper próprio e tipado sobre `fetch`, com estados consistentes de carregamento, sucesso, vazio e erro.
- [x] Cancelar requisições obsoletas e impedir que respostas antigas substituam estado recente.
- [x] Usar apenas estado, efeitos e Context nativos do React; Context não funciona como cache genérico de API.
- [x] Criar componentes e estilos próprios com CSS ou SCSS, sem Tailwind, shadcn/ui ou kit equivalente.
- [x] Centralizar cores, tipografia, espaçamento, raios, sombras e temas claro e escuro em tokens visuais compartilhados.
- [x] Criar botões configuráveis com estados normal, hover, foco, pressionado e desabilitado.
- [x] Criar primitivas e wrappers configuráveis de formulário com rótulo, ajuda, erro e estados acessíveis, incluindo `Textarea` e `TextareaField` para conteúdo multilinha e `FileField` para seleção de imagens.
- [ ] Criar padrões compartilhados para tabelas.
- [ ] Criar padrões compartilhados para modais.
- [x] Criar padrões compartilhados de feedback.
- [x] Criar padrões compartilhados para estados vazios com lista semântica e anúncio acessível.
- [ ] Fazer todo o frontend seguir o design system e manter nele, no mesmo trabalho, qualquer novo componente, variante ou estado visual necessário ao produto.
- [ ] Garantir navegação por teclado, foco visível, contraste, rótulos acessíveis e alvos de toque adequados.
- [ ] Implementar interface e mensagens em `pt-BR`, valores em `BRL` e datas no fuso do estabelecimento.
- [x] Manter a página pública informativa mobile-first.
- [ ] Manter agenda operacional mobile-first; painel administrativo e financeiro desktop-first e responsivo.
- [x] Usar Server Components para cascas e conteúdo estático e Client Components somente nas fronteiras interativas necessárias.
- [x] Consumir `/api/v1` diretamente; não criar BFF, Server Actions de negócio ou Route Handlers intermediários.
- [x] Criar home deslogada responsiva em `pt-BR`, com proposta da plataforma e links para cadastro e login, sem chamadas à API.

## 5. Autenticação

- [x] Implementar cadastro público do proprietário com criação atômica de usuário, estabelecimento e vínculo de proprietário.
- [x] Oferecer formulário público de cadastro com validação de conveniência, conflitos específicos por campo e estado de confirmação pendente após `202`, sem iniciar o trial na interface.
- [x] Permitir solicitar novo link no estado pós-cadastro e no login após confirmação obrigatória, sempre com resposta genérica.
- [x] Confirmar o e-mail em `/confirmar-email` usando apenas o fragmento da URL e o corpo da API, sincronizar a sessão e direcionar o proprietário pela entrada `/painel` conforme a presença do logo.
- [x] Garantir e-mail globalmente único, normalizado, case-insensitive e imutável no MVP.
- [x] Validar senhas de 8 a 32 pontos de código sem Unicode `White_Space`, controles (`Cc`), formatação (`Cf`) ou `Default_Ignorable_Code_Point`, sem normalização nem regra obrigatória de composição, com regressões equivalentes no backend e no frontend e cobertura HTTP dos cinco fluxos de credenciais.
- [x] Armazenar senhas com Argon2id, salt individual e parâmetros centralizados e ajustáveis.
- [x] Implementar confirmação de e-mail do proprietário com token aleatório, expirável, de uso único e armazenado de forma segura.
- [x] Iniciar o trial somente depois da confirmação do e-mail.
- [x] Implementar login de proprietário e profissional na API e em `/login`, com resposta genérica para credenciais inválidas, reenvio de confirmação e sincronização de sessão.
- [x] Implementar access JWT e sessões com expiração fixa de 15 dias, sem refresh token ou renovação, centralizada com o prazo dos cookies em `auth.SessionLifetime`.
- [x] Transportar access e CSRF em cookies `Secure` e `SameSite=Strict`, com escopo mínimo e `HttpOnly` no access.
- [x] Implementar logout idempotente e revogação da sessão atual identificada no access JWT.
- [x] Tratar qualquer `401` como sessão ausente sem renovação ou repetição automática; não declarar logout em erro temporário e não expor ou persistir tokens no JavaScript.
- [x] Criar o Context único de sessão para rotas autenticadas, com consulta de identidade, distinção entre sessão ausente e erro temporário, guardas reutilizáveis e ações explícitas de login, confirmação, troca de senha e logout.
- [x] Criar `/testes` como área temporária protegida que mostra identidade não sensível e permite consultar novamente `GET /auth/me`, trocar a senha temporária quando exigido e encerrar a sessão com `POST /auth/logout`.
- [x] Direcionar login, confirmação e redirecionamento de proprietário autenticado pela entrada protegida `/painel`, encaminhando a Configurações sem logo e a Profissionais com logo, permitindo acesso direto ao restante do painel; manter profissionais em `/testes` e oferecer navegação por teclado para Profissionais, Configurações e `/testes` com indicação correta da página ativa.
- [ ] Definir janelas de retenção de sessões, confirmações de e-mail, tokens de recuperação de senha e tentativas de login.
- [ ] Implementar limpeza diária, idempotente e em lotes de sessões, confirmações de e-mail, tokens de recuperação de senha e eventos de rate limiting elegíveis, com índices adequados, proteção contra execução concorrente e métricas agregadas.
- [x] Implementar proteção CSRF e validação de origem nas mutações autenticadas.
- [x] Definir `Max-Age` de 15 dias para o cookie CSRF.
- [x] Implementar recuperação de senha por e-mail somente para o proprietário.
- [x] Entregar solicitação e verificação da recuperação de senha do proprietário na API.
- [x] Oferecer entrada pelo login e tela de solicitação de recuperação com resposta genérica.
- [x] Entregar a tela de redefinição e concluir o retorno ao login após sucesso.
- [x] Invalidar credenciais e sessões anteriores após recuperação ou redefinição de senha.
- [x] Permitir que o proprietário cadastre profissional pela API com senha temporária e exigir troca no primeiro login.
- [x] Restringir profissional com senha temporária à manutenção da sessão, troca de senha e logout, bloqueando recursos de domínio.
- [ ] Ao reativar profissional, exigir nova senha temporária sem restaurar credenciais antigas.
- [x] Aplicar limitação progressiva por IP e e-mail normalizado no login, com `429` e `Retry-After`.
- [x] Estender a limitação progressiva à confirmação por IP e e-mail normalizado.
- [x] Estender a limitação progressiva à solicitação de recuperação por IP e e-mail normalizado.
- [ ] Não implementar MFA, troca periódica ou consulta de senha comprometida no MVP.

## 6. Autorização e multi-tenancy

- [x] Implementar papéis `owner` e `professional`; clientes permanecem anônimos.
- [x] Garantir que cada usuário pertença a exatamente um estabelecimento.
- [x] Derivar estabelecimento e permissões da sessão autenticada, nunca de identificador enviado como autorização; resolver o estabelecimento público por slug no serviço central compartilhado pelo perfil e pelo logo.
- [ ] Aplicar isolamento do estabelecimento em toda consulta e mutação, inclusive em SQL e recursos relacionados.
- [ ] Garantir ao proprietário administração completa apenas do próprio estabelecimento.
- [ ] Restringir o profissional à própria agenda, clientes relacionados e visão financeira própria.
- [ ] Impedir profissional de acessar despesas, resultado geral ou dados financeiros de colegas.
- [x] Revogar imediatamente todas as sessões e impedir login quando um profissional for desativado, com serialização contra login e troca obrigatória.
- [x] Criar componente reutilizável que decide o direito de escrita a partir do trial persistido, incluindo o início e excluindo o fim.
- [x] Exigir esse direito de escrita na API de cadastro de profissionais.
- [x] Informar `can_write` na listagem de profissionais pelo componente central, preservando leitura sem trial ativo e distinguindo falhas de infraestrutura.
- [ ] Estender a decisão de escrita aos períodos de assinatura e aplicá-la aos endpoints de mutação autenticados e públicos.
- [ ] Com acesso expirado, permitir autenticação, leitura e assinatura/renovação, bloqueando demais mutações.
- [ ] Cobrir a matriz de autorização e isolamento com testes unitários.

## 7. Cadastro e administração do estabelecimento

- [x] Criar onboarding do proprietário com nome, logo, slug e fuso do estabelecimento.
- [x] Tornar nome e logo obrigatórios para publicação; descrição, endereço, telefone de contato e redes sociais são opcionais.
- [x] Validar slug único com 3 a 63 caracteres, letras ASCII minúsculas, números e hífens, sem hífen nas extremidades.
- [x] Reservar slugs que colidem com rotas da plataforma (`admin`, `api`, `assets`, `cadastro`, `confirmar-email`, `design-system`, `login`, `painel`, `recuperar-senha`, `redefinir-senha` e `testes`); preservar legados em conflito e diagnosticá-los sem renomeação automática.
- [x] Manter o slug imutável nos endpoints atualmente entregues.
- [x] Publicar a URL canônica `/{establishment_slug}` no MVP.
- [x] Validar e persistir somente identificadores IANA informados ao backend.
- [x] Na interface de cadastro, detectar o fuso do dispositivo, usar `America/Sao_Paulo` como fallback e permitir confirmação ou alteração antes do envio.
- [x] Consultar em `/testes` o estabelecimento da sessão sem enviar ID de tenant, exibindo nome, slug, fuso e trial ausente, ativo ou expirado; bloquear a consulta durante troca obrigatória de senha.
- [x] Consultar `GET /api/v1/establishment/settings` somente como proprietário, derivando o tenant da sessão autenticada e mantendo disponível a leitura em trial expirado.
- [x] Retornar dados opcionais, links ordenados, limites, metadados públicos do logo e estados de conclusão/publicação; sinalizar slug legado reservado sem expor chave ou caminho interno do arquivo.
- [x] Consultar sem sessão o perfil informativo de estabelecimento publicável por `GET /api/v1/public/establishments/{slug}`, expondo apenas nome, slug, URL pública do logo e dados opcionais preenchidos; manter a resposta após expiração do trial e recusar estabelecimentos incompletos ou com slug reservado.
- [x] Informar elegibilidade de escrita pelo componente central de direitos e permitir alteração futura do fuso somente antes do primeiro dado operacional.
- [x] Permitir que `PATCH /api/v1/establishment/settings` altere o fuso somente enquanto o marcador operacional estiver ausente, sob o mesmo bloqueio transacional; integrar o marcador aos futuros primeiros agendamentos, bloqueios e lançamentos permanece pendente.
- [x] Configurar pela API antecedência mínima e prazo de alteração em minutos não negativos, horizonte futuro em dias positivos e ausência/`null` como ausência de restrição.
- [ ] Aplicar os limites configurados nas validações de criação, cancelamento e reagendamento de clientes.
- [x] Criar painel de configurações do estabelecimento para editar dados da página pública e limites opcionais de agenda, preservando links legados incompatíveis com o editor e exigindo reconciliação quando a consulta falha após salvamento confirmado.
- [x] Permitir ao proprietário pré-visualizar no lugar da imagem publicada, limpar a seleção antes do envio, enviar e substituir o logo em Configurações com multipart e CSRF, atualizar o perfil após sucesso, restaurar o logo e limpar a seleção após falha confirmada, reconciliar respostas ambíguas e conservar a seleção até a reconciliação quando necessário, e bloquear alterações quando o trial expirar; não oferecer remoção do logo.
- [ ] Não criar perfil ou painel de administrador da plataforma no MVP.

## 8. Logos e arquivos públicos

- [x] Implementar upload e substituição de logo controlados pelo backend por `PUT /api/v1/establishment/logo`, com tenant da sessão, CSRF, entitlement de escrita, limite multipart e associação transacional; upload concluído pode completar os obrigatórios sem permitir remoção.
- [x] Validar conteúdo PNG, JPEG ou WebP, com até 2.097.152 bytes e dimensões positivas de até 4.096 × 4.096 pixels; rejeitar SVG e conteúdo truncado ou incompatível.
- [x] Derivar tipo, tamanho, dimensões e hash do conteúdo decodificado, sem confiar em extensão ou `Content-Type` informado pelo cliente.
- [x] Gerar nome interno UUID v4 não previsível e impedir que nome, extensão e cabeçalhos do cliente determinem chave ou caminho.
- [x] Persistir metadados e chave lógica estável no PostgreSQL, com associação do logo atual estruturalmente limitada ao mesmo estabelecimento.
- [x] Definir interface de object storage por streams e chave UUID v4 opaca, com adaptador local atômico, não sobrescrevível e confinado à raiz.
- [x] Disponibilizar o volume persistente próprio em `LOGO_STORAGE_DIR`, com permissões para o backend não privilegiado e preservação em deploys.
- [x] Servir o logo atual por `GET /api/v1/public/establishments/{slug}/logo` somente quando publicável, com resolução central do slug, storage por interface, MIME/tamanho validados, `nosniff`, ETag SHA-256 e cache revalidável; trial expirado não oculta a imagem e nenhuma chave/caminho interno é exposto.
- [x] Entregar em `/painel/configuracoes` o envio e a substituição do logo com pré-visualização substitutiva, limpeza da seleção temporária, mensagens de recusa, restauração do logo e limpeza do seletor após falha confirmada, reconciliação de respostas ambíguas, nova seleção para nova tentativa e atualização do endereço público após sucesso; conservar a consulta em `read_only` e não permitir remoção do logo publicado.

## 9. Administração de profissionais

- [x] Criar profissional do próprio estabelecimento pela API com e-mail globalmente único e senha temporária.
- [x] Listar pela API apenas profissionais do próprio estabelecimento, com paginação, `can_write` centralizado e leitura após expiração do trial.
- [x] Exibir no painel nome, e-mail e estado ativo com carregamento, vazio, erro, paginação e descarte de consultas obsoletas.
- [x] Consultar individualmente pela API profissionais ativos e inativos do próprio estabelecimento, inclusive sem trial ativo.
- [x] Editar somente o nome de profissional ativo ou inativo pela API, com validação compartilhada, CSRF parametrizado e direito de escrita, preservando identidade e credenciais.
- [x] Desativar profissionais pela API atomicamente com revogação de acesso e repetição segura.
- [ ] Reativar profissionais pela API.
- [ ] Consultar individualmente, editar, ativar e desativar profissionais pelo painel.
- [x] Oferecer cadastro pela página do painel com validação de conveniência, CSRF, tratamento seguro de erros, limpeza da senha temporária e atualização da lista; o backend mantém a validação autoritativa e o bloqueio de escrita após o trial.
- [ ] Configurar comissão padrão percentual ou fixa por profissional.
- [ ] Associar explicitamente serviços oferecidos por cada profissional.
- [ ] Permitir comissão específica por associação profissional-serviço sem alterar preço ou duração do serviço.
- [ ] Exibir quantidade de agendamentos futuros afetados antes da desativação.
- [x] Preservar perfil, identidade, e-mail e hash existentes após desativação, sem exclusão física.
- [ ] Integrar a preservação dos futuros vínculos, agendamentos e histórico financeiro após desativação.
- [ ] Alertar o proprietário sobre agendamentos futuros de profissional desativado, sem cancelamento automático.

## 10. Administração de serviços

- [x] Criar serviço ativo pela API exclusiva ao proprietário, com tenant da sessão, CSRF e direito centralizado de escrita.
- [x] Listar serviços pela API com paginação, ativos e inativos e direito de escrita atual, com leitura após expiração.
- [x] Consultar individualmente serviço ativo ou inativo pela API, com isolamento e leitura após expiração.
- [x] Editar nome, preço e duração pela API com PATCH parcial, validação atômica, CSRF em rota parametrizada e preservação da identidade.
- [x] Ativar e desativar serviço pela API com estado explícito idempotente, tenant, CSRF e direito de escrita.
- [x] Listar serviços no painel com paginação, preço BRL exato, estados de consulta e somente leitura.
- [x] Cadastrar serviço no painel com preço decimal convertido exatamente em centavos, limites visíveis de nome, preço e duração, duração padrão, erros por campo e bloqueio de envio simultâneo e modo somente leitura.
- [x] Consultar e editar serviço no painel com PATCH parcial, duração vazia como 30 minutos, cancelamento e reconciliação que preserva o rascunho e sincroniza o registro persistido na lista.
- [x] Ativar e desativar serviço no painel com confirmação, coordenação dos envios e consulta após resultado ambíguo, atualizando lista e editor aberto.
- [x] Validar nome obrigatório normalizado com até 120 pontos de código, sem caractere NUL, e preço de 0 a 9.999.999 centavos no backend; nomes repetidos e serviços gratuitos são permitidos.
- [x] Validar duração de 1 a 1.440 minutos inteiros no backend e usar 30 minutos quando omitida ou nula no cadastro.
- [ ] Manter preço e duração iguais para todos os profissionais associados.
- [x] Preservar registro, identidade e dados do serviço após desativação, sem endpoint de exclusão física.
- [ ] Preservar referências históricas e snapshots dos serviços nos futuros fluxos de agenda e financeiro.
- [ ] Exibir quantidade de agendamentos futuros afetados antes da desativação.
- [ ] Impedir novos agendamentos com serviço inativo sem alterar os já existentes.

## 11. Clientes

- [ ] Criar cliente implicitamente no primeiro agendamento público usando somente nome e telefone.
- [ ] Normalizar telefone brasileiro e garantir unicidade por estabelecimento.
- [ ] Reutilizar cliente existente por telefone sem sobrescrever seu nome pelo fluxo público.
- [ ] Não revelar divergência entre o nome informado e o cadastro existente.
- [ ] Permitir ao proprietário corrigir nome e telefone de qualquer cliente do estabelecimento.
- [ ] Permitir ao profissional corrigir somente o nome de cliente relacionado à própria agenda.
- [ ] Rejeitar troca para telefone já usado no estabelecimento e não mesclar históricos automaticamente.
- [ ] Preservar identidade e histórico após correções de nome ou telefone.
- [ ] Não coletar e-mail, documento, endereço ou observações do cliente no MVP.

## 12. Jornada, bloqueios e disponibilidade

- [ ] Permitir ao proprietário e ao profissional configurar a jornada semanal recorrente do profissional.
- [ ] Aceitar múltiplos períodos não sobrepostos no mesmo dia.
- [ ] Permitir bloqueios pontuais de intervalos pelo proprietário e pelo próprio profissional.
- [ ] Gerar inícios de slots a cada 15 minutos, sem buffer entre atendimentos.
- [ ] Calcular disponibilidade usando jornada, bloqueios, duração do serviço e agendamentos `scheduled`.
- [ ] Exigir associação ativa entre profissional e serviço para consultar e reservar disponibilidade.
- [ ] Considerar indisponível qualquer intervalo com sobreposição total ou parcial do profissional.
- [ ] Permitir ao mesmo cliente manter agendamentos sobrepostos, inclusive com profissionais diferentes.
- [ ] Aplicar fuso, antecedência mínima e horizonte máximo configurados para o cliente.
- [ ] Revalidar disponibilidade no backend durante criação e reagendamento.
- [ ] Impedir dupla ocupação do profissional sob concorrência.
- [ ] Criar testes unitários abrangentes para cálculo de slots, limites, sobreposições e fronteiras de data.

## 13. Página pública e agendamento do cliente

- [x] Renderizar no servidor a página canônica do estabelecimento com metadados, nome, logo e informações opcionais preenchidas, com nova consulta ao servidor ao tentar novamente após falha temporária.
- [x] Não renderizar seções vazias para informações públicas opcionais.
- [x] Limpar seleção e pré-visualização do logo após recusas definitivas, inclusive no primeiro upload, preservando a reconciliação de falhas ambíguas.
- [ ] Criar fluxo interativo para escolher serviço, profissional associado, dia e horário.
- [ ] Coletar somente nome e telefone para criar o agendamento.
- [ ] Criar agendamento com exatamente um serviço e um profissional.
- [ ] Copiar snapshots de preço e duração do serviço para o agendamento.
- [ ] Proteger criação contra submissões duplicadas e condições de corrida.
- [ ] Permitir consulta dos próprios agendamentos mediante telefone, sem conta, OTP, SMS ou WhatsApp.
- [ ] Na consulta pública, retornar somente agendamentos futuros `scheduled` e dados mínimos para identificação e ação.
- [ ] Aplicar rate limiting, identificadores não previsíveis e respostas resistentes a enumeração.
- [ ] Permitir cancelamento do cliente sem exigir motivo, antes do início e dentro do prazo configurado.
- [ ] Permitir reagendamento do cliente antes do início e dentro do prazo configurado.
- [ ] Reagendar atomicamente o mesmo registro, preservar histórico e manter o horário original em caso de falha.
- [ ] Preservar snapshots originais de preço e duração no reagendamento.
- [ ] Não expor histórico concluído, cancelado, `no_show` ou passado pela consulta pública.
- [x] Manter a página informativa identificável após expiração do trial, sem controles de agendamento nesta entrega.
- [ ] Quando assinatura ou trial estiver expirado, bloquear consulta de disponibilidade e criação/mutação nos futuros fluxos de agendamento.

## 14. Agenda autenticada

- [ ] Criar agenda operacional mobile-first para proprietário e profissional.
- [ ] Permitir ao proprietário visualizar e operar agendamentos de todos os profissionais.
- [ ] Restringir o profissional aos próprios agendamentos.
- [ ] Permitir criação, cancelamento e reagendamento interno, respeitando jornada, bloqueios e conflitos.
- [ ] Exigir justificativa para cancelamento feito por proprietário ou profissional e auditar autor e data.
- [ ] Exibir estados `scheduled`, `completed`, `canceled` e `no_show` traduzidos.
- [ ] Permitir ao profissional concluir ou marcar `no_show` apenas nos próprios agendamentos.
- [ ] Permitir ao proprietário concluir ou marcar `no_show` qualquer agendamento do estabelecimento.
- [ ] Impedir cliente e profissional de reverter estados finais.
- [ ] Permitir somente ao proprietário corrigir estados finais, com justificativa e auditoria.
- [ ] Permitir retorno para `scheduled` somente em horário futuro, após revalidar disponibilidade.
- [ ] Preservar histórico de estados, horários, atores, datas e justificativas.
- [ ] Alertar sobre agendamentos futuros ligados a serviço ou profissional desativado.
- [ ] Após reativação da assinatura, alertar sobre agendamentos passados que permaneceram `scheduled` durante a expiração.

## 15. Pagamentos dos atendimentos

- [ ] Registrar uma única forma de recebimento por atendimento com valor positivo: `pix`, `card` ou `cash`.
- [ ] Não exigir forma de pagamento para atendimento gratuito.
- [ ] Solicitar forma de pagamento somente na conclusão do atendimento.
- [ ] Permitir ao profissional informar a forma de pagamento ao concluir o próprio atendimento, sem alterar valor.
- [ ] Permitir somente ao proprietário corrigir valor ou forma de pagamento após conclusão, com justificativa e auditoria.
- [ ] Não processar pagamento, confirmar liquidação nem integrar adquirente para serviços do cliente final.
- [ ] Nunca coletar número do cartão, código de segurança ou outro dado de pagamento sensível.

## 16. Atendimentos e receitas

- [ ] Ao concluir atendimento com valor positivo, gerar receita e comissão atomicamente.
- [ ] Para atendimento gratuito, registrar receita e comissão zero sem forma de pagamento.
- [ ] Reconhecer receita pelo regime de caixa somente em atendimento concluído e válido.
- [ ] Permitir somente ao proprietário ajustar o valor antes ou depois da conclusão.
- [ ] Ao corrigir valor concluído, recalcular comissão com a regra registrada na conclusão.
- [ ] Implementar atendimento sem agendamento com um serviço, profissional, valor, data e forma de pagamento quando aplicável.
- [ ] Permitir atendimento avulso sem nome nem telefone do cliente.
- [ ] Exigir associação ativa entre profissional e serviço no atendimento avulso.
- [ ] Permitir ao proprietário registrar atendimento avulso atual ou passado para qualquer profissional.
- [ ] Permitir ao profissional registrar atendimento avulso apenas para si e no dia atual, usando o preço do serviço.
- [ ] Rejeitar atendimento avulso futuro.
- [ ] Persistir separadamente ocorrência, criação e autor do lançamento.
- [ ] Permitir somente ao proprietário anular atendimento lançado por engano, com justificativa e auditoria.
- [ ] Excluir receita e comissão anuladas dos totais sem apagar registros ou representar reembolso.

## 17. Comissões

- [ ] Suportar comissão padrão percentual ou fixa por profissional.
- [ ] Suportar regra específica por profissional e serviço, com precedência sobre a padrão.
- [ ] Validar percentual entre 0% e 100%.
- [ ] Validar valor fixo não negativo e não superior ao valor do atendimento.
- [ ] Calcular percentual sobre o valor efetivamente cobrado.
- [ ] Arredondar para centavos pela regra de metade para cima.
- [ ] Criar snapshot do tipo, regra e valor de comissão aplicados na conclusão.
- [ ] Preservar histórico diante de mudanças futuras na configuração.
- [ ] Exibir comissões calculadas sem implementar pagamento, liquidação, fechamento ou saldo devedor.

## 18. Despesas

- [ ] Criar despesas com descrição, categoria, valor e data prevista, sem forma de pagamento.
- [ ] Implementar estados `planned`, `paid` e `canceled`.
- [ ] Permitir despesa prevista com data atual ou futura sem incluí-la nos totais.
- [ ] Permitir criação direta como `paid` somente com data real atual ou passada.
- [ ] Permitir que somente o proprietário marque uma despesa como paga e informe a data real.
- [ ] Não alterar automaticamente o estado quando a data prevista chegar.
- [ ] Calcular alerta de atraso para despesa `planned` com data prevista passada, sem criar outro estado ou notificação.
- [ ] Permitir somente ao proprietário criar, editar e cancelar despesas.
- [ ] Exigir justificativa e auditoria em edição ou cancelamento.
- [ ] Excluir despesas canceladas dos totais sem exclusão física.
- [ ] Criar categorias padrão imutáveis: aluguel, água, energia, internet/telefone, materiais e insumos, manutenção, marketing, impostos e taxas e outros.
- [ ] Permitir categorias personalizadas limitadas ao estabelecimento.
- [ ] Desativar categoria personalizada já usada em vez de excluí-la.
- [ ] Não implementar recorrência de despesas no MVP.

## 19. Painel financeiro

- [ ] Criar painel geral exclusivo do proprietário.
- [ ] Exibir faturamento bruto de receitas válidas no período.
- [ ] Exibir total de comissões válidas no período.
- [ ] Exibir despesas pagas e não canceladas no período.
- [ ] Exibir resultado líquido como faturamento menos comissões e despesas.
- [ ] Exibir quantidade de atendimentos concluídos e ticket médio, retornando zero quando não houver atendimentos.
- [ ] Permitir filtros combinados por período, profissional, serviço e forma de pagamento.
- [ ] Usar o fuso do estabelecimento igualmente em cards, tabelas e cálculos.
- [ ] Criar visão financeira somente leitura do profissional com apenas seus atendimentos, valores e comissões.
- [ ] Não expor ao profissional despesas, resultado geral ou finanças de outros profissionais.

## 20. Trial, planos e assinatura

- [x] Iniciar um único trial de 14 dias após confirmação do e-mail, com acesso completo.
- [x] Serializar reenvio e consumo da confirmação na ordem proprietário → token, com regressão concorrente em PostgreSQL real.
- [x] Persistir uso do trial independentemente da assinatura atual para impedir reutilização.
- [ ] Modelar catálogo com planos, opções de preço e período, sem assumir plano único no schema ou domínio.
- [ ] Preservar versão, preço e termos contratados no histórico da assinatura.
- [ ] Preparar adicionais como itens independentes com preço e direitos próprios, sem implementar adicional real no MVP.
- [x] Centralizar a decisão de escrita durante o trial, sem condicionar endpoints diretamente às datas do benefício.
- [ ] Estender a resolução de direitos aos planos, períodos e adicionais futuros sem condicionais espalhadas por nome de plano.
- [ ] Criar interface de gateway de cobrança desacoplada.
- [ ] Implementar gateway mockado, explicitamente configurado e identificado, sem dados ou dinheiro reais.
- [ ] Permitir assinatura mockada mensal de `BRL 20,00`, aprovada imediatamente e de forma idempotente.
- [ ] Permitir renovação manual e idempotente; estender período ativo ou iniciar novo período após expiração.
- [ ] Permitir cancelamento ao fim do período, mantendo acesso até `current_period_end`.
- [ ] Não produzir reembolso automático nem apagar dados no cancelamento.
- [ ] Com trial ou assinatura expirados, preservar dados, congelar agendamentos e colocar o painel em leitura.
- [ ] Na reativação, restaurar direitos sem alterar automaticamente estados ou registros preservados.
- [ ] Criar telas de situação do trial, assinatura, contratação, renovação, cancelamento e acesso expirado.
- [ ] Não criar administração de catálogo pela interface no MVP; alterações são seeds ou migrations versionados.
- [ ] Definir estados, webhooks e idempotência antes de substituir o mock por gateway real.

## 21. E-mail transacional

- [x] Criar interface pequena de envio desacoplada do provedor.
- [x] Versionar o template de confirmação de e-mail em `pt-BR`.
- [x] Versionar o template de solicitação de recuperação de senha em `pt-BR`.
- [x] Integrar Mailpit por SMTP somente no ambiente local, com deadline total de 10 segundos incluindo conexão e interrupção por cancelamento.
- [x] Não registrar endereço completo, token, link ou credencial nas falhas dos fluxos de onboarding entregues.
- [x] Não registrar endereço completo, token ou link nas falhas de solicitação de recuperação.
- [ ] Preservar a mesma garantia nos demais fluxos futuros de e-mail transacional.
- [x] Falhar com segurança em produção quando o envio obrigatório não possuir provedor real configurado.
- [ ] Escolher e integrar um provedor real antes da disponibilização de produção que dependa de confirmação e recuperação.

## 22. Segurança e privacidade

- [x] Tratar nome do estabelecimento, e-mail, tokens e credenciais como sensíveis nos fluxos de onboarding entregues.
- [ ] Estender o tratamento de dados sensíveis aos futuros fluxos de clientes e financeiro.
- [x] Compartilhar a captura segura de tokens transacionais entre confirmação e recuperação, com testes de limpeza prévia da URL, falha segura e Strict Mode.
- [x] Não expor dados sensíveis em logs, path, query, mensagens de erro ou fixtures dos fluxos de onboarding; a interface de confirmação remove o token do fragmento antes de qualquer chamada de API e não o persiste.
- [ ] Preservar a mesma garantia nos demais fluxos futuros.
- [x] Usar respostas genéricas no login e na solicitação e verificação de confirmação para reduzir enumeração.
- [x] Aplicar resposta genérica à solicitação de recuperação para contas elegíveis, ausentes e inelegíveis.
- [ ] Aplicar respostas resistentes a enumeração na verificação do token de recuperação e na consulta pública futura.
- [x] Aplicar rate limiting ao login, cadastro e solicitação de confirmação por IP e e-mail normalizado.
- [x] Aplicar rate limiting à solicitação de recuperação.
- [ ] Aplicar rate limiting aos futuros fluxos públicos suscetíveis a abuso quando implementados.
- [x] Derivar e validar o tenant do `Principal` na consulta autenticada do estabelecimento.
- [x] Derivar o tenant do `Principal` e validar papel de proprietário no cadastro autenticado de profissionais.
- [x] Derivar o tenant do `Principal` e validar papel de proprietário na listagem autenticada de profissionais.
- [x] Validar tenant, papel do proprietário e perfil profissional na consulta individual, sem expor credenciais ou sessões.
- [x] Validar tenant e papel na desativação e revogar o acesso sem destruir registros existentes.
- [ ] Validar ownership, tenant e papel em cada recurso futuro acessado.
- [ ] Preservar histórico e preferir desativação a exclusão destrutiva.
- [ ] Não implementar exclusão ou anonimização automatizada de clientes no MVP.
- [ ] Revisar cookies, CORS, CSRF, headers de segurança e confiança em proxy antes da produção.

## 23. Logs, desempenho e auditoria

- [x] Gerar ou propagar `request_id` em todas as requisições.
- [x] Emitir logs JSON estruturados no backend em produção.
- [x] Registrar método, rota parametrizada, status, duração total e `request_id`.
- [ ] Medir tempo acumulado e quantidade de consultas PostgreSQL por requisição.
- [ ] Medir etapas relevantes, como validação, disponibilidade e cálculo financeiro, com nomes estáveis.
- [ ] Distinguir claramente tempo total, aplicação e banco sem dupla contagem.
- [x] Usar relógio monotônico e milissegundos consistentemente.
- [ ] Implementar históricos de auditoria exigidos para agenda, estados, valores, despesas e anulações.
- [ ] Manter logs do Caddy sem telefone, segredo ou parâmetros sensíveis.
- [x] Configurar rotação de todos os containers com `max-size: 10m` e `max-file: 5`.
- [x] Usar somente `stdout` e `stderr`, sem arquivos de log internos aos containers.

## 24. Docker, proxy e ambiente local

- [x] Criar Dockerfiles reproduzíveis para frontend e backend com versões fixadas e imagens mínimas.
- [x] Criar Docker Compose de produção com frontend, backend, PostgreSQL e Caddy.
- [x] Criar override ou profile de desenvolvimento com Mailpit.
- [x] Manter frontend, backend, PostgreSQL e SMTP em redes internas; somente Caddy fica público em produção.
- [x] Expor a interface do Mailpit somente localmente.
- [x] Configurar volumes persistentes para PostgreSQL, dados do Caddy e logos.
- [x] Adicionar health checks reais e esperas por prontidão, sem depender apenas de `depends_on`.
- [x] Configurar Caddy para HTTPS automático, `/api/v1` no backend e demais rotas no frontend.
- [ ] Preservar cabeçalhos necessários e confiar em proxy headers somente quando vindos do Caddy controlado.
- [x] Manter o deploy agnóstico ao provedor da VPS.

## 25. CI/CD, migrations e releases

- [x] Criar GitHub Actions para geração de código e detecção de artefatos desatualizados.
- [ ] Executar testes, lint, formato, tipos e builds aplicáveis antes de release.
- [x] Executar testes unitários e testes de integração do backend como gate obrigatório de pull request.
- [ ] Disparar produção apenas por GitHub Release estável com tag SemVer `vMAJOR.MINOR.PATCH`.
- [ ] Não implantar drafts ou pre-releases.
- [ ] Fazer deploy por SSH com usuário dedicado e segredos do GitHub Actions.
- [ ] Fazer checkout explícito do commit da tag na VPS e construir imagens localmente.
- [ ] Validar Compose antes de substituir containers e registrar o hash implantado.
- [ ] Executar `goose up` como tarefa isolada antes de atualizar o backend.
- [ ] Interromper deploy em falha de migration e manter versão anterior em execução.
- [ ] Usar expand/contract em releases separadas para mudanças destrutivas.
- [ ] Preservar todos os volumes durante deploys.
- [ ] Não introduzir registry de imagens ou backup automatizado sem nova decisão.

## 26. Testes e critérios de qualidade

- [x] Criar testes unitários de backend com o pacote `testing`.
- [x] Testar elegibilidade, expiração, recusa de senha atual e falhas da recuperação sem PostgreSQL, preservando integração para bloqueios, uso único e rollback.
- [x] Criar testes unitários de frontend com Vitest.
- [x] Orientar testes unitários a comportamentos observáveis e regras de negócio, evitando detalhes internos e casos sem risco ou requisito identificável.
- [x] Criar infraestrutura de integração do backend com Testcontainers, PostgreSQL real, migrations Goose e servidor HTTP TLS descartável.
- [x] Cobrir por integração os fluxos entregues de health check, cadastro e confirmação do proprietário, autenticação, sessão, consulta do estabelecimento e troca obrigatória de senha do profissional.
- [ ] Cobrir prioritariamente isolamento de tenant, autorização, autenticação e política de assinatura.
- [ ] Cobrir disponibilidade, concorrência lógica, snapshots e transições de agendamento.
- [ ] Cobrir receitas, despesas, comissões, arredondamento, anulação e indicadores.
- [ ] Cobrir validação de dados, estados vazios e comportamento dos componentes críticos.
- [ ] Adicionar teste unitário de regressão para toda correção de bug tecnicamente testável.
- [x] Executar `gofmt`, testes unitários e de integração e `go vet -tags=integration ./...` no backend.
- [x] Executar testes, ESLint, Prettier, checagem de tipos e build no frontend.
- [x] Manter a integração restrita ao backend e não adicionar E2E nem integração de frontend sem nova decisão.

## 27. Fora do MVP

- [ ] Não implementar notificações ou confirmações de agenda por WhatsApp ou SMS.
- [ ] Não implementar processamento de pagamentos dos serviços dos clientes finais.
- [ ] Não implementar nota fiscal ou integração fiscal.
- [ ] Não implementar reembolso ou estorno financeiro ao anular um lançamento.
- [ ] Não implementar despesas recorrentes.
- [ ] Não implementar liquidação ou pagamento de comissões.
- [ ] Não implementar MFA.
- [ ] Não implementar alteração de slug ou e-mail de login.
- [ ] Não implementar múltiplos serviços no mesmo agendamento.
- [ ] Não implementar buffer entre atendimentos.
- [ ] Não implementar exclusão ou anonimização automatizada de clientes.
- [ ] Não implementar painel de administrador da plataforma.
- [ ] Não implementar subdomínios no MVP; apenas manter o resolvedor preparado para evolução futura.
- [ ] Não implementar backup automatizado ou registry de imagens sem nova decisão.

## 28. Definição de conclusão do MVP

- [ ] Todos os escopos do MVP acima estão implementados ou explicitamente reclassificados por decisão documentada.
- [ ] OpenAPI, código gerado, migrations, backend e frontend estão sincronizados.
- [ ] Testes unitários, testes de integração do backend e verificações completas passam a partir de um checkout limpo.
- [ ] O projeto sobe integralmente com Docker Compose e dados persistem após recriação dos containers.
- [ ] O fluxo público funciona em dispositivo móvel e o painel funciona em desktop e tela responsiva.
- [ ] Isolamento de tenant, papéis e expiração de assinatura foram revisados de ponta a ponta.
- [ ] Logs permitem correlacionar requisições e identificar tempo de aplicação e banco sem revelar dados sensíveis.
- [ ] Deploy por GitHub Release foi validado na VPS com migration segura e preservação de volumes.
- [ ] Provedor real de e-mail está configurado para produção.
- [ ] Documentação e checklist refletem exatamente o comportamento entregue.
