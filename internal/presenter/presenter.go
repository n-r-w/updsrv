package presenter

import (
	"net/http"

	"github.com/n-r-w/eno"
	"github.com/n-r-w/httprouter"
	"github.com/n-r-w/nerr"
	"github.com/n-r-w/updsrv/internal/config"
	"github.com/n-r-w/updsrv/internal/entity"
)

type Presenter struct {
	controller httprouter.Router
	repo       UpdateInterface
	config     *config.Config

	tokens      map[string]bool // список всех токенов
	tokensRead  map[string]bool // список токенов доступа на чтение
	tokensWrite map[string]bool // список токенов доступа на запись
}

// New Инициализация маршрутов
func New(router httprouter.Router, repo UpdateInterface, config *config.Config) (*Presenter, error) {
	p := &Presenter{
		controller:  router,
		repo:        repo,
		config:      config,
		tokens:      map[string]bool{},
		tokensRead:  map[string]bool{},
		tokensWrite: map[string]bool{},
	}

	if len(config.TokensRead) == 0 {
		return nil, nerr.New("no access read tokens")
	}
	if len(config.TokensWrite) == 0 {
		return nil, nerr.New("no access write tokens")
	}

	// инициализация хранилища токенов
	for _, v := range config.TokensRead {
		p.tokensRead[v] = true
		p.tokens[v] = true
	}
	for _, v := range config.TokensWrite {
		p.tokensWrite[v] = true
		p.tokens[v] = true
	}

	// устанавливаем middleware для проверки валидности сессии
	router.AddMiddleware("/api", p.authenticateUser)

	// добавить новую версию
	router.AddRoute("/api", "/add", p.add(), "POST")
	// проверить наличие новой версии
	router.AddRoute("/api", "/check", p.check(), "POST")
	// получить новую версию
	router.AddRoute("/api", "/update", p.update(), "POST")

	return p, nil
}

// Аутентификация пользователя
func (p *Presenter) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Authorization")
		if _, ok := p.tokens[token]; !ok {
			p.controller.RespondError(w, http.StatusUnauthorized, nerr.New(eno.ErrNoAccess))
			return
		}

		// добавляем в контекст инфу о клиенте
		ci := &entity.ClientInfo{
			IP: r.RemoteAddr,
		}
		ctx := entity.PutClientInfoToContext(ci, r.Context())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Проверка прав
func (p *Presenter) checkRights(r *http.Request, writeAccess bool) error {
	token := r.Header.Get("X-Authorization")
	if writeAccess {
		if _, ok := p.tokensWrite[token]; !ok {
			return nerr.New(eno.ErrNoAccess, "no write access")
		}
	} else {
		if _, ok := p.tokensRead[token]; !ok {
			return nerr.New(eno.ErrNoAccess, "no read access")
		}
	}
	return nil
}
