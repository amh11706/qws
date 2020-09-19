package qws

import (
	"github.com/amh11706/logger"
	"github.com/amh11706/qws/outcmds"
)

type CmdHandler func(c *UserConn, params []string) string

type Command struct {
	Base    string     `json:"base"`
	Params  string     `json:"params"`
	Help    string     `json:"help"`
	Handler CmdHandler `json:"-"`
}

var HelpCmd = Command{"/help", "", "List all available commands.", sendList}

type CommandMessage struct {
	Type    byte       `json:"type"`
	Message *CmdRouter `json:"message"`
}

type CmdRouter struct {
	Global     []Command `json:"global"`
	Lobby      []Command `json:"lobby"`
	LobbyAdmin []Command `json:"lobbyAdmin"`
}

func (r *CmdRouter) ServeWS(c *UserConn, m *RawMessage) {
	if len(m.Data) < 3 {
		return
	}
	input := []byte(m.Data[1 : len(m.Data)-1])
	cmd := ""
	for cursor := 1; cursor < len(input); cursor++ {
		if input[cursor] == ' ' {
			cmd = string(input[:cursor])
			input = input[cursor+1:]
			break
		}
	}
	if cmd == "" {
		cmd = string(input)
		input = nil
	}
	if cmd == "/" {
		logger.Check(c.Conn.Send(outcmds.ChatMessage, &CommandMessage{Type: 6, Message: r}))
		return
	}

	match := r.findHandler(cmd)
	var params []string
	if len(match.Params) > 0 {
		wantParams := 1
		for _, c := range match.Params {
			if c == ' ' {
				wantParams++
			}
		}
		params = make([]string, 0, wantParams)
		last := 0
		for i := 0; i < len(input) && wantParams > 1; i++ {
			if input[i] == ' ' {
				params = append(params, string(input[last:i]))
				last = i
				wantParams--
			}
		}
		if len(input) > last {
			params = append(params, string(input[last:]))
			wantParams--
		}
		for wantParams > 0 {
			params = append(params, "")
			wantParams--
		}
	}

	if res := match.Handler(c, params); len(res) > 0 {
		c.Conn.SendInfo(res)
	}
}

func sendList(c *UserConn, _ []string) string {
	logger.Check(c.Conn.Send(outcmds.ChatMessage, &CommandMessage{Type: 6, Message: c.CmdRouter}))
	return ""
}

func (r *CmdRouter) findHandler(cmd string) Command {
	if cmd == "/" {
		return HelpCmd
	}

	for _, c := range r.Global {
		if len(c.Base) >= len(cmd) && c.Base[:len(cmd)] == cmd {
			return c
		}
	}
	for _, c := range r.Lobby {
		if len(c.Base) >= len(cmd) && c.Base[:len(cmd)] == cmd {
			return c
		}
	}
	for _, c := range r.LobbyAdmin {
		if len(c.Base) >= len(cmd) && c.Base[:len(cmd)] == cmd {
			return c
		}
	}
	return HelpCmd
}