package presenter

import (
	"archive/zip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/n-r-w/nerr"
	"github.com/n-r-w/tools"
	"github.com/n-r-w/updsrv/internal/entity"
)

// добавить новую версию
func (p *Presenter) add() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(int64(p.config.MaxUpdateSize) << 20); err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}

		// файл с ПО
		file, header, err := r.FormFile("update")
		if err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}
		defer file.Close()

		zr, err := zip.NewReader(file, header.Size)
		if err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}

		var files []entity.FileInfo
		for _, zipFile := range zr.File {
			f, err := zipFile.Open()
			if err != nil || len(zipFile.Name) == 0 {
				p.controller.RespondError(w, http.StatusBadRequest, err)
				return
			}

			if zipFile.Name[len(zipFile.Name)-1] == '/' {
				continue
			}

			fi := entity.FileInfo{}
			fi.Name = zipFile.Name

			fi.Data, err = ioutil.ReadAll(f)
			f.Close()
			if err != nil {
				p.controller.RespondError(w, http.StatusBadRequest, err)
				return
			}

			if fi.Checksum, err = tools.Sha256sum(fi.Data); err != nil {
				p.controller.RespondError(w, http.StatusInternalServerError, err)
				return
			}

			files = append(files, fi)
		}

		// информация
		var info entity.UpdateInfo
		info.Files = files

		info.BuildTime, err = time.Parse("2006-01-02T15:04", r.FormValue("buildTime"))
		if err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}
		info.Channel = r.FormValue("channel")
		info.Info = r.FormValue("info")

		if v := strings.Split(r.FormValue("version"), "."); len(v) == 0 || len(v) > 4 {
			p.controller.RespondError(w, http.StatusBadRequest, nerr.NewFmt("invalid version %s", r.FormValue("version")))
			return

		} else {
			if info.Version.Major, err = strconv.Atoi(v[0]); err != nil {
				p.controller.RespondError(w, http.StatusBadRequest, err)
				return
			}
			if len(v) >= 2 {
				if info.Version.Minor, err = strconv.Atoi(v[1]); err != nil {
					p.controller.RespondError(w, http.StatusBadRequest, err)
					return
				}
			}
			if len(v) >= 3 {
				if info.Version.Patch, err = strconv.Atoi(v[2]); err != nil {
					p.controller.RespondError(w, http.StatusBadRequest, err)
					return
				}
			}
			if len(v) >= 4 {
				if info.Version.Revision, err = strconv.Atoi(v[3]); err != nil {
					p.controller.RespondError(w, http.StatusBadRequest, err)
					return
				}
			}
		}

		enabled := r.FormValue("enabled")
		if len(enabled) == 0 || strings.EqualFold(enabled, "true") {
			info.Enabled = true
		} else if strings.EqualFold(enabled, "false") {
			info.Enabled = false
		} else {
			p.controller.RespondError(w, http.StatusBadRequest, nerr.NewFmt("invalid 'enabled': %s", enabled))
			return
		}

		if err := p.repo.Add(&info, r.Context()); err != nil {
			p.controller.RespondError(w, http.StatusForbidden, err)
			return
		}

		p.controller.RespondData(w, http.StatusCreated, "application/json; charset=utf-8", nil)
	}
}

// проверить наличие новой версии
func (p *Presenter) check() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// парсим входящий json
		var info entity.UpdateInfo
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}

		found, updateInfo, err := p.repo.Check(info.Channel, info.Version, r.Context())
		if err != nil {
			p.controller.RespondError(w, http.StatusInternalServerError, err)
			return
		}

		if found {
			p.controller.RespondData(w, http.StatusOK, "application/json; charset=utf-8", updateInfo)
		} else {
			p.controller.RespondData(w, http.StatusNoContent, "", nil)
		}

	}
}

// получить новую версию
func (p *Presenter) update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// парсим входящий json
		var info entity.UpdateInfo
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			p.controller.RespondError(w, http.StatusBadRequest, err)
			return
		}

		data, updateInfo, err := p.repo.Update(info.Channel, info.Version, r.Context())
		if err != nil {
			p.controller.RespondError(w, http.StatusInternalServerError, err)
			return
		}

		if len(data) == 0 {
			p.controller.RespondData(w, http.StatusNoContent, "", nil)
			return
		}

		w.Header().Set("Version-Date", updateInfo.BuildTime.Format("2006-01-02T15:04"))
		w.Header().Set("Version-Major", strconv.Itoa(updateInfo.Version.Major))
		w.Header().Set("Version-Minor", strconv.Itoa(updateInfo.Version.Minor))
		w.Header().Set("Version-Patch", strconv.Itoa(updateInfo.Version.Patch))
		w.Header().Set("Version-Revision", strconv.Itoa(updateInfo.Version.Revision))

		p.controller.RespondData(w, http.StatusOK, "application/zip", data)
	}
}
