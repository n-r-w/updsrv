package psql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/n-r-w/eno"
	"github.com/n-r-w/lg"
	"github.com/n-r-w/nerr"
	"github.com/n-r-w/sqlb"
	"github.com/n-r-w/sqlq"
	"github.com/n-r-w/tools"
	"github.com/n-r-w/updsrv/internal/entity"
)

// Add добавить обновление
func (p *Repo) Add(ui *entity.UpdateInfo, ctx context.Context) error {
	ctxChild, cancel := context.WithTimeout(ctx, time.Second*time.Duration(p.config.DbWriteTimeout))
	defer cancel()

	p.logOp(ctx, lg.Info, "request to add a new version: %s, %s", ui.Channel, ui.Version.String())

	tx := sqlq.NewTx(p.Pool, ctxChild)
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()

	sql, err := sqlb.Bind(
		`INSERT INTO public.updates(
			channel, major, minor, patch, revision, build_time, info, enabled)
			VALUES (:channel, :major, :minor, :patch, :revision, :build_time, :info, :enabled) RETURNING id`,
		map[string]interface{}{
			"channel":    ui.Channel,
			"major":      ui.Version.Major,
			"minor":      ui.Version.Minor,
			"patch":      ui.Version.Patch,
			"revision":   ui.Version.Revision,
			"build_time": ui.BuildTime,
			"info":       ui.Info,
			"enabled":    ui.Enabled,
		}, "add")
	if err != nil {
		return err
	}

	q, err := sqlq.SelectTxRow(tx, sql)
	if err != nil {
		if nerr.SqlCode(err) == pgerrcode.UniqueViolation {
			// такое обновление уже есть
			return nerr.New(eno.ErrObjectExist)
		}
		return nerr.New(err, tools.SimplifyString(sql))
	}
	idUpdate := q.UInt64("id")

	// сначала сохраняем содержимое файлов
	var fileOids []uint32
	for i, fi := range ui.Files {
		oid, err := sqlq.SaveLargeObject(tx, 0, fi.Data)
		if err != nil {
			return err
		}
		fileOids = append(fileOids, oid)
		ui.Files[i].Data = nil // для экономии памяти
	}

	// затем информацию о файлах
	var filesSql []string
	for i, fi := range ui.Files {
		fsql, err := sqlb.Bind("(:id_update, :file_name, :checksum, :data_oid)",
			map[string]interface{}{
				"id_update": idUpdate,
				"file_name": fi.Name,
				"checksum":  fi.Checksum,
				"data_oid":  fileOids[i],
			},
			"files")
		if err != nil {
			return err
		}
		filesSql = append(filesSql, fsql)
	}

	sql = fmt.Sprintf(`INSERT INTO public.files(id_update, file_name, checksum, data_oid) VALUES %s`, strings.Join(filesSql, ","))
	if _, err := sqlq.ExecTx(tx, sql); err != nil {
		return nerr.New(err, tools.SimplifyString(sql))
	}

	// удаляем старые версии
	sql, err = sqlb.Bind(
		`WITH deleted AS (
		DELETE FROM updates 
		WHERE channel = :channel AND date_part ('day', now()-record_time) > :min_days AND
		id NOT IN 
		(
			SELECT id 
			FROM updates
			WHERE channel = :channel
			ORDER BY major DESC, minor DESC, patch DESC, revision DESC
			LIMIT :max_count
		)
		RETURNING *
		)
		SELECT major, minor, patch, revision 
		FROM deleted
		GROUP BY major, minor, patch, revision`,
		map[string]interface{}{
			"channel":   ui.Channel,
			"max_count": p.config.MaxVersionCount,
			"min_days":  p.config.MinVersionAge,
		}, "DeleteOld")
	if err != nil {
		return err
	}
	if q, err = sqlq.SelectTx(tx, sql); err != nil {
		return nerr.New(err, tools.SimplifyString(sql))
	}

	deletedVersions := []entity.Version{}
	for q.Next() {
		deletedVersions = append(deletedVersions, entity.Version{
			Major:    q.Int("major"),
			Minor:    q.Int("minor"),
			Patch:    q.Int("patch"),
			Revision: q.Int("revision"),
		})
	}
	if len(deletedVersions) > 0 {
		var delInfo string
		for i, v := range deletedVersions {
			delInfo += v.String()
			if i < len(deletedVersions)-1 {
				delInfo += ", "
			}
		}
		p.logOp(ctx, lg.Info, "%s, old versions deleted: %s", ui.Channel, delInfo)
	}

	if err = tx.Commit(); err == nil {
		p.logOp(ctx, lg.Info, "new version added: %s, %s", ui.Channel, ui.Version.String())
	}

	return err
}
