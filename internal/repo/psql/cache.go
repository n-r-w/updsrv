package psql

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/n-r-w/eno"
	"github.com/n-r-w/lg"
	"github.com/n-r-w/nerr"
	"github.com/n-r-w/sqlb"
	"github.com/n-r-w/sqlq"
	"github.com/n-r-w/updsrv/internal/entity"
	"golang.org/x/time/rate"
)

type processVersion struct {
	fromC string
	fromV entity.Version
	toC   string
	toV   entity.Version
}

func (v *processVersion) String() string {
	return fmt.Sprintf("%s_%s_%s_%s", v.fromC, v.fromV.String(), v.toC, v.fromV.String())
}

// Cache отвечает за получение обновлений из БД с использованием кэша
type Cache struct {
	r          *Repo
	mutex      sync.Mutex
	processing map[string]int
	limiter    *rate.Limiter
}

func NewCache(r *Repo) *Cache {
	return &Cache{
		r:          r,
		mutex:      sync.Mutex{},
		processing: map[string]int{},
		limiter:    rate.NewLimiter(rate.Limit(r.config.RateLimit), r.config.RateLimitBurst),
	}
}

func (c *Cache) Get(v processVersion, ctx context.Context) (*entity.UpdateInfo, []byte, error) {
	// Защита от DDOS и в целом от перегрузки сервера БД запросами
	if !c.limiter.Allow() {
		return nil, nil, nerr.New(eno.ErrTooManyRequests)
	}

	if v.fromC != v.toC {
		return nil, nil, nerr.New(eno.ErrInternal, "can't get update for different channels")
	}

	// если уже готовится такой диф, то ждем
	if err := c.waitReady(v, ctx); err != nil {
		return nil, nil, err
	}

	// ищем диф в БД, не блокируем других

	// сначала смотрим в кэше прямое обновление
	res, zipData, updateCache, err := c.askCache(v, ctx, true)
	if err != nil {
		return nil, nil, err
	}
	if zipData != nil {
		c.r.logOp(ctx, lg.Info, "diff from cache: %s, %s => %s", v.fromC, v.fromV.String(), v.toV.String())
		return res, zipData, nil
	}

	if !updateCache {
		// затем смотрим в кэше полное обновление
		res, zipData, updateCache, err = c.askCache(v, ctx, false)
		if err != nil {
			return nil, nil, err
		}
		if zipData != nil {
			c.r.logOp(ctx, lg.Info, "full data from cache: %s, %s => %s", v.toC, v.fromV.String(), v.toV.String())
			return res, zipData, nil
		}
	}

	// начинаем готовить diff
	c.r.logOp(ctx, lg.Info, "calculating diff: %s, %s => %s", v.fromC, v.fromV.String(), v.toV.String())

	c.mutex.Lock()
	counter := c.processing[v.String()]
	// в теории возможна ситуация, когда два запроса стали готовить одинаковый диф.
	// бороться с этим сложно, поэтому лучше просто увеличивать счетчик блокировки
	c.processing[v.String()] = counter + 1
	c.mutex.Unlock()

	// уменьшаем счетчик при выходе
	defer func() {
		c.mutex.Lock()
		counter := c.processing[v.String()]
		if counter <= 0 {
			panic("counter error")
		}
		if counter == 1 {
			delete(c.processing, v.String())
		} else {
			c.processing[v.String()] = counter - 1
		}
		c.r.logOp(ctx, lg.Info, "diff created: %s, %s => %s", v.fromC, v.fromV.String(), v.toV.String())
		c.mutex.Unlock()

	}()

	var fullUpdate bool
	// информация об версии, с которой обновляем
	ok, fromI, err := c.r.getUpdateInfo(v.fromC, v.fromV, false, ctx)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		// версия не найдена, возвращаем полное содержимое последней версии
		res = &entity.UpdateInfo{}
		if ok, *res, err = c.r.getUpdateInfo(v.fromC, v.fromV, true, ctx); err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, nil
		}
		fullUpdate = true
	}

	// информация об версии, на которую обновляем
	ok, toI, err := c.r.getUpdateInfo(v.toC, v.toV, false, ctx)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		// версия не найдена, возвращаем полное содержимое последней версии
		res = &entity.UpdateInfo{}
		if ok, *res, err = c.r.getUpdateInfo(v.fromC, v.fromV, true, ctx); err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, nil
		}
		fullUpdate = true
	}

	if fullUpdate {
		// делаем полный zip
		c.r.logOp(ctx, lg.Warn, "no diff found: %s, %s => %s", v.fromC, v.fromV.String(), v.toV.String())
		zipData, err = c.createZip(res.Files, ctx)
		if err != nil {
			return nil, nil, err
		}

	} else {
		// вычисляем дельту
		res = &toI
		res.Enabled = true
		res.Files = createDiff(fromI.Files, toI.Files)

		// делаем zip
		zipData, err = c.createZip(res.Files, ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	// сохраняем кэш в БД
	if err = c.save(updateCache, fromI, toI, res, zipData, ctx); err != nil {
		return nil, nil, err
	}

	return res, zipData, nil
}

// сохранить кэш в БД
func (c *Cache) save(updateCache bool, fromI entity.UpdateInfo, toI entity.UpdateInfo, res *entity.UpdateInfo, zipData []byte, ctx context.Context) error {
	tx := sqlq.NewTx(c.r.Pool, ctx)
	tx.Begin()
	defer tx.Rollback()

	oid, err := sqlq.SaveLargeObject(tx, 0, zipData)
	if err != nil {
		return err
	}

	jsinfo, err := json.Marshal(*res)
	if err != nil {
		return err
	}

	var sql string
	if updateCache {
		sql, err = sqlb.Bind(
			`UPDATE cache SET diff_oid = :diff_oid, diff_info = :diff_info
			WHERE id_update_from = :id_update_from AND id_update_to = :id_update_to
			`,
			map[string]interface{}{
				"id_update_from": vNull(fromI.ID),
				"id_update_to":   toI.ID,
				"diff_oid":       oid,
				"diff_info":      string(jsinfo),
			}, "UpdateCache")
		if err != nil {
			return err
		}

	} else {
		sql, err = sqlb.Bind(`INSERT INTO cache(id_update_from, id_update_to, diff_oid, diff_info) VALUES (:id_update_from, :id_update_to, :diff_oid, :diff_info)`,
			map[string]interface{}{
				"id_update_from": vNull(fromI.ID),
				"id_update_to":   toI.ID,
				"diff_oid":       oid,
				"diff_info":      string(jsinfo),
			}, "UpdateCache")
		if err != nil {
			return err
		}
	}

	if err = sqlq.ExecTx(tx, sql); err != nil {
		// ошибку обновления кэша игнорируем, т.к. могут быть коллизии с другими пользователями и нам это не важно
		return nil
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// ожидание готовности к выдаче данных
func (c *Cache) waitReady(v processVersion, ctx context.Context) error {
	// если уже готовится такой диф, то ждем
	wasWarn := false
	for {
		c.mutex.Lock()
		counter := c.processing[v.String()]
		c.mutex.Unlock()

		if counter > 0 {
			if !wasWarn {
				wasWarn = true
				c.r.logOp(ctx, lg.Info, "waiting calculating diff: %s, %s => %s", v.fromC, v.fromV.String(), v.toV.String())
			}

			select {
			case <-time.After(time.Second):
				break
			case <-ctx.Done():
				return nerr.New(eno.ErrDeadlineExceeded)
			}
		} else {
			break
		}
	}

	return nil
}

func (c *Cache) askCache(v processVersion, ctx context.Context,
	// если истина, то ищет точно обновление, иначе ищет полный апдейт
	direct bool) (res *entity.UpdateInfo, zipData []byte, updateCache bool, err error) {

	var sql string
	res = &entity.UpdateInfo{}

	if direct {
		sql, err = sqlb.Bind(
			`SELECT c.diff_oid, c.diff_info::text   
		FROM cache c   
		WHERE 
			EXISTS(    
				SELECT * FROM updates u    
				WHERE u.channel = :channel AND 				
				(u.id = c.id_update_from AND u.major = :from_major AND u.minor = :from_minor AND u.patch = :from_patch AND u.revision = :from_revision))
			AND
			EXISTS(    
				SELECT * FROM updates u    
				WHERE u.channel = :channel AND
				(u.id = c.id_update_to AND u.major = :to_major AND u.minor = :to_minor AND u.patch = :to_patch AND u.revision = :to_revision))`,
			map[string]interface{}{
				"channel":       v.fromC,
				"from_major":    v.fromV.Major,
				"from_minor":    v.fromV.Minor,
				"from_patch":    v.fromV.Patch,
				"from_revision": v.fromV.Revision,
				"to_major":      v.toV.Major,
				"to_minor":      v.toV.Minor,
				"to_patch":      v.toV.Patch,
				"to_revision":   v.toV.Revision,
			}, "GetCacheDirect")

	} else {
		sql, err = sqlb.Bind(
			`SELECT c.diff_oid, c.diff_info::text   
		FROM cache c   
		WHERE 
			EXISTS(    
				SELECT * FROM updates u    
				WHERE u.channel = :channel AND c.id_update_from IS NULL AND
				NOT EXISTS(
					SELECT * FROM updates u1 WHERE (u1.channel = :channel AND u1.major = :from_major AND 
						u1.minor = :from_minor AND u1.patch = :from_patch AND u1.revision = :from_revision)
				)
			)
			AND
			EXISTS(    
				SELECT * FROM updates u    
				WHERE u.channel = :channel AND
				(u.id = c.id_update_to AND u.major = :to_major AND u.minor = :to_minor AND u.patch = :to_patch AND u.revision = :to_revision))`,
			map[string]interface{}{
				"channel":       v.fromC,
				"from_major":    v.fromV.Major,
				"from_minor":    v.fromV.Minor,
				"from_patch":    v.fromV.Patch,
				"from_revision": v.fromV.Revision,
				"to_major":      v.toV.Major,
				"to_minor":      v.toV.Minor,
				"to_patch":      v.toV.Patch,
				"to_revision":   v.toV.Revision,
			}, "GetCacheFull")
	}
	if err != nil {
		return nil, nil, false, err
	}
	q, err := sqlq.SelectRow(c.r.Pool, ctx, sql)
	if err != nil {
		return nil, nil, false, err
	}

	if q != nil {
		// найдено в кэше
		if err = json.Unmarshal(q.Bytes("diff_info"), res); err == nil {
			// извлекаем zip
			tx := sqlq.NewTx(c.r.Pool, ctx)
			defer tx.Rollback()
			tx.Begin()
			zipData, err = sqlq.LoadLargeObject(tx, uint32(q.UInt64("diff_oid")))
			if err != nil {
				return nil, nil, false, err
			}
			tx.Rollback()

			return res, zipData, false, nil
		} else {
			// в кэше что-то старое и непонятное
			updateCache = true
		}
	}

	return nil, nil, updateCache, nil
}

// создание архива
func (c *Cache) createZip(fs []entity.FileInfo, ctx context.Context) ([]byte, error) {
	tx := sqlq.NewTx(c.r.Pool, ctx) // для загрузки LO
	tx.Begin()
	defer tx.Rollback()

	file, err := ioutil.TempFile("", "upsrvdif")
	if err != nil {
		return nil, nerr.New(err)
	}
	defer os.Remove(file.Name())

	zipWriter := zip.NewWriter(file)

	for _, fi := range fs {
		if fi.Status == entity.FileRemoved {
			continue
		}

		zipFile, err := zipWriter.Create(fi.Name)
		if err != nil {
			return nil, nerr.New(err)
		}

		// грузим содержимое файла
		data, err := sqlq.LoadLargeObject(tx, fi.DataID)
		if err != nil {
			return nil, err
		}

		if _, err = zipFile.Write(data); err != nil {
			return nil, nerr.New(err)
		}
	}

	zipFile, err := zipWriter.Create(".update_file_info.txt")
	if err != nil {
		return nil, nerr.New(err)
	}
	if _, err = zipFile.Write(createDiffInfoFile(fs)); err != nil {
		return nil, nerr.New(err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, nerr.New(err)
	}

	return ioutil.ReadFile(file.Name())
}

// подготовка значения перез записью в БД
func vNull(v interface{}) interface{} {
	switch d := v.(type) {
	case int:
		if d == 0 {
			return nil
		}
		return d
	case uint:
		if d == 0 {
			return nil
		}
		return d
	case int8:
		if d == 0 {
			return nil
		}
		return d
	case int16:
		if d == 0 {
			return nil
		}
		return d
	case int32:
		if d == 0 {
			return nil
		}
		return d
	case int64:
		if d == 0 {
			return nil
		}
		return d
	case uint8:
		if d == 0 {
			return nil
		}
		return d
	case uint16:
		if d == 0 {
			return nil
		}
		return d
	case uint32:
		if d == 0 {
			return nil
		}
		return d
	case uint64:
		if d == 0 {
			return nil
		}
		return d
	case string:
		if len(strings.TrimSpace(d)) == 0 {
			return nil
		}
		return d
	case []byte:
		if len(d) == 0 {
			return nil
		}
		return d
	default:
		return v
	}

}

// вычисление разницы между дистрибутивами
func createDiff(from []entity.FileInfo, to []entity.FileInfo) []entity.FileInfo {
	var res []entity.FileInfo

	fromFiles := map[string]*entity.FileInfo{}
	for i, fi := range from {
		fromFiles[fi.Name] = &from[i]
	}
	toFiles := map[string]*entity.FileInfo{}
	for i, fi := range to {
		toFiles[fi.Name] = &to[i]
	}

	for name, fiTo := range toFiles {
		fiFrom := fromFiles[name]

		if fiFrom == nil { // новый файл
			fiTo.Status = entity.FileCreated
			res = append(res, *fiTo)
		} else if fiFrom.Checksum != fiTo.Checksum { // измененный файл
			fiTo.Status = entity.FileModified
			res = append(res, *fiTo)
		}
	}

	for name, fiFrom := range fromFiles {
		fiTo := toFiles[name]
		if fiTo == nil { // удаленный файл
			fiFrom.Status = entity.FileRemoved
			res = append(res, *fiFrom)
		}
	}

	return res
}

// создание текстового файла с описанием обновления
func createDiffInfoFile(fs []entity.FileInfo) []byte {
	res := new(bytes.Buffer)

	for _, fi := range fs {
		var mark string
		switch fi.Status {
		case entity.FileCreated:
			mark = "+"
		case entity.FileRemoved:
			mark = "-"
		case entity.FileModified:
			mark = "*"
		default:
			mark = "?"
		}

		res.WriteString(fmt.Sprintf("%s %s\n", mark, fi.Name))
	}

	return res.Bytes()
}
