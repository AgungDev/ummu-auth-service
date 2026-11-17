package repositories

import (
	"auth_service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureRecordExists(db *gorm.DB, model any, id uuid.UUID) error {
	var count int64
	if err := db.Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func ensureRecordsExist(db *gorm.DB, model any, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	uniqueIDs := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		uniqueIDs[id] = struct{}{}
	}

	var count int64
	if err := db.Model(model).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(uniqueIDs)) {
		return domain.ErrNotFound
	}
	return nil
}
