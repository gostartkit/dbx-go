package app

import (
	"database/sql"

	"pkg.gostartkit.com/dbx/internal/config"
)

type Session struct {
	Connection *config.ConnectionConfig
	DB         *sql.DB
}

func (s *Session) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}

	err := s.DB.Close()
	s.DB = nil
	return err
}
