package psql

import (
	"context"
	"time"

	"github.com/n-r-w/lg"
	"github.com/n-r-w/updsrv/internal/entity"
)

// Update получить обновление
func (p *Repo) Update(сhannel string, version entity.Version, ctx context.Context) ([]byte, entity.UpdateInfo, error) {
	ctxChild, cancel := context.WithTimeout(ctx, time.Second*time.Duration(p.config.DbWriteTimeout))
	defer cancel()

	ok, toI, err := p.getUpdateInfo(сhannel, version, true, ctxChild)
	if err != nil {
		return nil, entity.UpdateInfo{}, err
	}
	if !ok {
		p.logOp(ctx, lg.Info, "update not found: %s, %s", сhannel, version.String())
		return nil, entity.UpdateInfo{}, nil
	}

	res, zipData, err := p.cache.Get(processVersion{
		fromC: сhannel,
		fromV: version,
		toC:   toI.Channel,
		toV:   toI.Version,
	}, ctxChild)
	if err != nil {
		return nil, entity.UpdateInfo{}, err
	}

	return zipData, *res, nil
}
