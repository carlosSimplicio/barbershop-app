# Operação e entrega

## Infraestrutura

- Produção roda em uma única VPS com Docker Compose.
- Serviços: frontend, backend, PostgreSQL e Caddy.
- O Compose de desenvolvimento inclui também o Mailpit, preferencialmente por arquivo de override ou profile próprio; ele não faz parte do deploy de produção.
- Exponha a interface web do Mailpit somente no ambiente local e mantenha seu SMTP acessível apenas pela rede interna do Compose.
- Somente Caddy expõe portas HTTP/HTTPS publicamente. Frontend, backend e PostgreSQL permanecem nas redes internas necessárias.
- PostgreSQL e dados operacionais do Caddy usam volumes persistentes.
- Logos enviados pelos estabelecimentos usam um volume persistente local próprio, preservado em deploys e não incorporado às imagens.
- Adicione health checks. `depends_on` não substitui verificação real de prontidão.
- Considere os recursos da VPS e não introduza serviços gerenciados ou dependências externas sem justificativa e decisão explícita.
- Mantenha o deploy agnóstico ao provedor da VPS. Não dependa de serviços proprietários do fornecedor sem uma decisão arquitetural explícita.
- Ao criar cada componente, selecione a versão estável mais recente disponível naquele momento e fixe-a explicitamente em Dockerfiles, imagens, módulos e lockfiles.
- Não use tags `latest`, intervalos flutuantes ou resolução dinâmica de versão em build e deploy. Atualizações são mudanças deliberadas, revisadas e verificadas antes da release.

### Ambiente local da fundação

- Copie `.env.example` para `.env` e execute `make dev` para subir o ambiente de desenvolvimento. Use `make dev-down` para remover seus containers e redes, incluindo serviços órfãos, sem apagar os volumes persistentes.
- A aplicação é publicada pelo Caddy em `https://localhost:8443`; `http://localhost:8088` apenas redireciona para HTTPS. A interface do Mailpit fica restrita a `http://127.0.0.1:8025`.
- Quando houver migrations de domínio, execute `make migrate-up` para aplicá-las com a imagem isolada do Goose após o PostgreSQL ficar saudável.
- `compose.yaml` representa a topologia de produção; `compose.dev.yaml` acrescenta o Mailpit, sua interface local e fixa o endereço HTTPS de desenvolvimento do Caddy.
- PostgreSQL e SMTP ficam na rede interna de dados. No override local, o Mailpit também participa de uma bridge exclusiva de desenvolvimento para publicar somente sua UI em `127.0.0.1`; a porta SMTP não é publicada. Backend e frontend ficam atrás da rede interna de proxy. Somente o Caddy participa também da rede de borda e publica as portas da aplicação.
- No PostgreSQL 18, monte o volume persistente em `/var/lib/postgresql`; o layout interno versionado é responsabilidade da imagem oficial.
- Use `make compose-config` para validar a composição, `make test-unit` para os testes sem infraestrutura, `make test-integration` para os fluxos do backend contra PostgreSQL descartável e `make test` para ambas as suítes. `make check` também executa geração, formato, análise estática, tipos e todos os testes.
- Os testes de integração exigem Docker acessível ao processo de teste. Testcontainers cria e remove um PostgreSQL `18.6-alpine3.24`, e Goose aplica todas as migrations antes da execução; eles não reutilizam nem alteram o banco de desenvolvimento.
- Antes do primeiro `make dev`, use `make auth-keys` e copie as cinco variáveis produzidas para o `.env`. A saída contém segredos locais e não deve ser versionada nem reutilizada em produção.
- Para confiar no certificado local, copie `/data/caddy/pki/authorities/local/root.crt` do container Caddy para um arquivo temporário e instale-o manualmente no repositório de autoridades do sistema ou navegador. Nunca versione essa CA. Reinicie o navegador depois da instalação.

## Caddy e HTTPS

- Caddy é o único ponto público de entrada e termina HTTPS.
- Encaminhe `/api/v1` ao backend e as demais rotas ao frontend.
- Use HTTPS automático para o domínio único do MVP e persista `/data` do Caddy.
- Preserve cabeçalhos de host, protocolo e cliente necessários, mas confie neles somente quando vierem do proxy controlado.
- A rede `proxy` usa subnet interna fixa e reserva `172.30.0.4` para o Caddy. O backend confia em `X-Forwarded-*` somente quando a conexão direta vem de `172.30.0.4/32`; frontend e outros containers não são proxies confiáveis.
- Uma futura adoção de wildcard/subdomínios exige DNS-01 e é uma mudança separada de infraestrutura.

## Configuração e segredos

- Produção usa `.env` somente na VPS, com permissões restritas.
- Versione `.env.example` com todas as chaves, sem valores sensíveis.
- Ignore arquivos `.env` reais e nunca os copie para imagens ou logs.
- Cada aplicação valida configuração obrigatória na inicialização e falha claramente se estiver ausente ou inválida.
- Segredos nunca usam `NEXT_PUBLIC_*`.
- Chaves JWT, credenciais do PostgreSQL e outros segredos ficam somente no backend e na infraestrutura.
- Configure `JWT_ACTIVE_KEY_ID`, `JWT_PRIVATE_KEY_BASE64`, `JWT_PUBLIC_KEYS_JSON`, `CSRF_SIGNING_KEY_BASE64`, `RATE_LIMIT_HMAC_KEY_BASE64`, `TRUSTED_ORIGINS` e `TRUSTED_PROXY_CIDRS`. A chave JWT privada é um seed Ed25519 em base64 sem padding; o keyring público é um objeto JSON de `kid` para chave pública.
- Configure `PUBLIC_APP_URL` como a origem HTTPS usada nos links transacionais. No ambiente local, `EMAIL_PROVIDER=smtp`, `SMTP_HOST`, `SMTP_PORT` e `SMTP_FROM` apontam exclusivamente para o Mailpit interno. O modo desabilitado é aceito somente em testes; produção recusa a inicialização enquanto não houver um provedor real suportado e explicitamente configurado.
- `PUBLIC_APP_URL` e cada item de `TRUSTED_ORIGINS` devem ser origens HTTPS sem path, query ou fragmento. `SMTP_HOST` não inclui porta; `SMTP_PORT` aceita de 1 a 65535; `SMTP_FROM` precisa ser um endereço válido. `EMAIL_PROVIDER=disabled` é exclusivo de testes.
- Ao adicionar configuração obrigatória, atualize `.env.example` e a documentação operacional.

### Validação local do onboarding pela API

- Suba o ambiente com `make dev`, aplique as migrations com `make migrate-up` e acesse o Mailpit em `http://127.0.0.1:8025`.
- Cadastre um owner por `POST /api/v1/auth/register`; a resposta esperada é `202` e a mensagem deve aparecer no Mailpit. Falha de SMTP depois da persistência não desfaz o cadastro.
- Solicite uma substituição por `POST /api/v1/auth/email-confirmation`; a resposta permanece `202` para conta inexistente, já confirmada ou elegível. Para uma conta elegível, use somente a mensagem mais recente, pois o token anterior foi invalidado.
- Copie manualmente apenas o valor de `token` do fragmento de `/confirmar-email#token=...` e envie-o no corpo de `POST /api/v1/auth/email-confirmation/verify`. Nunca coloque o token em path ou query. A resposta `200` cria a sessão e o trial.
- Preserve os cookies retornados e consulte `GET /api/v1/establishment`; a resposta deve conter o mesmo tenant, o fuso persistido e o trial ativo com limites UTC separados por exatamente 14 dias.
- Use identificadores fictícios e descarte cookie jar e token ao terminar. Os comandos completos estão no `README.md`.

## CI/CD e releases

- GitHub Actions executa verificações e inicia deploy por SSH na VPS.
- Produção é acionada somente pela publicação de uma GitHub Release estável.
- Use SemVer em tags `vMAJOR.MINOR.PATCH`; não mantenha fluxo de deploy para drafts ou pre-releases.
- O deploy usa exatamente o commit referenciado pela tag.
- Antes do SSH, execute geração, testes, lint, formato, tipos e builds aplicáveis; qualquer falha impede o deploy.
- Host, usuário, chave SSH e credenciais ficam em GitHub Actions Secrets e não aparecem em logs.
- Use um usuário de deploy com permissões mínimas.
- A VPS recebe o código, faz checkout explícito do commit e constrói as imagens localmente com Docker Compose.
- Não use registry no fluxo inicial. Adoção futura de GHCR ou equivalente é uma mudança de processo.
- Valide a configuração Compose antes de substituir containers e registre apenas o hash do commit implantado.
- Preserve volumes do PostgreSQL e Caddy e evite comandos destrutivos.

### Migrations no deploy

- Com PostgreSQL saudável, execute `goose up` como tarefa isolada antes de atualizar o backend.
- Falha de migration interrompe o deploy e mantém a versão anterior em execução.
- Migrations precisam ser compatíveis com a versão ainda ativa durante a atualização.
- Para mudanças destrutivas, use etapas expansivas e contrativas em releases separadas.
- A migration `00002_owner_onboarding.sql` exige que `establishments` esteja vazia, pois os registros mínimos anteriores não possuem nome, slug ou fuso que possam ser preenchidos sem inventar dados. Se a precondição falhar, interrompa o deploy e trate explicitamente os registros antes de reaplicar; não os apague nem atribua valores fictícios.

## Persistência

- Não existe rotina de backup do PostgreSQL no escopo inicial.
- Não introduza backup externo ou sua automação sem nova decisão.
- Preserve o volume em deploys e manutenções.
- A ausência de backup também se aplica ao volume de logos no escopo inicial.
- Antes de alteração destrutiva, destaque a ausência de backup e prefira alternativa reversível.

### Manutenção dos dados de autenticação

- Ainda não existe tarefa agendada para retenção de `auth_sessions`, `refresh_tokens`, `email_confirmations` ou `auth_login_attempts`; enquanto isso, as quatro tabelas crescem sem limite. Implemente e valide essa manutenção antes de operar autenticação continuamente em produção.
- Execute a manutenção ao menos diariamente como comando idempotente e observável, invocado pelo ambiente da VPS sem criar um novo serviço permanente. Proteja execuções concorrentes e processe exclusões em lotes para limitar bloqueios, WAL e picos de I/O.
- Antes de habilitar exclusões, registre apenas contagens agregadas dos candidatos, valide as janelas definidas na arquitetura e confirme a ordem exigida pelas chaves estrangeiras. Nunca registre hashes, tokens, IPs, e-mails ou identificadores individuais.
- Acompanhe por tabela a contagem aproximada, o tamanho total e a idade do registro elegível mais antigo. Confirmações só podem ser removidas depois de consumidas, invalidadas ou expiradas e da margem operacional definida. Emita alerta quando a rotina falhar ou quando a idade ultrapassar a janela esperada; não dependa apenas do tamanho atual do volume.
- Deixe o autovacuum recuperar o espaço para reutilização normal. `VACUUM FULL` exige bloqueio e reescrita da tabela e só pode ser uma intervenção excepcional planejada.

## Logs e desempenho

- Use somente logs dos containers via Docker Compose; não adicione plataforma externa no MVP.
- Configure rotação para todos os containers com `max-size: 10m` e `max-file: 5`, evitando crescimento ilimitado no disco da VPS.
- Aplicações e Caddy escrevem em `stdout`/`stderr`; não mantenha arquivos de log próprios dentro dos containers.
- Backend escreve JSON estruturado em `stdout`/`stderr` em produção.
- Gere ou propague `request_id` em todo o fluxo.
- Para cada requisição, registre método, rota parametrizada, status, duração total e `request_id`.
- Cronometre etapas relevantes com nomes estáveis, incluindo tempo acumulado e quantidade de consultas PostgreSQL.
- Quando útil, separe validação, disponibilidade e cálculos financeiros.
- Use relógio monotônico e milissegundos de forma consistente.
- Evite dupla contagem e indique se tempo de aplicação inclui ou exclui banco.
- Não registre corpos completos, senhas, JWTs, telefones, parâmetros SQL, dados pessoais ou segredos.
- Use rota parametrizada, nunca URL concreta, para evitar cardinalidade e vazamentos.
- Logs de acesso do Caddy não contêm dados sensíveis; telefones e segredos nunca vão em URL ou query string.

## Testes e qualidade

- O backend combina testes unitários com `testing` e testes de integração sob a build tag `integration`; o frontend usa testes unitários com Vitest.
- Regras devem continuar testáveis sem rede, HTTP ou banco reais. A integração complementa essa cobertura nos contratos entre roteamento, middleware, serviços, persistência, migrations e PostgreSQL.
- A suíte de integração usa Testcontainers e exige Docker. Ela sobe um banco descartável por execução, aplica as migrations reais e usa servidor HTTP TLS local; falha se essa infraestrutura não puder ser provisionada.
- Não adicione E2E nem integração de frontend sem decisão nova.
- Testes unitários verificam comportamentos observáveis e regras de negócio, cobrindo fluxos, recusas e invariantes relevantes. Evite acoplá-los a detalhes internos, casos sem risco ou requisito identificável e comportamento de bibliotecas.
- Testes de integração verificam as jornadas reais entregues e priorizam autenticação, autorização, isolamento de tenant, transações e constraints; amplie-os quando novos fluxos persistidos forem implementados.
- Correções incluem teste de regressão no nível mais baixo capaz de reproduzir o defeito com fidelidade.
- Backend: `gofmt`, testes unitários, testes de integração e `go vet -tags=integration ./...`.
- Frontend: testes unitários, ESLint, Prettier e checagem de tipos.
- Verifique builds quando a mudança afetar compilação ou empacotamento.

## Git e pull requests

- Não há formato obrigatório ou Conventional Commits.
- O gate obrigatório de pull request executa os testes unitários aplicáveis e os testes de integração do backend.
- Lint, formato, tipos, geração e builds continuam parte da validação e do pipeline de release.
- Commits devem ser focados e não misturar alterações não relacionadas.
