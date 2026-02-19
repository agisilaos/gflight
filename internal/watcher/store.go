package watcher

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/agisilaos/gflight/internal/model"
)

type Store struct {
	Path string
}

func (s Store) Load() (model.WatchStore, error) {
	var ws model.WatchStore
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ws.Watches = []model.Watch{}
			return ws, nil
		}
		return ws, err
	}
	if err := json.Unmarshal(b, &ws); err != nil {
		return ws, err
	}
	if ws.Watches == nil {
		ws.Watches = []model.Watch{}
	}
	return ws, nil
}

func (s Store) Save(ws model.WatchStore) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(s.Path, b, 0o600)
}
