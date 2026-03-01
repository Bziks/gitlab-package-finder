# Projects Sync Flow

The projects sync is a periodic background job that synchronizes GitLab projects, their default branches, and detected package types into the local database.

## Overview

```mermaid
flowchart LR
    Timer["Interval Timer<br>(default: 1h)"] --> Worker
    Worker --> Sync["Projects Sync Command"]
    Sync --> GitLab["GitLab API"]
    Sync --> DB["MySQL"]
```

## Worker Lifecycle

The sync command runs inside a worker that handles scheduling, panic recovery, and graceful shutdown.

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Executing: Interval elapsed
    Executing --> Idle: Success
    Executing --> Idle: Error (logged)
    Executing --> Idle: Panic (recovered & logged)
    Idle --> [*]: Context cancelled
    Executing --> ShuttingDown: Context cancelled
    ShuttingDown --> [*]: Shutdown delay elapsed
```

## Full Sync Sequence

```mermaid
sequenceDiagram
    participant W as Worker
    participant C as Sync Command
    participant GL as GitLab API
    participant PS as Project Service
    participant DB as MySQL
    participant Cache as In-Memory Cache

    W->>C: Execute(ctx)

    C->>GL: ListProjects(page=1, perPage=100, membership=true)
    GL-->>C: Projects + TotalPages + TotalItems

    Note over C: Log "Total items: X, Total pages: Y"

    par Process pages concurrently (max 2)
        C->>C: checkProjectsChunk(page 1)
        C->>C: checkProjectsChunk(page 2)
    end

    Note over C: Semaphore limits concurrency to 2 goroutines

    loop For each page
        C->>GL: ListProjects(page=N, perPage=100)
        GL-->>C: Up to 100 projects

        loop For each project
            C->>PS: UpsertWithDefaultBranch(project)
            PS->>DB: INSERT projects ... ON DUPLICATE KEY UPDATE
            PS->>DB: Find branch by (projectID, branchName)
            alt Branch not found
                PS->>DB: INSERT project_branches
            end

            C->>GL: GetProjectLanguages(projectID)
            GL-->>C: {"Go": 45.5, "JavaScript": 30.2}

            C->>C: Map languages to package types<br>via languageMap

            C->>PS: SyncPackageTypes(projectID, typeNames)
            loop For each type name
                PS->>Cache: GetByName(typeName)
                alt Cache miss
                    Cache->>DB: SELECT * FROM package_types<br>WHERE name = ?
                    DB-->>Cache: PackageType
                end
                Cache-->>PS: PackageType (with ID)
            end
            PS->>DB: BEGIN TRANSACTION
            PS->>DB: DELETE FROM project_package_types<br>WHERE project_id = ?
            PS->>DB: INSERT INTO project_package_types<br>(project_id, package_type_id) VALUES ...
            PS->>DB: COMMIT
        end
    end

    C-->>W: Done
```

## Concurrent Page Processing

Pages are processed in parallel with a semaphore limiting concurrency to 2 goroutines at a time.

```mermaid
flowchart TD
    Start["ListProjects(page=1)<br>Get TotalPages"] --> Spawn

    Spawn --> P1["Page 1"]
    Spawn --> P2["Page 2"]
    Spawn --> P3["Page 3"]
    Spawn --> P4["Page 4"]
    Spawn --> PN["Page N"]

    subgraph "Semaphore (capacity: 2)"
        P1 --> Sem["Concurrent Slot"]
        P2 --> Sem
    end

    P3 -.->|Waits for slot| Sem
    P4 -.->|Waits for slot| Sem
    PN -.->|Waits for slot| Sem

    Sem --> Done["WaitGroup.Wait()<br>All pages processed"]
```

## Language to Package Type Mapping

The system uses a factory pattern to map GitLab-reported programming languages to package manager types.

```mermaid
flowchart LR
    subgraph "GitLab Languages Response"
        GL["{ 'Go': 45.5,<br>'JavaScript': 30.2,<br>'PHP': 24.3 }"]
    end

    subgraph "Language Map (from Factory)"
        GL --> |lowercase| LM
        LM["languageMap"]
        LM --> |go| Go["go"]
        LM --> |javascript| JS["npm"]
        LM --> |typescript| TS["npm"]
        LM --> |php| PHP["composer"]
    end

    subgraph "Package Managers"
        Go --> GoMgr["Go Package Manager"]
        JS --> NpmMgr["NPM Package Manager"]
        TS --> NpmMgr
        PHP --> CompMgr["Composer Package Manager"]
    end

    subgraph "Result"
        GoMgr --> R["packageTypeNames:<br>[go, npm, composer]"]
        NpmMgr --> R
        CompMgr --> R
    end
```

## Per-Project Processing

```mermaid
flowchart TD
    P["GitLab Project"] --> Upsert["Upsert project in DB"]
    Upsert --> Branch{"Default branch<br>exists in DB?"}
    Branch -->|Yes| Lang["Fetch languages from GitLab"]
    Branch -->|No| CreateBranch["Create branch record"] --> Lang

    Lang --> Map["Map languages → package types"]
    Map --> Sync["Sync package types"]

    Sync --> TX["Transaction"]
    TX --> Del["DELETE old project_package_types"]
    Del --> Ins["INSERT new project_package_types"]

    Upsert -->|Error| Warn1["Log warning, continue"]
    Lang -->|Error| Warn2["Log warning, skip to next project"]
    Sync -->|Error| Warn3["Log warning, continue"]
```
