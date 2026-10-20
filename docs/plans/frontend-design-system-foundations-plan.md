# Fundações do design system em React

Criar os primeiros componentes reutilizáveis do frontend com APIs configuráveis, estilos isolados e tokens centralizados. O escopo inicial cobre somente os padrões já aprovados no protótipo; páginas reais, tabelas, modais, feedback e estados vazios ficam para etapas posteriores.

## Interfaces públicas

- Expor os componentes por `frontend/components/ui/index.ts`.
- `Button` e `IconButton` aceitarão conteúdo, callbacks, `disabled`, `aria-*`, `data-*`, `ref` e demais atributos nativos. `IconButton` exigirá nome acessível e aceitará estado `pressed`.
- `Input`, `Select` e controle com prefixo preservarão a API dos elementos HTML, incluindo eventos e refs.
- `Field`, `TextField`, `SelectField` e `PasswordField` receberão `label`, ajuda, erro e propriedades do controle. Os wrappers gerarão e associarão IDs, `aria-invalid` e `aria-describedby`.
- `PasswordField` controlará apenas a visibilidade, com mensagens configuráveis e callback opcional de mudança; valor e validação continuarão sob controle da página.

### 1. Centralizar os fundamentos visuais

- **O que:** criar tokens semânticos globais para cores, tipografia Montserrat, espaçamento, raios, sombras, foco e dimensões de controles; suportar temas claro e escuro por `data-theme`; manter estilos de cada componente em CSS Modules.
- **Motivação:** permitir mudanças visuais amplas sem alterar a marcação ou as páginas consumidoras.
- **Onde:** `frontend/app/globals.css`, novo diretório de estilos/tokens e layout raiz; adaptar a página provisória atual à nova paleta.
- **Base documental:** `docs/architecture.md`, seção “Frontend”, e `docs/implementation-scope.md`, item 4.
- **Verificação:** temas preservam contraste, foco visível, redução de movimento e controles com pelo menos 44 × 44 px.

### 2. Implementar os controles de ação

- **O que:** criar `Button` primário e `IconButton` com estados normal, hover, foco, pressionado e desabilitado, sem textos ou ações de negócio embutidos.
- **Motivação:** permitir que cada página forneça label, ícone, gesto e callback mantendo aparência e acessibilidade consistentes.
- **Onde:** `frontend/components/ui`.
- **Base documental:** estados de botões em `frontend/prototypes/design-system.html`.
- **Verificação:** testes Vitest comprovam propagação de atributos e refs, execução do callback, bloqueio quando desabilitado e semântica de `aria-pressed`.

### 3. Implementar as primitivas de formulário

- **O que:** criar `Input`, `Select` e composição de prefixo, aceitando integralmente atributos, eventos e refs nativos.
- **Motivação:** oferecer blocos substituíveis para casos especiais sem obrigar as páginas a usar wrappers de alto nível.
- **Onde:** `frontend/components/ui`.
- **Base documental:** campos, estados e validação do protótipo vigente.
- **Verificação:** testes cobrem alteração de valor, seleção, prefixo apenas visual, estados inválido/desabilitado e callbacks fornecidos pelo consumidor.
- **Dependências:** subtarefa 1.

### 4. Implementar campos acessíveis prontos para páginas

- **O que:** criar `Field`, `TextField`, `SelectField` e `PasswordField`, compondo label persistente, ajuda, erro e controles primitivos; reservar espaço para erro e configurar todas as mensagens por props.
- **Motivação:** reduzir repetição nas páginas sem impedir composições customizadas.
- **Onde:** `frontend/components/ui`.
- **Base documental:** regras de rótulo, erro associado, foco e senha no design system atual.
- **Verificação:** testes cobrem associação do label, IDs explícitos e gerados, descrição por ajuda/erro, `aria-invalid`, mudança de visibilidade da senha, labels configuráveis e preservação do valor.
- **Dependências:** subtarefas 2 e 3.

### 5. Substituir o catálogo estático pela referência React

- **O que:** criar `/design-system` usando os componentes reais, com exemplos de todos os estados, temas e layouts responsivos; disponibilizá-la somente em desenvolvimento e retornar `notFound` em produção. Após confirmar paridade, remover somente `design-system.html` e `design-system.css`; os protótipos de páginas permanecem.
- **Motivação:** impedir divergência entre a documentação visual e os componentes consumidos pelo produto.
- **Onde:** `frontend/app/design-system`, `frontend/prototypes`, `docs/architecture.md` e `AGENTS.md`.
- **Base documental:** decisão confirmada de tornar React a referência oficial; essa decisão substitui as menções atuais ao HTML estático.
- **Verificação:** revisão manual nos temas claro/escuro e larguras desktop/mobile, navegação integral por teclado e confirmação de indisponibilidade da rota em produção.
- **Dependências:** subtarefas 1–4.

### 6. Sincronizar checklist e validar a entrega

- **O que:** desmembrar o item amplo de fundações visuais em `docs/implementation-scope.md`, marcando apenas tokens, botões e formulários como entregues; manter tabelas, modais, feedback e estados vazios pendentes.
- **Motivação:** registrar com precisão o que existe sem declarar todo o design system concluído.
- **Onde:** `docs/implementation-scope.md`.
- **Base documental:** regras de documentação viva do `AGENTS.md`.
- **Verificação:** executar `npm test`, `npm run lint`, `npm run format:check`, `npm run typecheck` e `npm run build`; revisar o diff final, acessibilidade, responsividade e ausência de estilos duplicados.
- **Dependências:** subtarefas 1–5.

### 7. Isolar cada componente e adotar refs nativas do React 19

- **O que:** mover cada componente para uma pasta própria contendo implementação `.tsx`, teste `.test.tsx` e CSS Module; substituir `forwardRef` pela prop `ref` nativa do React 19, preservando a API pública pelo barrel `frontend/components/ui/index.ts`.
- **Motivação:** tornar componentes mais fáceis de localizar, alterar e substituir, alinhando a implementação ao modelo atual de refs do React.
- **Onde:** `frontend/components/ui`, catálogo React e `docs/architecture.md`.
- **Base documental:** convenções de frontend de `docs/architecture.md` e decisão explícita desta continuação do plano.
- **Verificação:** testes isolados por componente comprovam comportamento e encaminhamento de refs; executar `npm test`, `npm run lint`, `npm run format:check`, `npm run typecheck` e `npm run build`; confirmar ausência de `forwardRef` no design system.
- **Dependências:** subtarefas 1–6.

## Premissas

- Nenhuma página funcional, chamada de API ou regra de negócio será implementada nesta entrega.
- Não serão adicionados Storybook, bibliotecas de componentes, gerenciadores de estado ou novas dependências.
- O tema será escolhido pelo contêiner ou página via `data-theme`; não haverá seletor global nem persistência da preferência.
- Novos componentes compostos serão adicionados somente quando uma página real estabelecer seus requisitos.
