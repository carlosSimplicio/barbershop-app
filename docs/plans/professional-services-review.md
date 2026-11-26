# Revisão da implementação de serviços por profissional

- Data: 26/11/2026.
- Plano revisado: [professional-services-plan.md](professional-services-plan.md).
- Referência: implementação até o commit `a11c613`, comparada com a base `a9f74e2`.
- Escopo: bugs, lógica, concorrência, contratos, duplicações, arquitetura, legibilidade e performance.
- Método: inspeção de código, documentação, OpenAPI e testes, com reproduções adicionais em PostgreSQL descartável. A skill de revisão do projeto não foi utilizada.
- Situação: dois problemas de correção e uma oportunidade de performance identificados. As correções não foram implementadas nesta revisão.

## 1. [P2] `updated_at` pode retroceder em escritas concorrentes

**Local:** [backend/db/query/professional_services.sql](../../backend/db/query/professional_services.sql), expressão de `updated_at` em `SetProfessionalOffering`, linhas 29–30 na referência revisada.

### Problema e impacto

`statement_timestamp()` representa o início do comando SQL, antes de uma eventual espera pelo bloqueio da linha. Uma requisição pode começar, aguardar outra transição e, ao prosseguir, gravar uma data anterior àquela que já estava persistida.

O estado final do vínculo continua sendo atualizado, mas a data deixa de representar corretamente a sequência das transições. Essas datas ainda não são expostas pela resposta da API de vínculos; o defeito está nos metadados persistidos e compromete seu uso para acompanhamento temporal.

### Evidência

Uma reprodução temporária utilizou a query `sqlc` real para as duas transições:

1. Uma transação adquiriu o bloqueio de um vínculo existente.
2. Um PUT para associar o mesmo par iniciou o upsert e ficou aguardando esse bloqueio.
3. A primeira transação removeu a associação pela query real e confirmou a transação.
4. O PUT prosseguiu e reassociou o vínculo com um `updated_at` anterior ao da remoção.

Na execução observada, `updated_at` retrocedeu aproximadamente 3,5 ms. A espera foi coordenada pela observação do bloqueio em `pg_stat_activity`.

O teste concorrente existente em [professional_services_mutation_test.go](../../backend/integration/professional_services_mutation_test.go) protege identidade, unicidade e estados dos pares, mas não verifica a monotonicidade das datas.

### Recomendação

Calcular a data da transição após adquirir o bloqueio, preservando sua monotonicidade em relação ao valor persistido. Adicionar uma regressão concorrente com bloqueio coordenado que verifique a ordem das datas, além do estado final. Regenerar os artefatos `sqlc` após alterar a query.

## 2. [P3] A validação aceita propriedades fora do contrato JSON

**Locais:** [offerings/handler.go](../../backend/internal/professionals/offerings/handler.go), função `decodeAssociationRequest`, e [httpserver/json.go](../../backend/internal/platform/httpserver/json.go), função `DecodeJSONRequest`.

### Problema e impacto

O schema `ProfessionalServiceRequest` no [OpenAPI](../../backend/openapi/openapi.yaml) exige a propriedade `associated` e define `additionalProperties: false`. Entretanto, a decodificação para uma struct Go aceita variações de maiúsculas e minúsculas nos nomes dos campos, mesmo com `DisallowUnknownFields()`.

Assim, um corpo sem a chave exata exigida pelo contrato pode provocar uma alteração persistida, em vez de receber `400`.

### Evidência

Uma requisição HTTP real ao endpoint de mutação, com autenticação e CSRF válidos, enviou:

```json
{"Associated": true}
```

A API retornou `200`, com `associated: true`, e criou a associação. Pelo contrato documentado, esse corpo deveria ser recusado: `associated` está ausente e `Associated` é uma propriedade adicional.

### Recomendação

Validar as chaves exatas na borda HTTP, preservando as regras existentes de presença, nulabilidade, tipo e tamanho do corpo. Adicionar um teste de regressão do contrato HTTP para essa entrada e verificar os consumidores do decoder compartilhado caso ele seja alterado.

## 3. [P3 — performance] Repetições idempotentes regravam a linha

**Local:** [backend/db/query/professional_services.sql](../../backend/db/query/professional_services.sql), cláusula `ON CONFLICT DO UPDATE` de `SetProfessionalOffering`, linhas 27–30 na referência revisada.

### Problema e impacto

O `DO UPDATE` executa mesmo quando `active` já corresponde ao estado solicitado. O `CASE` preserva `updated_at`, mas não evita a atualização física da linha.

A idempotência observável é preservada, porém repetições geram novas versões da linha, WAL e trabalho de vacuum desnecessários. A relevância cresce com o volume de repetições e disputas pelo mesmo par.

### Evidência

Após criar um vínculo ativo, a reprodução repetiu o mesmo PUT com `associated: true`. O `ctid` mudou de `(0,1)` para `(0,2)`, confirmando a criação de uma nova versão física apesar da ausência de transição de estado.

### Recomendação

Condicionar o `DO UPDATE` à diferença de estado. Ajustar também o tratamento do resultado em [offerings/store.go](../../backend/internal/professionals/offerings/store.go), pois um conflito sem atualização pode deixar de produzir uma linha em `RETURNING`; isso deve continuar confirmando corretamente a operação sem mudança. Preservar a remoção de par ausente sem inserção e a segurança concorrente.

## Arquitetura, duplicação e legibilidade

- A separação do slice de vínculos está coerente com o plano e evita ampliar o store legado de profissionais e credenciais.
- A autorização permanece no backend, e as FKs compostas protegem as referências entre tenants. Não foram identificadas falhas de isolamento ou CSRF nos fluxos revisados.
- A leitura é paginada e não executa uma consulta por serviço. Não foi identificado um problema de N+1 no fluxo entregue.
- A repetição da conversão de registros para `Offering` no store é pequena e não justifica, por si só, uma abstração genérica.
- Como melhoria opcional de legibilidade, considerar separar a apresentação da extensa coordenação de estados em `ProfessionalServicesRequest`, no [painel de serviços](../../frontend/components/professionals/professional-services-panel/professional-services-panel.tsx). Preservar o controle de cancelamento, identidade, permissões e confirmação das mutações.

## Verificações executadas

Passaram durante a revisão:

- Testes unitários do backend: `go test ./...`.
- Análise estática: `go vet -tags=integration ./...`.
- Integração específica de vínculos: `go test -count=1 -tags=integration ./integration -run ProfessionalServices`.
- Integração dos fluxos de profissionais: `go test -count=1 -tags=integration ./integration -run 'TestProfessional'`.
- Testes do frontend: `npm test`, com 401 testes em 74 arquivos.
- Tipos, ESLint e Prettier: `npm run typecheck`, `npm run lint` e `npm run format:check`.

As sondagens adicionais reproduziram os achados 1 e 2 com falhas nas expectativas de comportamento correto e confirmaram a atualização física descrita no achado 3. Foram executadas somente no PostgreSQL descartável de Testcontainers e removidas após a revisão.

Não foram reexecutados os builds, a geração de artefatos, a suíte completa de integração do backend ou a validação no navegador. A revisão não modificou código de produção nem implementou correções.
