package qws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/amh11706/logger"
	"github.com/amh11706/qdb"
	"github.com/amh11706/qsql"
	"github.com/amh11706/qws/lock"
)

var Users = qsql.NewTable(&qdb.DB, "users")

type Invitation struct {
	From   qsql.LazyString `json:"f"`
	Admin  AdminLevel      `json:"a"`
	Type   byte            `json:"ty"`
	Target int64           `json:"tg"`
}

type AdminLevel qsql.LazyInt

func (a *AdminLevel) Scan(src interface{}) error {
	li := qsql.LazyInt(0)
	err := li.Scan(src)
	*a = AdminLevel(li)
	return err
}

const (
	AdminLevelUser AdminLevel = iota
	AdminLevelMapCreator
	AdminLevelMod
	AdminLevelAdmin
	AdminLevelSuperAdmin
)

type User struct {
	Id        qsql.LazyInt    `db:"id"`
	Name      qsql.LazyString `db:"username"`
	Pass      qsql.LazyString `json:"password" db:"password"`
	Inventory qsql.LazyInt    `db:"inventory"`
	Email     qsql.LazyString `json:"email" db:"email"`
	AdminLvl  AdminLevel      `db:"admin_level"`
	Token     qsql.LazyString `db:"token"`
	TokenSent qsql.LazyUnix   `db:"token_sent"`
	Online    map[string]UserList
	Blocked   map[string]struct{}
	Invites   []*Invitation
	Lock      *lock.Lock
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Name)
}

func (u *User) RemoveInvite(ctx context.Context, invite *Invitation) {
	u.Lock.MustLock(ctx)
	defer u.Lock.Unlock()
	newInvites := make([]*Invitation, 0, len(u.Invites)-1)
	for _, inv := range u.Invites {
		if inv != invite {
			newInvites = append(newInvites, inv)
		}
	}
	u.Invites = newInvites
}

func (u *User) SaveSeen(ctx context.Context) {
	_, err := qdb.DB.ExecContext(ctx, "UPDATE users SET last_seen=NOW() WHERE id=?", u.Id)
	logger.CheckP(err, fmt.Sprintf("Saving user %d:", u.Id))
}

func (u *User) IsBlocked(c *UserConn) bool {
	if u.Blocked == nil {
		return false
	}
	var name string
	if c.User.Id == 0 {
		name = c.PrintName()
	} else {
		name = string(c.User.Name)
	}
	_, b := u.Blocked[name]
	return b
}

func FormatName(n string) string {
	if n == "" {
		return ""
	}
	return strings.ToUpper(n[:1]) + strings.ToLower(n[1:])
}

func ParseName(n string) (string, int64) {
	if i := strings.Index(n, "("); i != -1 {
		copy, err := strconv.Atoi(n[i+1 : len(n)-1])
		if err != nil {
			return FormatName(n[:i]), -1
		}
		return FormatName(n[:i]), int64(copy)
	}
	return FormatName(n), -1
}
