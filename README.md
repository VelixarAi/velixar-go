# Velixar Go SDK

Go client for the [Velixar](https://velixarai.com) AI memory platform.

## Install

```bash
go get github.com/VelixarAi/velixar-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    velixar "github.com/VelixarAi/velixar-go"
)

func main() {
    client := velixar.New("your-api-key")
    ctx := context.Background()

    // Store a memory
    id, err := client.Store(ctx, "Go SDK is working",
        velixar.WithTags("test", "go"),
        velixar.WithTier(2),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Stored:", id)

    // Search memories
    results, err := client.Search(ctx, "Go SDK", velixar.WithLimit(5))
    if err != nil {
        log.Fatal(err)
    }
    for _, m := range results.Memories {
        fmt.Printf("  [%.2f] %s\n", m.Score, m.Content)
    }

    // Knowledge graph
    graph, err := client.GraphTraverse(ctx, "Go", 2)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Graph: %d nodes, %d edges\n", len(graph.Nodes), len(graph.Edges))
}
```

## Environment Variables

| Variable | Description |
|---|---|
| `VELIXAR_API_KEY` | API key (used if not passed to `New()`) |

## API

| Method | Description |
|---|---|
| `Store(ctx, content, opts...)` | Store a memory, returns ID |
| `Search(ctx, query, opts...)` | Semantic search |
| `Get(ctx, id)` | Get memory by ID |
| `Update(ctx, id, content, tags)` | Update memory |
| `Delete(ctx, id)` | Delete memory |
| `List(ctx, limit)` | List recent memories |
| `GraphTraverse(ctx, entity, depth)` | Walk knowledge graph |
| `Health(ctx)` | Check API status |

## Options

```go
// Store options
velixar.WithTags("tag1", "tag2")
velixar.WithTier(0)  // 0=pinned, 1=session, 2=semantic, 3=org
velixar.WithUserID("user-123")

// Search options
velixar.WithLimit(10)

// Client options
velixar.WithBaseURL("https://custom.api.com")
velixar.WithHTTPClient(&http.Client{Timeout: 60 * time.Second})
```

## License

MIT
