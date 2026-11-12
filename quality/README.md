# Regras de qualidade

Tudo o que barra código ruim no InfraDesktop mora aqui. As únicas exceções são as regras ESLint próprias, em `frontend/quality/`, porque precisam do `node_modules` do frontend.

- **Uma entrada:** `make quality` roda `quality/run.sh`, que executa todas as verificações e termina com uma tabela de id, status e tempo. O hook de pre-commit e o CI chamam o mesmo alvo.
- **Lista fixa de exceções:** dívida antiga fica congelada em `quality/baselines/quality-baseline.json` e `frontend/quality/quality-ratchet.json`. Não cresce; código novo ou tocado sai dentro dos limites.

## Pastas

| Pasta | Conteúdo |
|---|---|
| `go/` | `quality.go` (medição e regras Go); `qualityrules/` (limites, alterações); `qualitymetrics/` (medidor) |
| `architecture/` | Guards de arquitetura Go e frontend (`architecture.go`, `architecture-frontend.mjs`) e de estrutura do backend (`backend-layout.mjs`) |
| `layout/` | Guard das telas (`layout-guard.mjs`) |
| `checks/` | Verificações avulsas: versão única e domínio de rádio no frontend |
| `baselines/` | Lista fixa de exceções Go/TS (`quality-baseline.json`) |
| `../frontend/quality/` | Regras ESLint próprias e lista fixa de exceções do frontend |

## Catálogo

| Id | O que mede | Limite | Onde roda | Como corrigir |
|---|---|---|---|---|
| GO-FMT | Go formatado | `gofmt -l` vazio | `make quality` | `gofmt -w <arquivo>` |
| GO-VET | Erros comuns de Go | zero achado | `make quality` | corrigir o achado; nunca silenciar |
| GO-STATIC | Análise estática (código morto, uso errado de API) | zero achado | `make quality` | corrigir; código sem uso sai |
| FE-FORMAT | Frontend formatado | Prettier limpo | `make quality` | `npx prettier --write <arquivo>` |
| VERSION | Uma única fonte de versão | `VERSION` igual em pacote, Wails e npm | `make quality` | mudar só `VERSION` |
| BE-LAYOUT | Arquivos rastreados em `internal/` fora de `testdata/` e `migrations/` | Apenas `.go` | `make quality` | mover o arquivo para o diretório adequado |
| QG1 | Arquitetura: domínio sem banco, SSH, Wails ou rede; frontend em camadas iguais às dos módulos Go (`access` < `equipment` < `inventory`/`monitoring` < `pppoe` < `radio`); só `app/` importa `app/`; `shared/` folha; bindings só em `platform/wails/` | zero violação | `make check` (`architecture-check`) | mover o arquivo para a camada certa, ou compor em `app/pages/` |
| QG2 | Tamanho | função ≤ 100 linhas (120 em testes TS); arquivo Go ≤ 500, `.tsx` ≤ 600, frontend ≤ 500; linha ≤ 140 | `make quality` (QG2-QG5, FE-LINT) | dividir por regra do domínio ou da tela, nunca em pedaços sem nome |
| QG3 | Complexidade | ≤ 15 por função tocada | idem | extrair a regra com nome; mapa no lugar de ternário aninhado |
| QG4 | Literal de domínio (papel, família, modo, código) fora do pacote dono | proibido | QG2-QG5 (Go), FE-LINT (`radio-domain`) | usar a constante do dono |
| QG5 | Nome de teste Go por comportamento, nunca por slice (`L0x…`, `BF0x`, `Focal`, `Piloto`, `Avulso`, `Live2`, `CPU90`) | proibido em teste tocado | QG2-QG5 | renomear pelo que o teste prova |
| FE-LINT | ESLint do frontend: limites, kit de UI, sem `eslint-disable` | zero aviso | `make quality` | seguir a mensagem; o kit fica em `src/shared/ui/` |
| QG9 | Telas âncora fiéis ao wireframe congelado, só com o kit | guard e 10 âncoras em `make test-layout` | QG9 (`make quality`) e `make test-layout` (local, antes do push) | ver a skill de layout |
