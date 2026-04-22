package service

import (
    "bytes"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "math/rand"
    "os"
    "regexp"
    "runtime"
    "strconv"
    "strings"
    "time"

    "github.com/attilamonteiro/deserialize-explosion/repository"
    "github.com/attilamonteiro/deserialize-explosion/model"
)

type Service struct {
    repo *repository.CacheRepo
}

func NewService(r *repository.CacheRepo) *Service {
    return &Service{repo: r}
}

func (s *Service) Repo() *repository.CacheRepo { return s.repo }

// Comportamento legado: se o blob mestre estiver ausente, gera todos os itens,
// salva um blob grande e, em cada request de página, descomprime+decodifica
// o blob inteiro e só então faz o slice solicitado.
func (s *Service) GetPageLegacy(page, pageSize, numItems, dbDelay int) ([]model.Recipe, error) {
    cacheKey := "catalogo_receitas"
    if b, ok := s.repo.Get(cacheKey); ok {
        // CACHE HIT (comportamento problemático): descomprime+decodifica todo o
        // blob mestre na memória e depois realiza o slice da página solicitada.
        // Isso reproduz o bug estrutural do sistema original, onde a paginação
        // não evitava desserializar todo o conjunto de dados.
        items, err := decompressAndDecodeAll(b)
        if err != nil {
            return nil, err
        }

        // Simula processamento caro por item (baseado em Regex) para reproduzir
        // a pressão de CPU/GC observada no incidente original. A intensidade é
        // configurável via `HEAVY_MULTIPLIER` para reproduzir cenários leves a
        // severos.
        heavy := getHeavyMultiplier()
        heavyProcessing(items, heavy)

        // Por fim, tira o slice da página solicitada da fatia já decodificada.
        from := (page - 1) * pageSize
        if from >= len(items) {
            return []model.Recipe{}, nil
        }
        to := min(from+pageSize, len(items))
        return items[from:to], nil
    }

    // cache miss: simula atraso do BD e gera todos os itens
    time.Sleep(time.Duration(dbDelay) * time.Second)
    items := make([]model.Recipe, 0, numItems)
    for i := 1; i <= numItems; i++ {
        items = append(items, makeRecipe(i))
    }
    // serializa tudo e comprime (blob mestre)
    blob := generateMasterBlob(items)
    s.repo.Set(cacheKey, blob)

    from := 0
    to := min(pageSize, len(items))
    return items[from:to], nil
}

// ---------------------------- Auxiliares ----------------------------

// getHeavyMultiplier lê a variável de ambiente HEAVY_MULTIPLIER e retorna
// um valor padrão (1). Controla quantas vezes o loop de processamento caro
// por item será executado para exagerar a pressão de CPU/GC para testes.
func getHeavyMultiplier() int {
    heavy := 1
    if v := os.Getenv("HEAVY_MULTIPLIER"); v != "" {
        if hv, err := strconv.Atoi(v); err == nil && hv > 0 {
            heavy = hv
        }
    }
    return heavy
}

// decompressAndDecodeAll descomprime um blob gzip e decodifica o array JSON
// completo para uma slice de model.Recipe. Isso encapsula o comportamento
// problemático de "decodificar-tudo" para facilitar a substituição por um
// decoder em streaming ou por página ao implementar a correção.
func decompressAndDecodeAll(b []byte) ([]model.Recipe, error) {
    gr, err := gzip.NewReader(bytes.NewReader(b))
    if err != nil {
        return nil, err
    }
    var items []model.Recipe
    if err := json.NewDecoder(gr).Decode(&items); err != nil {
        _ = gr.Close()
        return nil, err
    }
    _ = gr.Close()
    return items, nil
}

// heavyProcessing executa o trabalho caro por item que reproduz a sobrecarga
// do sistema original (uso de Regex.Replace milhares de vezes). Mantê-lo em
// função separada facilita a substituição por uma implementação sem alocações
// posteriormente.
func heavyProcessing(items []model.Recipe, heavy int) {
    if heavy <= 0 || len(items) == 0 {
        return
    }
    cpfRe := regexp.MustCompile("\\D")
    for k := 0; k < heavy; k++ {
        for i := range items {
            _ = cpfRe.ReplaceAllString(items[i].Meta["code"], "")
        }
    }
}

// generateMasterBlob serializa a slice completa e retorna um blob gzipped.
// Mantém a lógica de "criar blob grande" em um único lugar para maior clareza.
func generateMasterBlob(items []model.Recipe) []byte {
    b, _ := json.Marshal(items)
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    _, _ = gw.Write(b)
    _ = gw.Close()
    return buf.Bytes()
}

func makeRecipe(i int) model.Recipe {
    // Torna os itens maiores e mais aninhados para emular grafos de objetos
    // mais pesados
    longNotes := strings.Repeat("X", 4000) // string grande por item
    item := model.Recipe{
        ID:    i,
        Title: fmt.Sprintf("Receita %d", i),
        Info: model.Info{
            Calories: rand.Float64()*800 + 100,
            Notes:    longNotes,
        },
        Meta: map[string]string{"tags": fakeTags(i), "code": fakeCode(i)},
    }
    stepCount := 50
    item.Steps = make([]model.Step, 0, stepCount)
    for j := 0; j < stepCount; j++ {
        desc := strings.Repeat("step-desc-", 20) + fmt.Sprintf(" %d of %d", j+1, stepCount)
        item.Steps = append(item.Steps, model.Step{Index: j, Description: desc})
    }
    return item
}

func fakeTags(i int) string {
    pool := []string{"vegan", "gluten-free", "dessert", "quick", "spicy", "low-carb"}
    t1 := pool[i%len(pool)]
    t2 := pool[(i/3)%len(pool)]
    return t1 + "," + t2
}

func fakeCode(i int) string {
    return fmt.Sprintf("%03d.%03d.%03d-%02d", i%1000, (i/1000)%1000, (i/1000000)%1000, i%100)
}

// Bench: executa experimentos e retorna um mapa com métricas
func (s *Service) Bench(num, pageSize, dbDelay int) (map[string]interface{}, error) {
    report := map[string]interface{}{}
    cacheKey := "catalogo_receitas"

    // 1) Save (ruim): serializa tudo e salva o blob mestre
    time.Sleep(time.Duration(dbDelay) * time.Second)
    items := make([]model.Recipe, 0, num)
    for i := 1; i <= num; i++ {
        items = append(items, makeRecipe(i))
    }
    t0 := time.Now()
    b, _ := json.Marshal(items)
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    _, _ = gw.Write(b)
    _ = gw.Close()
    s.repo.Set(cacheKey, buf.Bytes())
    report["save_bad_ms"] = time.Since(t0).Milliseconds()

    dDur, dAlloc, _ := measureDecompressAndDecode(buf.Bytes())
    report["decode_bad_ms"] = dDur.Milliseconds()
    report["decode_bad_alloc_bytes"] = dAlloc

    // 2) Save em streaming: escreve itens um a um no gzip
    t1 := time.Now()
    var buf2 bytes.Buffer
    gw2 := gzip.NewWriter(&buf2)
    _, _ = gw2.Write([]byte("["))
    for i := 1; i <= num; i++ {
        it := makeRecipe(i)
        bb, _ := json.Marshal(it)
        if i > 1 {
            _, _ = gw2.Write([]byte(","))
        }
        _, _ = gw2.Write(bb)
    }
    _, _ = gw2.Write([]byte("]"))
    _ = gw2.Close()
    s.repo.Set(cacheKey+":stream", buf2.Bytes())
    report["save_stream_ms"] = time.Since(t1).Milliseconds()
    dDur2, dAlloc2, _ := measureDecompressAndDecode(buf2.Bytes())
    report["decode_stream_ms"] = dDur2.Milliseconds()
    report["decode_stream_alloc_bytes"] = dAlloc2

    // 3) Construção de cache por página (per-page)
    // limpa o repositório
    s.repo.Clear()
    t2 := time.Now()
    pages := (num + pageSize - 1) / pageSize
    for p := 0; p < pages; p++ {
        from := p*pageSize + 1
        to := min(from+pageSize-1, num)
        var bbuf bytes.Buffer
        gw3 := gzip.NewWriter(&bbuf)
        _, _ = gw3.Write([]byte("["))
        for i := from; i <= to; i++ {
            it := makeRecipe(i)
            bb, _ := json.Marshal(it)
            if i > from {
                _, _ = gw3.Write([]byte(","))
            }
            _, _ = gw3.Write(bb)
        }
        _, _ = gw3.Write([]byte("]"))
        _ = gw3.Close()
        key := fmt.Sprintf("%s:page:%d", cacheKey, p+1)
        s.repo.Set(key, bbuf.Bytes())
    }
    report["perpage_build_ms"] = time.Since(t2).Milliseconds()

    // servir a página 2 (exercício de leitura por página)
    keyp2 := fmt.Sprintf("%s:page:%d", cacheKey, 2)
    if blob, ok := s.repo.Get(keyp2); ok {
        start := time.Now()
        gr, _ := gzip.NewReader(bytes.NewReader(blob))
        var pageItems []model.Recipe
        _ = json.NewDecoder(gr).Decode(&pageItems)
        _ = gr.Close()
        report["perpage_serve_ms"] = time.Since(start).Milliseconds()
        report["perpage_page_items"] = len(pageItems)
    }

    runtime.GC()
    return report, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func measureDecompressAndDecode(blob []byte) (time.Duration, uint64, error) {
    var msBefore, msAfter runtime.MemStats
    runtime.ReadMemStats(&msBefore)
    start := time.Now()
    gr, err := gzip.NewReader(bytes.NewReader(blob))
    if err != nil {
        return 0, 0, err
    }
    var items []model.Recipe
    if err := json.NewDecoder(gr).Decode(&items); err != nil {
        _ = gr.Close()
        return 0, 0, err
    }
    _ = gr.Close()
    dur := time.Since(start)
    runtime.ReadMemStats(&msAfter)
    alloc := msAfter.TotalAlloc - msBefore.TotalAlloc
    items = nil
    return dur, alloc, nil
}
