package qws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	Id         qsql.LazyInt    `db:"id"`
	Name       qsql.LazyString `db:"username"`
	Decoration qsql.LazyString `db:"decoration"`
	Pass       qsql.LazyString `json:"password" db:"password"`
	Inventory  qsql.LazyInt    `db:"inventory"`
	Email      qsql.LazyString `json:"email" db:"email"`
	AdminLvl   AdminLevel      `db:"admin_level"`
	Token      qsql.LazyString `db:"token"`
	TokenSent  qsql.LazyUnix   `db:"token_sent"`
	Online     map[string]UserList[*UserConn]
	Blocked    map[string]struct{}
	Invites    []*Invitation
	Lock       *lock.Lock
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

func timeAgo(t time.Time) string {
	elapsed := time.Since(t)

	switch {
	case elapsed < 2*time.Second:
		return "just now"
	case elapsed < 2*time.Minute:
		return fmt.Sprintf("%d seconds ago", int(elapsed/time.Second))
	case elapsed < 2*time.Hour:
		return fmt.Sprintf("%d minutes ago", int(elapsed/time.Minute))
	case elapsed < 2*time.Hour*24:
		return fmt.Sprintf("%d hours ago", int(elapsed/time.Hour))
	default:
		return fmt.Sprintf("%d days ago", int(elapsed/(time.Hour*24)))
	}
}

type loginData struct {
	UpdatedAt qsql.LazyTime `db:"updated_at"`
	Ip        string        `db:"ip"`
}

func (u *User) LastSeenMessage(ctx context.Context, ip string) string {
	if u.Id == 0 {
		return ""
	}
	var lastSeen loginData
	err := qdb.DB.GetContext(ctx, &lastSeen, "SELECT updated_at,ip FROM user_ips WHERE user_id=? ORDER BY updated_at DESC", u.Id)
	if logger.CheckP(err, "Get last seen for user "+string(u.Name)) {
		return ""
	}
	if lastSeen.Ip == ip {
		return fmt.Sprintf("You last connected %s from this IP.", timeAgo(lastSeen.UpdatedAt.Time))
	}
	return fmt.Sprintf("You last connected %s from another IP.", timeAgo(lastSeen.UpdatedAt.Time))
}

func LookupIp(ctx context.Context, ip string) string {
	matches := make([]string, 0, 2)
	err := qdb.DB.SelectContext(ctx, &matches, `
	SELECT DISTINCT username FROM users INNER JOIN user_ips ON users.id=user_ips.user_id
	WHERE ip=?
	ORDER BY user_ips.updated_at DESC`,
		ip)
	if logger.CheckP(err, "Lookup IP "+ip+":") || len(matches) == 0 {
		return "No users found."
	}
	return "Known users: " + strings.Join(matches, ", ")
}

func LookupUser(ctx context.Context, c *UserConn, name string) string {
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

func SetUserDecoration(ctx context.Context, c *UserConn, decoration string) {
	if c.User.Decoration == "" {
		logger.Error("Set invalid user decoration for user " + c.Name() + ": " + decoration)
		return
	}
	_, err := qdb.DB.ExecContext(ctx, "UPDATE users SET decoration=? WHERE id=?", decoration, c.UserId())
	logger.CheckP(err, "Set user decoration for user "+c.Name())
	c.User.Decoration = qsql.LazyString(decoration)
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
