# AGENTS.md

## Projeto

Plataforma multi-tenant de agendamento e gestão financeira para barbearias, salões de cabeleireiro, manicures e estabelecimentos semelhantes. Clientes agendam pela página pública do estabelecimento; proprietários e profissionais operam agenda e financeiro pelo painel.

## Documentação obrigatória por contexto

- Leia `docs/product.md` ao alterar usuários, permissões, clientes, agenda, disponibilidade, serviços, profissionais, atendimentos ou financeiro.
- Leia `docs/architecture.md` ao alterar frontend, backend, API, autenticação, banco de dados, estrutura do monorepo ou multi-tenancy.
- Leia `docs/operations.md` ao alterar Docker Compose, Caddy, configuração, logs, CI/CD, deploy, migrations de produção ou verificações automatizadas.
- Leia `docs/implementation-scope.md` ao planejar trabalho, iniciar um novo escopo ou verificar o que ainda precisa ser construído. Atualize seus checkboxes no mesmo trabalho que concluir ou reabrir um item.
- Consulte somente os documentos relacionados à tarefa. Se uma mudança cruzar áreas, leia todos os documentos aplicáveis.
- Quando uma decisão do produto ou da arquitetura mudar, atualize no mesmo trabalho o documento correspondente e este arquivo apenas se a regra for universal.
- Trate `AGENTS.md` e os arquivos em `docs/` como documentação viva e parte obrigatória da entrega. Toda alteração que modifique comportamento, regra de negócio, contrato, arquitetura, operação ou processo deve atualizar os documentos afetados no mesmo pull request ou commit.
- Antes de concluir uma tarefa, confira se implementação, testes, OpenAPI e documentação permanecem consistentes. Não deixe para uma atualização futura uma divergência conhecida.
- Não altere a documentação apenas para justificar acidentalmente uma implementação divergente. Quando houver conflito, siga a decisão documentada ou confirme explicitamente uma nova decisão antes de atualizar código e documentação em conjunto.

## Regras universais

- Preserve o isolamento entre estabelecimentos em toda consulta e alteração. Nunca confie apenas em um identificador fornecido pelo cliente; valide estabelecimento, papel e propriedade do recurso no backend.
- O backend Go é a autoridade para regras de negócio, autorização, validação e persistência. O frontend nunca substitui validações do backend.
- Não presuma regras de agenda, disponibilidade, autorização ou cálculos financeiros que não estejam documentadas. Confirme decisões que mudem comportamento de negócio.
- Trate nome, telefone, e-mail, tokens e dados financeiros como sensíveis. Não os exponha em logs, URLs, mensagens de erro ou fixtures públicas.
- Preserve histórico de agendamentos e financeiro. Prefira desativação e registros imutáveis a exclusões que destruam referências históricas.
- Use inglês para código, schema, endpoints e identificadores técnicos. Use português brasileiro nos textos apresentados ao usuário.
- Siga os padrões existentes antes de introduzir novas dependências ou abstrações. Mantenha a solução simples e justifique qualquer divergência.

## Estrutura e stack

- Monorepo com `frontend/` e `backend/` separados.
- Frontend: Next.js App Router, React e TypeScript estrito; npm; CSS ou SCSS próprios.
- Backend: Go, Gin, Vertical Slice Architecture simples e API REST em `/api/v1`.
- Dados: PostgreSQL, `pgx/v5`, SQL explícito com `sqlc` e migrations Goose.
- Contrato: OpenAPI contract-first em `backend/openapi/openapi.yaml`, com `oapi-codegen` no backend e `openapi-typescript` no frontend.
- Produção: Docker Compose em uma VPS, com Caddy como único ponto público de entrada.

## Código gerado e banco de dados

- Não edite manualmente arquivos gerados por `sqlc`, `oapi-codegen` ou `openapi-typescript`.
- Após alterar SQL ou OpenAPI, regenere os artefatos afetados e verifique backend e frontend.
- Toda mudança de schema exige uma migration Goose nova. Não reescreva migrations que possam ter sido aplicadas.
- Use transações para invariantes que envolvam múltiplas escritas.
- Não há backup no escopo inicial. Antes de qualquer operação destrutiva de schema ou dados, destaque o risco e prefira uma alternativa reversível.

## Testes e verificação

- O escopo atual inclui apenas testes unitários: pacote `testing` no Go e Vitest no frontend.
- Não introduza testes de integração, testcontainers ou E2E sem uma nova decisão.
- Escreva testes unitários orientados a comportamentos observáveis e regras de negócio. Priorize fluxos, recusas e invariantes relevantes; evite testar detalhes internos, casos sem risco ou requisito identificável e comportamento de bibliotecas.
- Toda correção de bug deve incluir teste unitário de regressão quando tecnicamente aplicável.
- Antes de concluir mudanças no backend, execute os testes unitários, `gofmt` e `go vet ./...`.
- Antes de concluir mudanças no frontend, execute testes unitários, ESLint, Prettier e checagem de tipos.
- Verifique builds de produção quando a mudança puder afetar compilação ou empacotamento.
- Se uma verificação relevante não puder ser executada ou já estiver falhando, informe claramente na entrega.

## Revisão final de features

- Ao finalizar a implementação de cada feature, revise todo o diff e os fluxos afetados antes de concluir a tarefa.
- Procure por bugs, casos-limite não tratados, erros lógicos, regressões e violações das regras de negócio, autorização ou isolamento entre estabelecimentos.
- Avalie se nomes, responsabilidades, fluxo de controle e estrutura do código permanecem legíveis e coerentes com a arquitetura existente, evitando duplicações e abstrações desnecessárias.
- Identifique possíveis refatorações que reduzam complexidade, acoplamento ou duplicação e melhorem a manutenibilidade sem ampliar desnecessariamente o escopo da feature.
- Analise possíveis preocupações de performance proporcionais ao escopo, como consultas repetidas ou sem filtros adequados, operações desnecessárias, processamento redundante, renderizações excessivas e crescimento não limitado de dados em memória.
- Corrija antes da entrega os problemas encontrados que pertençam ao escopo da feature e adicione ou ajuste testes quando houver comportamento relevante a proteger.
- Informe claramente na entrega qualquer risco, limitação ou oportunidade de melhoria identificada que não possa ser resolvida com segurança dentro do escopo atual.

## Git

- Não imponha Conventional Commits.
- Mantenha commits focados e não inclua alterações não relacionadas.
- O gate obrigatório de pull requests são os testes unitários aplicáveis. As verificações completas continuam obrigatórias antes de uma release.
