package session

import (
	"fmt"

	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util"
)

func newDbStorage(cf *Config, opts *Options) (storage, error) {
	db := opts.db

	if db == nil {
		return nil, fmt.Errorf("unable get db storage")
	}

	fn := func() {
		db.Exec(
			"delete from `"+cf.TableName+"` where updated_at<? and cookie_name=?",
			opts.clock.Now().Add(-cf.MaxIdleTime.Duration),
			cf.CookieName,
		)
	}
	util.UntilWithTick(fn, opts.clock.NewTicker(cf.GcInterval.Duration).C(), opts.ctx.Done())

	if cf.AutoMigrate {
		if err := db.AutoMigrate(&sessionConn{}, orm.WithTable(cf.TableName)); err != nil {
			return nil, err
		}
	}

	return &dbStorage{config: cf, db: db}, nil
}

type dbStorage struct {
	db     orm.DB
	config *Config
}

func (p *dbStorage) all() (ret int) {
	err := p.db.Query("select count(*) from `"+p.config.TableName+"` where cookie_name=?",
		p.config.CookieName).Row(&ret)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	return
}

func (p *dbStorage) get(sid string) (ret *sessionConn, err error) {
	err = p.db.Query("select * from session where sid=?", sid).Row(&ret)
	return
}

func (p *dbStorage) insert(s *sessionConn) error {
	return p.db.Insert(s, orm.WithTable(p.config.TableName))
}

func (p *dbStorage) del(sid string) error {
	return p.db.ExecNumErr("DELETE FROM `"+p.config.TableName+"` where sid=?", sid)
}

func (p *dbStorage) update(s *sessionConn) error {
	return p.db.Update(s, orm.WithTable(p.config.TableName))
}
