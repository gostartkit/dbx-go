package driver

import (
	"context"
	"database/sql"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func OpenMySQL(ctx context.Context, cfg *config.ConnectionConfig) (*sql.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, util.WrapLayer("config", "invalid MySQL connection config", err)
	}

	password, err := cfg.PasswordValue()
	if err != nil {
		return nil, util.WrapLayer("config", "read MySQL password", err)
	}

	dsn := mysql.NewConfig()
	dsn.User = cfg.User
	dsn.Passwd = password
	dsn.Addr = cfg.Address()
	dsn.Net = "tcp"
	dsn.AllowNativePasswords = true
	dsn.Params = map[string]string{
		"charset": "utf8mb4",
	}

	if cfg.Mode == "proxy" {
		dsn.Net, err = registerProxyDialer(cfg)
		if err != nil {
			return nil, util.WrapLayer("proxy", "prepare proxy dialer", err)
		}
	}
	if cfg.UsesSSH() {
		dsn.Net, err = registerSSHDialer(cfg)
		if err != nil {
			return nil, util.WrapLayer("ssh", "prepare SSH tunnel", err)
		}
	}

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, util.WrapLayer("mysql", "create MySQL client", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout())
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, util.WrapLayer("mysql", "ping database", err)
	}

	return db, nil
}
