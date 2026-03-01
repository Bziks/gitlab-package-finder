package entity

type SearchPackage struct {
	Type    string
	Name    string
	Version string
	Status  string
}

type FailedRepository struct {
	ProjectID   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
	ProjectURL  string `json:"project_url"`
	BranchID    int    `json:"branch_id"`
	BranchName  string `json:"branch_name"`
	Error       string `json:"error"`
	Timestamp   int64  `json:"timestamp"`
}

const (
	SearchStatusPending    = "pending"
	SearchStatusProcessing = "processing"
	SearchStatusCompleted  = "completed"
	SearchStatusFailed     = "failed"
)
