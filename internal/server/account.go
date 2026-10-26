package server

import (
	"errors"
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
	"golang.org/x/time/rate"
)

const saveBurst = 3

var (
	saveRate     = rate.Every(time.Second)
	errNotStored = errors.New("applied for this session, but your profile could not be stored")
)

type account struct {
	client  *chat.Client
	store   *profile.Store
	logger  *log.Logger
	owner   string
	persist bool
	limiter *rate.Limiter

	mu        sync.Mutex
	profile   profile.Profile
	savedName string
}

func newAccount(client *chat.Client, store *profile.Store, logger *log.Logger, id chat.Identity, start profile.Profile, saved bool) *account {
	a := &account{
		client:  client,
		store:   store,
		logger:  logger,
		owner:   id.ID,
		persist: !id.Guest && store != nil,
		limiter: rate.NewLimiter(saveRate, saveBurst),
		profile: start,
	}
	if saved {
		a.savedName = start.Name
	}
	m := client.Member()
	a.profile.Name, a.profile.Color = m.Name, m.Color
	return a
}

func (a *account) Profile() profile.Profile {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.profile
}

func (a *account) Persistent() bool {
	return a.persist
}

func (a *account) Update(p profile.Profile) error {
	if err := p.Validate(); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.limiter.Allow() {
		return chat.ErrRateLimited
	}
	renamed := p.Name != a.profile.Name
	if renamed {
		if err := a.client.Rename(p.Name); err != nil {
			return err
		}
	}
	if p.Color != a.profile.Color {
		a.client.SetColor(p.Color)
	}
	a.profile = p

	if !a.persist {
		return nil
	}
	stored := p
	if !renamed && a.savedName != "" {
		stored.Name = a.savedName
	}
	if err := a.store.Save(a.owner, stored); err != nil {
		if errors.Is(err, chat.ErrNameTaken) {
			return err
		}
		a.logger.Error("storing profile", "owner", a.owner, "error", err)
		return errNotStored
	}
	a.savedName = stored.Name
	return nil
}
