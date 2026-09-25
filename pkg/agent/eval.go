package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/models"
)

// EvalCase is a frozen acceptance case. Expect is the standard; Script is only
// a stand-in model for CI. Live-model eval uses User and Ignore Script.
type EvalCase struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	User      string       `json:"user"`
	Tags      []string     `json:"tags,omitempty"`
	WriteMode string       `json:"writeMode,omitempty"`
	Product   *evalProduct `json:"product,omitempty"`
	NoPerms   bool         `json:"noPerms,omitempty"`
	Script    []evalTurn   `json:"script,omitempty"`
	Expect    EvalExpect   `json:"expect"`
}

type evalProduct struct {
	ID          string `json:"id"`
	NetworkType string `json:"networkType,omitempty"`
	Script      string `json:"script,omitempty"`
	State       bool   `json:"state,omitempty"`
}

type evalTurn struct {
	Tool    string          `json:"tool,omitempty"`
	Args    json.RawMessage `json:"args,omitempty"`
	Content string          `json:"content,omitempty"`
}

// EvalExpect is the baseline: what must hold after a turn, regardless of model.
type EvalExpect struct {
	RunStatus   string                    `json:"runStatus,omitempty"`
	MustCall    []string                  `json:"mustCall,omitempty"`
	MustNotCall []string                  `json:"mustNotCall,omitempty"`
	CallArgs    map[string]map[string]any `json:"callArgs,omitempty"`
	DraftTools  []string                  `json:"draftTools,omitempty"`
	ToolOK      map[string]bool           `json:"toolOK,omitempty"`
}

type EvalTrace struct {
	RunStatus string
	Calls     []EvalCall
	Drafts    []string // pending tool names, order of appearance
	Results   []EvalToolResult
}

type EvalCall struct {
	Name string
	Args map[string]any
}

type EvalToolResult struct {
	Name string
	OK   bool
	Msg  string
}

type EvalCheck struct {
	Name string `json:"name"`
	Pass bool   `json:"pass"`
	Got  string `json:"got,omitempty"`
	Want string `json:"want,omitempty"`
}

type EvalScore struct {
	ID     string      `json:"id"`
	Pass   bool        `json:"pass"`
	Checks []EvalCheck `json:"checks"`
}

func ExtractTrace(store Store, conv *models.AgentConversation) EvalTrace {
	tr := EvalTrace{}
	if conv != nil {
		tr.RunStatus = conv.RunStatus
	}
	if store == nil || conv == nil {
		return tr
	}
	fresh, _ := store.GetConversation(conv.Id)
	if fresh != nil {
		tr.RunStatus = fresh.RunStatus
	}
	msgs, _ := store.ListMessages(conv.Id)
	for _, m := range msgs {
		if m.Role == RoleAssistant {
			var obj map[string]any
			if json.Unmarshal([]byte(m.Payload), &obj) != nil {
				continue
			}
			for _, c := range extractCalls(obj) {
				args := map[string]any{}
				fn, _ := c["function"].(map[string]any)
				if fn != nil {
					raw := asString(fn["arguments"])
					_ = json.Unmarshal([]byte(raw), &args)
				}
				tr.Calls = append(tr.Calls, EvalCall{Name: callName(c), Args: args})
			}
		}
		if m.Role == RoleTool {
			var obj map[string]any
			ok := false
			msg := ""
			if json.Unmarshal([]byte(m.Content), &obj) == nil {
				ok, _ = obj["ok"].(bool)
				msg = asString(obj["message"])
			}
			tr.Results = append(tr.Results, EvalToolResult{Name: m.ToolName, OK: ok, Msg: msg})
		}
	}
	drafts, _ := store.ListDrafts(conv.Id)
	for _, d := range PendingDrafts(drafts) {
		tr.Drafts = append(tr.Drafts, d.ToolName)
	}
	return tr
}

func ScoreTrace(tr EvalTrace, exp EvalExpect) EvalScore {
	s := EvalScore{Pass: true}
	add := func(c EvalCheck) {
		s.Checks = append(s.Checks, c)
		if !c.Pass {
			s.Pass = false
		}
	}
	if exp.RunStatus != "" {
		add(EvalCheck{Name: "runStatus", Pass: tr.RunStatus == exp.RunStatus, Got: tr.RunStatus, Want: exp.RunStatus})
	}
	called := map[string]EvalCall{}
	var names []string
	for _, c := range tr.Calls {
		called[c.Name] = c
		names = append(names, c.Name)
	}
	for _, want := range exp.MustCall {
		_, ok := called[want]
		add(EvalCheck{Name: "mustCall:" + want, Pass: ok, Got: strings.Join(names, ","), Want: want})
	}
	for _, ban := range exp.MustNotCall {
		_, ok := called[ban]
		add(EvalCheck{Name: "mustNotCall:" + ban, Pass: !ok, Got: strings.Join(names, ","), Want: "absent"})
	}
	for tool, wantArgs := range exp.CallArgs {
		c, ok := called[tool]
		if !ok {
			add(EvalCheck{Name: "callArgs:" + tool, Pass: false, Got: "not called", Want: tool})
			continue
		}
		for k, wv := range wantArgs {
			gv := c.Args[k]
			pass := fmt.Sprint(gv) == fmt.Sprint(wv)
			add(EvalCheck{Name: "callArgs:" + tool + "." + k, Pass: pass, Got: fmt.Sprint(gv), Want: fmt.Sprint(wv)})
		}
	}
	if exp.DraftTools != nil {
		got := strings.Join(tr.Drafts, ",")
		want := strings.Join(exp.DraftTools, ",")
		add(EvalCheck{Name: "draftTools", Pass: sameNames(tr.Drafts, exp.DraftTools), Got: got, Want: want})
	}
	lastOK := map[string]EvalToolResult{}
	for _, r := range tr.Results {
		lastOK[r.Name] = r
	}
	for tool, wantOK := range exp.ToolOK {
		r, ok := lastOK[tool]
		if !ok {
			add(EvalCheck{Name: "toolOK:" + tool, Pass: false, Got: "no result", Want: fmt.Sprint(wantOK)})
			continue
		}
		add(EvalCheck{Name: "toolOK:" + tool, Pass: r.OK == wantOK, Got: fmt.Sprintf("ok=%v msg=%s", r.OK, r.Msg), Want: fmt.Sprint(wantOK)})
	}
	return s
}

func sameNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	a := append([]string{}, got...)
	b := append([]string{}, want...)
	// order-insensitive: HITL may not guarantee draft order across tools
	if len(a) <= 1 {
		return len(a) == 0 || a[0] == b[0]
	}
	count := map[string]int{}
	for _, x := range a {
		count[x]++
	}
	for _, x := range b {
		count[x]--
		if count[x] < 0 {
			return false
		}
	}
	for _, n := range count {
		if n != 0 {
			return false
		}
	}
	return true
}

func (s EvalScore) Failures() []EvalCheck {
	var out []EvalCheck
	for _, c := range s.Checks {
		if !c.Pass {
			out = append(out, c)
		}
	}
	return out
}
