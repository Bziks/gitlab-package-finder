package packagesearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
)

const (
	searchTTL = 24 * time.Hour
	queueTTL  = 24 * time.Hour
	failedTTL = 24 * time.Hour
)

type projectQueueItem struct {
	ProjectID   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
	ProjectURL  string `json:"project_url"`
	BranchID    int    `json:"branch_id"`
	BranchName  string `json:"branch_name"`
}

type Storage struct {
	redisClient          *redis.Client
	searchKey            string
	searchScoreKey       string
	searchProjectsKey    string
	searchFailedReposKey string
}

func New(redisClient *redis.Client) *Storage {
	return &Storage{
		redisClient:          redisClient,
		searchKey:            "search:packages",
		searchScoreKey:       "search:packages:scores",
		searchProjectsKey:    "search:projects",
		searchFailedReposKey: "search:failed:repos",
	}
}

func (s *Storage) CheckIfSearchIsRunning(ctx context.Context, searchID string) (bool, error) {
	const op = "check if search is running"

	status, err := s.GetSearchStatus(ctx, searchID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if status == entity.SearchStatusPending || status == entity.SearchStatusProcessing {
		return true, nil
	}

	return false, nil
}

// AcquireSearch atomically tries to claim a search by setting the status field
// only if the hash key does not already exist. Returns true if this caller acquired it.
func (s *Storage) AcquireSearch(ctx context.Context, searchID string) (bool, error) {
	const op = "acquire search"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	pipe := s.redisClient.TxPipeline()
	acquireCmd := pipe.HSetNX(ctx, searchHashKey, "status", entity.SearchStatusPending)
	pipe.Expire(ctx, searchHashKey, searchTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return acquireCmd.Val(), nil
}

func (s *Storage) AddSearchToQueue(ctx context.Context, searchID string, searchPackage entity.SearchPackage) error {
	const op = "add search to queue"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	pipe := s.redisClient.TxPipeline()

	pipe.HSet(ctx, searchHashKey, map[string]interface{}{
		"type":    searchPackage.Type,
		"name":    searchPackage.Name,
		"version": searchPackage.Version,
		"status":  entity.SearchStatusPending,
	})

	pipe.Expire(ctx, searchHashKey, searchTTL)

	pipe.ZIncrBy(ctx, s.searchScoreKey, 1.0, searchID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetSearchFromQueue(ctx context.Context) (string, *entity.SearchPackage, error) {
	const op = "get search from queue"

	val, err := s.redisClient.ZRevRange(ctx, s.searchScoreKey, 0, 0).Result()
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(val) == 0 {
		return "", nil, nil
	}

	p, err := s.GetSearchDetails(ctx, val[0])
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", op, err)
	}

	return val[0], p, nil
}

func (s *Storage) AddProjectsToQueue(ctx context.Context, searchID string, project entity.DetailedProject) error {
	const op = "add projects to queue"

	queueKey := fmt.Sprintf("%s:%s", s.searchProjectsKey, searchID)

	pipe := s.redisClient.TxPipeline()

	for _, branch := range project.Branches {
		item := projectQueueItem{
			ProjectID:   project.ID,
			ProjectName: project.Name,
			ProjectURL:  project.URL,
			BranchID:    branch.ID,
			BranchName:  branch.Name,
		}

		jsonData, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("%s: failed to marshal project queue item: %w", op, err)
		}

		pipe.LPush(ctx, queueKey, jsonData)
	}

	pipe.Expire(ctx, queueKey, queueTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: failed to execute Redis pipeline: %w", op, err)
	}

	return nil
}

func (s *Storage) GetProjectFromQueue(ctx context.Context, searchID string) (*entity.DetailedProject, error) {
	const op = "get project from queue"

	queueKey := fmt.Sprintf("%s:%s", s.searchProjectsKey, searchID)

	result, err := s.redisClient.RPop(ctx, queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: failed to pop from Redis list: %w", op, err)
	}

	var item projectQueueItem
	err = json.Unmarshal([]byte(result), &item)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to unmarshal project queue item: %w", op, err)
	}

	return &entity.DetailedProject{
		Project: entity.Project{
			ID:   item.ProjectID,
			Name: item.ProjectName,
			URL:  item.ProjectURL,
		},
		Branches: []entity.Branch{
			{
				ID:   item.BranchID,
				Name: item.BranchName,
			},
		},
	}, nil
}

func (s *Storage) GetProjectsQueueLength(ctx context.Context, searchID string) (int64, error) {
	const op = "get queue length"

	queueKey := fmt.Sprintf("%s:%s", s.searchProjectsKey, searchID)

	length, err := s.redisClient.LLen(ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get queue length: %w", op, err)
	}

	return length, nil
}

func (s *Storage) UpdateSearchStatus(ctx context.Context, searchID string, status string) error {
	const op = "update search status"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	err := s.redisClient.HSet(ctx, searchHashKey, "status", status).Err()
	if err != nil {
		return fmt.Errorf("%s: failed to update status: %w", op, err)
	}

	return nil
}

func (s *Storage) GetSearchStatus(ctx context.Context, searchID string) (string, error) {
	const op = "get search status"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	status, err := s.redisClient.HGet(ctx, searchHashKey, "status").Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return status, nil
}

func (s *Storage) GetSearchDetails(ctx context.Context, searchID string) (*entity.SearchPackage, error) {
	const op = "get search details"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	fields, err := s.redisClient.HGetAll(ctx, searchHashKey).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("%s: search not found: %s", op, searchID)
	}

	name, ok := fields["name"]
	if !ok || name == "" {
		return nil, fmt.Errorf("%s: missing required field 'name' for search %s", op, searchID)
	}

	return &entity.SearchPackage{
		Type:    fields["type"],
		Name:    name,
		Version: fields["version"],
		Status:  fields["status"],
	}, nil
}

func (s *Storage) CompleteSearch(ctx context.Context, searchID string) error {
	const op = "complete search"

	searchHashKey := fmt.Sprintf("%s:%s", s.searchKey, searchID)

	pipe := s.redisClient.TxPipeline()
	pipe.ZRem(ctx, s.searchScoreKey, searchID)
	pipe.HSet(ctx, searchHashKey, "status", entity.SearchStatusCompleted)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) AddFailedRepository(ctx context.Context, searchID string, project *entity.DetailedProject, branch entity.Branch, errorMsg string) error {
	const op = "add failed repository"

	failedReposKey := fmt.Sprintf("%s:%s", s.searchFailedReposKey, searchID)

	failedRepo := entity.FailedRepository{
		ProjectID:   project.ID,
		ProjectName: project.Name,
		ProjectURL:  project.URL,
		BranchID:    branch.ID,
		BranchName:  branch.Name,
		Error:       errorMsg,
		Timestamp:   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(failedRepo)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal failed repository: %w", op, err)
	}

	pipe := s.redisClient.TxPipeline()

	pipe.LPush(ctx, failedReposKey, jsonData)

	pipe.Expire(ctx, failedReposKey, failedTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: failed to execute Redis pipeline: %w", op, err)
	}

	return nil
}

func (s *Storage) GetFailedRepositories(ctx context.Context, searchID string) ([]entity.FailedRepository, error) {
	const op = "get failed repositories"

	failedReposKey := fmt.Sprintf("%s:%s", s.searchFailedReposKey, searchID)

	results, err := s.redisClient.LRange(ctx, failedReposKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []entity.FailedRepository{}, nil
		}
		return nil, fmt.Errorf("%s: failed to get from Redis list: %w", op, err)
	}

	if len(results) == 0 {
		return []entity.FailedRepository{}, nil
	}

	failedRepos := make([]entity.FailedRepository, 0, len(results))
	for _, result := range results {
		var repo entity.FailedRepository
		err = json.Unmarshal([]byte(result), &repo)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal failed repository: %w", op, err)
		}
		failedRepos = append(failedRepos, repo)
	}

	return failedRepos, nil
}
