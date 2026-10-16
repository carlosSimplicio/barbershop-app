# Autenticação backend: sessões seguras e isolamento de tenant

**Status:** implementado e verificado.

Entregar a fundação de autenticação para proprietários e profissionais: login, sessão atual, JWT, refresh rotativo, logout, senha temporária, CSRF, rate limiting e middleware de autorização. Cadastro, confirmação e recuperação por e-mail, criação de profissionais, trial e telas ficam para etapas posteriores.

## Interfaces públicas

- `POST /api/v1/auth/login`: recebe `email` e `password`, cria uma sessão e retorna o usuário autenticado.
- `POST /api/v1/auth/refresh`: rotaciona o refresh token e renova a expiração deslizante.
- `POST /api/v1/auth/logout`: revoga idempotentemente a sessão atual e remove os cookies.
- `GET /api/v1/auth/me`: retorna `user_id`, `establishment_id`, `email`, `role` e `password_change_required`.
- `POST /api/v1/auth/password/change`: aceita a nova senha do profissional autenticado com senha temporária, revoga todas as sessões e reemite a atual.
- Cookies:
  - `__Secure-barbershop_access`: `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/api/v1`, validade de 15 minutos.
  - `__Secure-barbershop_refresh`: mesmas proteções, `Path=/api/v1/auth`, validade deslizante de 7 dias.
  - `__Secure-barbershop_csrf`: legível pelo navegador, assinado, `Secure`, `SameSite=Strict`, `Path=/`.
- Erros estáveis: `invalid_credentials`, `email_confirmation_required`, `authentication_required`, `invalid_session`, `password_change_required`, `csrf_failed` e `authentication_rate_limited`.

### 1. Persistir identidades, credenciais e sessões

- **O que:** criar migration Goose para `establishments` como raiz mínima de tenant, `users`, `user_credentials`, `auth_sessions`, histórico de `refresh_tokens` e tentativas de login. Garantir e-mail normalizado globalmente único, papel `owner|professional`, vínculo obrigatório com um único estabelecimento, estado ativo, confirmação do e-mail, versão da credencial e indicador de senha temporária.
- **Motivação:** dar suporte a isolamento, revogação imediata, rotação e detecção de reutilização sem antecipar onboarding ou administração de profissionais.
- **Onde:** `backend/db/migrations`, `backend/db/query` e artefatos gerados pelo `sqlc`.
- **Base documental:** `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/architecture.md`, seções “Banco de dados” e “Autenticação”; `docs/implementation-scope.md`, itens 3, 5 e 6.
- **Verificação:** consultas sempre exigem `establishment_id` quando aplicável; constraints impedem e-mail duplicado, usuário sem tenant e papéis inválidos; geração do `sqlc` permanece limpa.

### 2. Implementar criptografia e configuração segura

- **O que:** criar componentes injetáveis para:
  - Argon2id em formato PHC, salt de 16 bytes e hash de 32 bytes, inicialmente com `19 MiB`, `t=2`, `p=1`, permitindo aumento controlado e rehash futuro. Esse é o mínimo indicado pela [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
  - JWT Ed25519/EdDSA com chave ativa identificada por `kid` e keyring de chaves públicas anteriores.
  - Refresh tokens opacos de 256 bits, persistidos somente como hash.
  - Assinatura dos tokens CSRF e HMAC dos identificadores de rate limiting, evitando persistir IP ou e-mail nessas tabelas.
- **Motivação:** proteger credenciais e tokens mesmo após vazamento do banco e permitir rotação de chaves sem invalidar access tokens ainda válidos.
- **Onde:** novos componentes compartilhados de autenticação no backend, `internal/config`, `.env.example` e configuração do container.
- **Base documental:** `docs/architecture.md`, seção “Autenticação”; `docs/operations.md`, seção “Configuração e segredos”; decisões confirmadas sobre Ed25519, keyring e Argon2id.
- **Verificação:** testes unitários cobrem validação de senha, PHC inválido, parâmetros antigos, assinatura, `kid` desconhecido, algoritmo divergente, token expirado e configuração obrigatória ausente.
- **Dependências:** subtarefa 1.

### 3. Entregar login e limitação progressiva

- **O que:** implementar login com normalização de e-mail, comparação de tempo uniforme para contas inexistentes, resposta genérica para senha inválida ou usuário desativado e `email_confirmation_required` somente depois de uma senha correta. Registrar falhas em dimensões independentes de IP e e-mail normalizado: após 5 falhas em 15 minutos, 10 em 1 hora ou 20 em 24 horas, bloquear novas tentativas pela maior janela ativa e retornar `429` com `Retry-After`.
- **Motivação:** autenticar ambos os papéis sem enumeração direta e conter ataques persistentes sem bloqueio permanente.
- **Onde:** slice de login, repositório PostgreSQL de rate limiting e contrato OpenAPI.
- **Base documental:** `docs/architecture.md`, seção “Autenticação”; `docs/implementation-scope.md`, item 5; decisões confirmadas sobre progressão e persistência no PostgreSQL.
- **Verificação:** testes cobrem login válido, conta inexistente, senha incorreta, proprietário não confirmado, profissional desativado, senha temporária, cada janela de bloqueio e ausência de dados sensíveis em respostas e logs.
- **Dependências:** subtarefas 1 e 2.

### 4. Implementar ciclo de sessão e proteção das rotas

- **O que:** implementar access JWT com usuário, estabelecimento, papel, expiração e versão da credencial; o middleware consulta o usuário para validar estado e versão, garantindo revogação imediata. Permitir várias sessões por usuário, sem limite absoluto, expirando após 7 dias sem refresh. Fazer refresh transacional com consumo único e rotação; reutilização revoga toda a família daquela sessão. Adicionar logout atual idempotente, validação estrita de `Origin`, double-submit CSRF assinado e contexto autenticado com tenant e papel derivados da sessão. Restringir o profissional com senha temporária aos endpoints de sessão, logout e troca de senha; a troca rejeita reutilizar a senha temporária, incrementa a versão da credencial, revoga todas as sessões e cria uma nova sessão atual, sem solicitar novamente a senha temporária.
- **Motivação:** tornar renovação e revogação observáveis, impedir CSRF e estabelecer a base centralizada de autorização e multi-tenancy.
- **Onde:** slices de sessão e senha, middleware HTTP, composição do servidor e OpenAPI; regenerar backend e tipos TypeScript sem editar gerados.
- **Base documental:** `docs/architecture.md`, seção “Autenticação”; `docs/product.md`, seção “Cadastro, usuários e permissões”; `docs/implementation-scope.md`, itens 5, 6 e 22.
- **Verificação:** testes unitários cobrem refresh válido, expirado, revogado e reutilizado; isolamento por tenant; papel adulterado; versão divergente; origem ou CSRF ausentes; logout repetido; restrição e troca da senha temporária.
- **Dependências:** subtarefas 1 a 3.

### 5. Habilitar cookies `Secure` também no desenvolvimento

- **O que:** publicar o ambiente local em `https://localhost:8443` usando a CA interna do Caddy, mantendo ACME automático em produção. Documentar a exportação e instalação manual da CA local; preservar frontend e backend em HTTP apenas nas redes internas. Configurar origens confiáveis e confiança em proxy somente para o Caddy controlado.
- **Motivação:** manter `Secure` obrigatório em todos os ambientes e obter o IP real necessário ao rate limiting sem confiar em headers externos arbitrários.
- **Onde:** Compose de desenvolvimento, Caddy, `.env.example`, `README.md` e `docs/operations.md`.
- **Base documental:** `docs/operations.md`, seções “Ambiente local”, “Caddy e HTTPS” e “Configuração e segredos”; decisão confirmada sobre HTTPS local com CA interna.
- **Verificação:** `make compose-config`; acesso local por HTTPS com certificado confiado; HTTP redireciona para HTTPS; cookies nunca são emitidos sem `Secure`.
- **Dependências:** subtarefas 2 e 4.

## Verificação conjunta

- Atualizar `docs/architecture.md` com contratos, cookies, CSRF, keyring, sessão e rate limiting; atualizar `docs/operations.md`, `README.md` e os checkboxes efetivamente concluídos de `docs/implementation-scope.md`, dividindo itens compostos quando parte deles permanecer pendente.
- Executar geração de OpenAPI e `sqlc`, `go test ./...`, `gofmt`, `go vet ./...`, verificação de artefatos gerados, checagens do frontend afetadas pelos tipos e builds de produção.
- Permanecem fora desta entrega: cadastro e onboarding, confirmação e recuperação por e-mail, criação e desativação de profissionais, trial e assinatura, telas e estado de sessão no frontend. Como nenhum fluxo cria usuários ainda, a capacidade será validada por testes unitários e ficará disponível para os slices seguintes, sem seed público de credenciais.
