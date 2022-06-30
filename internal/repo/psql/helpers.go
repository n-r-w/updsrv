// Package psql Содержит реализацию интерфейса репозитория логов для postgresql
package psql

import (
	"context"
	"fmt"

	"github.com/n-r-w/lg"
	"github.com/n-r-w/nerr"
	"github.com/n-r-w/sqlb"
	"github.com/n-r-w/sqlq"
	"github.com/n-r-w/updsrv/internal/entity"
)

/* Check проверить обновление
loadContent - грузить ли содержимое файлов
lastUpdate - если истина, ищет наличие обновления. иначе грузит инфу об указанной версии */
func (p *Repo) getUpdateInfo(сhannel string, version entity.Version, lastUpdate bool, ctx context.Context) (bool, entity.UpdateInfo, error) {
	tx := sqlq.NewTx(p.Pool, ctx)
	tx.Begin()
	defer tx.Rollback() // commit не нужен, т.к. мы ничего не меняем и транзация нужна для работы с LO

	var sql string
	var err error

	if lastUpdate {
		sql, err = sqlb.Bind(
			`SELECT id, record_time, channel, major, minor, patch, revision, build_time, info 
			FROM updates		
			WHERE enabled = TRUE AND channel = :channel AND 
			(
				major > :major 
				OR (major = :major AND minor > :minor) 
				OR (major = :major AND minor = :minor AND patch > :patch)
				OR (major = :major AND minor = :minor AND patch = :patch AND revision > :revision)
			) 
			ORDER BY major DESC, minor DESC, patch DESC, revision DESC
			LIMIT 1`,
			map[string]interface{}{
				"channel":  сhannel,
				"major":    version.Major,
				"minor":    version.Minor,
				"patch":    version.Patch,
				"revision": version.Revision,
			},
			"getUpdateInfoMain")
	} else {
		sql, err = sqlb.Bind(
			`SELECT id, record_time, channel, major, minor, patch, revision, build_time, info 
			FROM updates		
			WHERE enabled = TRUE AND channel = :channel AND major = :major AND minor = :minor AND patch = :patch AND revision = :revision`,
			map[string]interface{}{
				"channel":  сhannel,
				"major":    version.Major,
				"minor":    version.Minor,
				"patch":    version.Patch,
				"revision": version.Revision,
			},
			"getUpdateInfoDirect")
	}

	if err != nil {
		return false, entity.UpdateInfo{}, err
	}

	q, err := sqlq.SelectTxRow(tx, sql)
	if err != nil {
		return false, entity.UpdateInfo{}, nerr.New(err, sql)
	}

	if q == nil {
		return false, entity.UpdateInfo{}, nil
	}

	info := entity.UpdateInfo{
		ID:         q.UInt64("id"),
		CreateTime: q.Time("record_time"),
		BuildTime:  q.Time("build_time"),
		Channel:    q.String("channel"),
		Version: entity.Version{
			Major:    q.Int("major"),
			Minor:    q.Int("minor"),
			Patch:    q.Int("patch"),
			Revision: q.Int("revision"),
		},
		Info:    q.String("info"),
		Enabled: true, // раз получили инфу, то true
	}

	// файлы
	sql, err = sqlb.BindOne(
		`SELECT file_name, checksum, data_oid
		FROM files		
		WHERE id_update = :id_update`,
		"id_update", q.UInt64("id"),
		"getUpdateInfoFiles")
	if err != nil {
		return false, entity.UpdateInfo{}, err
	}
	if q, err = sqlq.SelectTx(tx, sql); err != nil {
		return false, entity.UpdateInfo{}, nerr.New(err, sql)
	}

	for q.Next() {
		fi := entity.FileInfo{
			Name:     q.String("file_name"),
			Checksum: q.String("checksum"),
			DataID:   uint32(q.UInt64("data_oid")),
		}

		info.Files = append(info.Files, fi)
	}

	return true, info, nil
}

func (p *Repo) logOp(ctx context.Context, level lg.Level, format string, args ...any) {
	ci := entity.GetClientInfoFromContext(ctx)
	if ci == nil {
		panic("no client info")
	}

	msg := fmt.Sprintf(format, args...)

	if ci.RealIP == ci.IP {
		p.logger.Level(level, "addr: %s, %s", ci.IP, msg)
	} else {
		p.logger.Level(level, "addr: %s(%s), %s", ci.RealIP, ci.IP, msg)
	}
}
