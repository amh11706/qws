package qws

import (
	"context"
	"encoding/json"

	"github.com/amh11706/logger"
)

var cmdLogger = NewCommandLogger("commands", commandLog{})

type CmdHandler func(ctx context.Context, c UserConner, params []string) string

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

func (r *CmdRouter) ServeWS(ctx context.Context, c *UserConn, m *RawMessage) {
	if len(m.Data) < 1 {
		return
	}
	var input string
	logger.CheckP(json.Unmarshal(m.Data, &input), "Command unmarshal:")
	cmd := ""
	for cursor := 1; cursor < len(input); cursor++ {
		if input[cursor] == ' ' {
			cmd = input[:cursor]
			input = input[cursor+1:]
			break
		}
	}
	if cmd == "" {
		cmd = string(input)
		input = ""
	}

	log := cmdLogger.Start(c, cmd, input)
	defer log.End(ctx)

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
				last = i + 1
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

	if res := match.Handler(ctx, c, params); len(res) > 0 {
		c.SendInfo(ctx, res)
		log.Status(res)
	} else {
		log.Status("Success")
	}
}

func sendList(ctx context.Context, c UserConner, _ []string) string {
	return "Unknown command. See the list to the left of the input box for valid commands."
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
