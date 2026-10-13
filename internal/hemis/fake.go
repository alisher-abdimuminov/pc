package hemis

import (
	"context"
	"sync"
)

// Fake is an in-memory HEMIS for tests and local development.
type Fake struct {
	mu       sync.Mutex
	accounts map[string]fakeAccount // by login
	Down     bool                   // simulate an outage
	Calls    int                    // number of Login calls
	LastPass string                 // last password received (tests assert it is not persisted)
}

type fakeAccount struct {
	password string
	profile  Profile
}

func NewFake() *Fake { return &Fake{accounts: map[string]fakeAccount{}} }

func (f *Fake) Add(login, password string, p Profile) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.accounts[login] = fakeAccount{password: password, profile: p}
}

func (f *Fake) SetProfile(login string, p Profile) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := f.accounts[login]
	a.profile = p
	f.accounts[login] = a
}

func (f *Fake) SetDown(v bool) { f.mu.Lock(); f.Down = v; f.mu.Unlock() }

func (f *Fake) Login(_ context.Context, username, password string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	f.LastPass = password
	if f.Down {
		return "", ErrUnavailable
	}
	a, ok := f.accounts[username]
	if !ok || a.password != password {
		return "", ErrInvalidCredentials
	}
	return "token:" + username, nil
}

func (f *Fake) Me(_ context.Context, token string) (Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Down {
		return Profile{}, ErrUnavailable
	}
	for login, a := range f.accounts {
		if token == "token:"+login {
			return a.profile, nil
		}
	}
	return Profile{}, ErrUnavailable
}
