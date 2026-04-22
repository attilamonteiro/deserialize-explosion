# projeto0 — Simulação: "A Armadilha da Paginação de Dados Massivos"

![Owner avatar](assets/owner-avatar.svg)

Visão geral

- Repositório para reproduzir um cenário clássico: cache monolítico + desserialização
  total + processamento caro por item que causa latência elevada, picos de GC/CPU e
  possíveis travamentos.
-- Propósito: ambiente para você praticar correções (streaming, cache por página,
  redução de alocações) sem impactar sistemas reais.

Requisitos

- Go 1.20+ instalado

- Sistema com RAM suficiente para testes grandes (atenção: execuções muito grandes
  podem travar sua máquina)

Como executar (exemplos)

- PowerShell (Windows):

```powershell
Set-Location 'C:\Users\atmal\Documents\estudos\projeto0'
$env:NUM_ITEMS="50000"
$env:DB_DELAY_SECONDS="30"
$env:PAGE_SIZE="500"
$env:HEAVY_MULTIPLIER="50"
go run main.go
```

- Unix shell (Linux/macOS):

```bash
cd ~/path/to/projeto0
export NUM_ITEMS=50000
export DB_DELAY_SECONDS=30
export PAGE_SIZE=500
export HEAVY_MULTIPLIER=50
go run main.go
```

Variáveis de ambiente relevantes

- `NUM_ITEMS` (default ~50000): quantos itens a geração simula

- `DB_DELAY_SECONDS` (default 30): atraso simulado na geração (simula consulta pesada ao BD)

- `PAGE_SIZE` (default 500): tamanho lógico da página

- `HEAVY_MULTIPLIER` (default 1): multiplica o trabalho caro por item (Regex loop)

Endpoints

- `GET /receitas` — endpoint que reproduz o problema (desserializa todo o blob e só
  então aplica `skip/take`).

  - Query params: `page` (1), `pageSize`, `num`, `dbDelay`

  - Exemplo: `http://localhost:8080/receitas?page=1&pageSize=500&num=50000&dbDelay=30`

- `GET /bench` — executa três cenários e retorna métricas:

  1) "bad" — salva o blob inteiro (bolo) e mede descompressão/desserialização

  2) "stream" — salva em streaming (escreve itens um-a-um no gzip)

  3) "per-page" — constrói cache por página e mede tempo de servir uma página

  - Query params: `num`, `pageSize`, `dbDelay`

  - Exemplo: `http://localhost:8080/bench?num=1000&pageSize=100&dbDelay=2`

- `GET /create_perpage_cache` — constrói o cache por página (útil para comparação). Query: `num`, `pageSize`.

- `GET /clear-cache` — limpa o cache em memória.

Como reproduzir seguro vs perigoso

- Seguro (recomendado para machine local): use `NUM_ITEMS=1000..5000`, `HEAVY_MULTIPLIER=1..5`.

- Perigoso (pode consumir toda a RAM / travar): `NUM_ITEMS=50000+` combinado com `HEAVY_MULTIPLIER>=20`.

- Dica: aumente `NUM_ITEMS` e `HEAVY_MULTIPLIER` gradualmente ao invés de pular direto para os valores extremos.

O que exatamente está sendo simulado / onde olhar no código

- Paginação ineficiente (desserializar tudo e só depois `slice`): função `GetPageLegacy` em [service/service.go](service/service.go#L1-L220) — a chamada problemática está encapsulada em `decompressAndDecodeAll`.

- Criação do "bolo único" (master blob): `generateMasterBlob` em [service/service.go](service/service.go#L220-L300).

- Processamento caro por item (Regex loop): `heavyProcessing` e controle por `HEAVY_MULTIPLIER` em [service/service.go](service.service.go#L120-L200).

- Endpoints HTTP: ver [controller/controller.go](controller/controller.go#L1-L200).

Interpretação rápida das métricas do `/bench`
- `save_bad_ms` / `decode_bad_ms` / `decode_bad_alloc_bytes`: custo de salvar e desserializar o blob inteiro.
- `save_stream_ms` / `decode_stream_ms` / `decode_stream_alloc_bytes`: custo do save em streaming e desserialização do mesmo blob (geralmente menores alocações durante escrita; decode pode se comportar similarmente se o cliente ainda decodifica tudo).
- `perpage_build_ms` / `perpage_serve_ms`: tempo para construir caches por página e tempo para servir uma página a partir do cache por página (espera-se <100ms em cargas grandes).

Recomendações de correção (próxima etapa para praticar)
1. Implementar cache por página (per-page caching): construir e armazenar blobs por página (chaves: `catalogo_receitas:page:X`). Servir uma página apenas descomprime/desserializa o blob daquela página.
2. Usar serialização em streaming quando criar blobs grandes (escrever JSON diretamente no `gzip.Writer` sem alocar uma string/[]byte intermédia).
3. Substituir uso de `regexp` no loop por uma função `CleanCpf` sem alocações (usar `[]rune`/`[]byte` ou `strings.Builder`/`bytes` com padrões eficientes). No repositório, `heavyProcessing` isola este ponto para fácil substituição.
4. No frontend (se aplicável): evitar `ref([])`/`reactive([])` e usar `shallowRef`/técnicas que evitem criar proxies recursivos para listas massivas.

Sugestões de testes
- Roda `bench` com `num=1000` e compare `decode_bad_alloc_bytes` vs `decode_stream_alloc_bytes`.
- Crie o cache por página e então faça `GET /receitas?page=2` várias vezes para verificar latência vs desserializar todo o blob.

Arquivo(s) relevantes
- [service/service.go](service/service.go)
- [controller/controller.go](controller/controller.go)
- [repository/repository.go](repository/repository.go)
- [model/models.go](model/models.go)

Avisos de segurança
- Execuções com valores extremos podem consumir muita memória e travar o sistema. Use valores conservadores para experimentos exploratórios.

Próximos passos sugeridos
- Posso adicionar uma rota alternativa `GET /receitas_fixed` que serve por página (não altera comportamento atual) para comparações lado-a-lado.
- Posso implementar `CleanCpf` sem regex e medir ganhos.

Contato
- Se quiser que eu implemente qualquer uma das correções acima, me diga qual (por exemplo: "adicione rota fixed" ou "troque regex por CleanCpf").
