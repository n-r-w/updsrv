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

	tokens map[string]bool // список токенов доступа
}

// New Инициализация маршрутов
func New(router httprouter.Router, repo UpdateInterface, config *config.Config) (*Presenter, error) {
	p := &Presenter{
		controller: router,
		repo:       repo,
		config:     config,
		tokens:     map[string]bool{},
	}

	if len(config.Tokens) == 0 {
		return nil, nerr.New("no access tokens")
	}

	// инициализация хранилища токенов
	for _, v := range config.Tokens {
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
