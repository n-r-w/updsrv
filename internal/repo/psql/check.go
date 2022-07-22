package psql

import (
	"context"
	"time"

	"github.com/n-r-w/lg"
	"github.com/n-r-w/updsrv/internal/entity"
)

// Check проверить обновление
func (p *Repo) Check(сhannel string, version entity.Version, ctx context.Context) (bool, entity.UpdateInfo, error) {
	ctxChild, cancel := context.WithTimeout(ctx, time.Second*time.Duration(p.config.DbWriteTimeout))
	defer cancel()

	ok, info, err := p.getUpdateInfo(сhannel, version, true, ctxChild)

	if err == nil {
		if ok {
			p.logOp(ctx, lg.Info, "update found: %s, %s => %s", сhannel, version.String(), info.Version.String())
		} else {
			p.logOp(ctx, lg.Info, "update not found: %s, %s", сhannel, version.String())
		}
	}

	return ok, info, err
}
