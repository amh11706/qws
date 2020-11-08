package qws

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/amh11706/logger"
	"github.com/amh11706/qdb"
	"github.com/amh11706/qsql"
)

var Users = qsql.NewTable(&qdb.DB, "users")

type Invitation struct {
	From   qsql.LazyString `json:"f"`
	Type   byte            `json:"ty"`
	Target int64           `json:"tg"`
}

type User struct {
	Id        qsql.LazyInt    `db:"id"`
	Name      qsql.LazyString `db:"username"`
	Pass      qsql.LazyString `json:"password" db:"password"`
	Inventory qsql.LazyInt    `db:"inventory"`
	Email     qsql.LazyString `db:"email"`
	AdminLvl  qsql.LazyInt    `db:"admin_level"`
	Token     qsql.LazyString `db:"token"`
	TokenSent qsql.LazyTime   `db:"token_sent"`
	Online    map[string]UserList
	Blocked   map[string]struct{}
	Invites   []*Invitation
	Lock      sync.Mutex
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Name)
}

func (u *User) RemoveInvite(invite *Invitation) {
	u.Lock.Lock()
	newInvites := make([]*Invitation, 0, len(u.Invites)-1)
	for _, inv := range u.Invites {
		if inv != invite {
			newInvites = append(newInvites, inv)
		}
	}
	u.Invites = newInvites
	u.Lock.Unlock()
}

func (u *User) SaveSeen() {
	_, err := qdb.DB.Exec("UPDATE users SET last_seen=NOW() WHERE id=?", u.Id)
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
