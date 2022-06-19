// Package presenter ...
package presenter

import (
	"context"

	"github.com/n-r-w/updsrv/internal/entity"
)

// UpdateInterface ...
type UpdateInterface interface {
	// Добавить обновление в БД. Внутри метода очищается содержимое Files.Data для экономии памяти
	Add(updateInfo *entity.UpdateInfo, ctx context.Context) error
	// Проверка наличия обновления
	Check(сhannel string, version entity.Version, ctx context.Context) (bool, entity.UpdateInfo, error)
	// Вернуть дельту обновления в формате zip
	Update(сhannel string, version entity.Version, ctx context.Context) ([]byte, entity.UpdateInfo, error)
}
