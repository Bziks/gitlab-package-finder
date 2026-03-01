package packagetype

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/ports"
)

const _formatToAll = "%s_all"
const _formatToName = "%s_%s"

type Repository struct {
	dbRepo ports.PackageTypeRepository
	client *cache.Cache
	key    string
	ttl    time.Duration
}

func NewCacheRepository(dbRepo ports.PackageTypeRepository, client *cache.Cache) *Repository {
	return &Repository{
		dbRepo: dbRepo,
		client: client,
		key:    "package_type",
		ttl:    24 * time.Hour,
	}
}

func (r *Repository) GetAll(ctx context.Context) ([]entity.PackageType, error) {
	key := fmt.Sprintf(_formatToAll, r.key)

	if x, found := r.client.Get(key); found {
		if val, ok := x.([]entity.PackageType); ok {
			return val, nil
		}
	}

	entities, err := r.dbRepo.GetAll(ctx)
	if err != nil {
		return make([]entity.PackageType, 0), err
	}

	r.client.Set(key, entities, r.ttl)

	return entities, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*entity.PackageType, error) {
	key := fmt.Sprintf(_formatToName, r.key, name)

	if x, found := r.client.Get(key); found {
		if val, ok := x.(*entity.PackageType); ok {
			return val, nil
		}
	}

	entity, err := r.dbRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	if entity != nil {
		r.client.Set(key, entity, r.ttl)
	}

	return entity, nil
}
