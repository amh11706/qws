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
	From   string     `json:"f"`
	Admin  AdminLevel `json:"a"`
	Type   byte       `json:"ty"`
	Target int64      `json:"tg"`
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
	Online    map[string]UserList[*UserConn]
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

func (u *User) AddIp(ctx context.Context, ip string) {
	_, err := qdb.DB.ExecContext(ctx, "INSERT INTO user_ips (user_id,ip) VALUES (?,?) ON DUPLICATE KEY UPDATE updated_at=NOW()", u.Id, ip)
	logger.CheckP(err, "Add user ip for user "+string(u.Name))
}

func LookupUser(ctx context.Context, c *UserConn, params []string) string {
	name := FormatName(params[0])
	var id int64
	err := qdb.DB.GetContext(ctx, &id, "SELECT id FROM users WHERE username=?", name)
	if logger.CheckP(err, "Lookup user "+name+":") || id == 0 {
		return "User '" + name + "' not found"
	}
	matches := make([]string, 0, 2)
	err = qdb.DB.SelectContext(ctx, &matches, `
	SELECT DISTINCT username FROM users INNER JOIN user_ips ON users.id=user_ips.user_id
	WHERE ip IN (SELECT ip FROM user_ips WHERE user_id=?) AND users.id!=?
	ORDER BY user_ips.updated_at DESC`,
		id, id)
	if logger.CheckP(err, "Lookup user "+name+":") || len(matches) == 0 {
		return "No aliases found for " + name
	}
	return "Known aliases for " + name + ": " + strings.Join(matches, ", ")
}

func (u *User) SaveSeen(ctx context.Context) {
	_, err := qdb.DB.ExecContext(ctx, "UPDATE users SET last_seen=NOW() WHERE id=?", u.Id)
	logger.CheckP(err, fmt.Sprintf("Saving user %d:", u.Id))
}

func (u *User) IsBlocked(c UserConner) bool {
	if u.Blocked == nil {
		return false
	}
	var name string
	if c.UserId() == 0 {
		name = c.PrintName()
	} else {
		name = c.Name()
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
