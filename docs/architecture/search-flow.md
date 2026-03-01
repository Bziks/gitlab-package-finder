# Package Search Flow

The search flow consists of two phases: **search initiation** (starting a new search) and **results retrieval** (polling for results).

## Overview

```mermaid
flowchart LR
    Client -->|1. Start search| API
    API -->|2. searchID| Client
    Client -->|3. Poll results| API
    API -->|4. Paginated results| Client
```

## Phase 1: Search Initiation

When a client sends a search request with package type and query, the system validates the input, checks for existing results, and queues projects for scanning.

```mermaid
sequenceDiagram
    participant C as Client
    participant H as HTTP Handler<br>(InternalSearchPackages)
    participant S as Search Service<br>(StartSearch)
    participant R as Redis
    participant DB as MySQL

    C->>H: GET /internal/search/packages<br>?type=npm&query=lodash
    H->>H: Validate type & query params

    H->>S: StartSearch(type, query)
    S->>S: ParseQuery(query)<br>Extract name + version
    S->>S: GenerateSearchID(type, name, version)

    S->>R: AcquireSearch(searchID)<br>HSetNX search:packages:{id}<br>status = "pending"

    alt Search already acquired by another request
        R-->>S: Already exists
        S-->>H: Return existing searchID
    else Search acquired successfully
        R-->>S: OK

        S->>DB: FindDependencies(type, name)
        alt Results exist in DB (cached)
            DB-->>S: Dependencies found
            S->>R: AddSearchToQueue(searchID)<br>Mark status = "completed"
            S-->>H: Return searchID
        else No cached results
            DB-->>S: Empty
            S->>S: addProjectsToQueue()
            Note over S: See queue population diagram
            S->>R: AddSearchToQueue(searchID)
            S-->>H: Return searchID
        end
    end

    H-->>C: 200 OK { searchId: "abc123" }
```

## Queue Population

When no cached results exist, the system populates a Redis queue with all projects that support the requested package type.

```mermaid
sequenceDiagram
    participant S as Search Service
    participant DB as MySQL
    participant R as Redis

    loop Paginated fetch (50 per page)
        S->>DB: GetDetailedByPackageType<br>WithPagination(type, page, 50)
        DB-->>S: Projects with branches
    end

    loop For each project
        loop For each branch
            S->>R: LPush search:projects:{searchID}<br>{project_id, branch_id, ...}
        end
    end

    Note over R: Queue expires in 1 hour

    S->>R: HSet search:packages:{searchID}<br>type, name, version, status
    S->>R: ZIncrBy search:packages:scores<br>searchID, 1.0

    Note over R: Search metadata expires in 24 hours
```

## Phase 2: Results Retrieval

The client polls for results using the searchID. The response includes current search status and paginated dependencies.

```mermaid
sequenceDiagram
    participant C as Client
    participant H as HTTP Handler<br>(InternalGetPackagesBySearch)
    participant P as Package Service<br>(GetPackagesBySearch)
    participant R as Redis
    participant DB as MySQL

    C->>H: GET /internal/search/{searchId}/packages<br>?page=1
    H->>P: GetPackagesBySearch(searchID, page)

    P->>R: CheckIfSearchIsRunning(searchID)<br>HGet status
    R-->>P: status (pending/processing/completed)

    P->>R: GetSearchDetails(searchID)<br>HGetAll search:packages:{id}
    R-->>P: {type, name, version}

    P->>DB: FindDependencies(page, type, name)<br>LIMIT 10 OFFSET (page-1)*10
    DB-->>P: Dependencies + total count

    P->>R: GetProjectsQueueLength(searchID)<br>LLen search:projects:{id}
    R-->>P: Remaining project count

    P->>R: GetSearchStatus(searchID)
    R-->>P: Status string

    P-->>H: Result{Dependencies, SearchFinished,<br>Total, RepositoriesLeft, Status}

    H-->>C: 200 OK<br>{packages, searchFinished,<br>repositoriesLeft, status, pagination}
```

## Redis Key Structure

```mermaid
flowchart TD
    subgraph "Search Metadata"
        H["search:packages:{searchID}<br>(Hash, TTL: 24h)"]
        H --> F1["type: npm"]
        H --> F2["name: lodash"]
        H --> F3["version: ^4.0.0"]
        H --> F4["status: pending|processing|completed|failed"]
    end

    subgraph "Project Queue"
        L["search:projects:{searchID}<br>(List, TTL: 1h)"]
        L --> I1["{ project_id, project_name,<br>branch_id, branch_name }"]
        L --> I2["..."]
    end

    subgraph "Search Priority"
        Z["search:packages:scores<br>(Sorted Set)"]
        Z --> S1["searchID_1 → score: 1.0"]
        Z --> S2["searchID_2 → score: 2.0"]
    end
```

## Search Deduplication

Identical searches (same type + name + version) produce the same `searchID`, which allows deduplication via Redis atomic `HSetNX`.

```mermaid
flowchart TD
    R1[Request: npm + lodash@^4] -->|GenerateSearchID| ID["searchID: abc123"]
    R2[Request: npm + lodash@^4] -->|GenerateSearchID| ID

    ID -->|HSetNX| Redis{Redis}
    Redis -->|First request wins| W[Search worker processes]
    Redis -->|Subsequent requests| E[Return same searchID]
```
