package app

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/components/confirm"
	"github.com/lazynop/lazynf/internal/tui/components/detail"
	"github.com/lazynop/lazynf/internal/tui/components/doctor"
	"github.com/lazynop/lazynf/internal/tui/components/fontlist"
	"github.com/lazynop/lazynf/internal/tui/components/help"
	"github.com/lazynop/lazynf/internal/tui/components/logpane"
	"github.com/lazynop/lazynf/internal/tui/components/statusbar"
	"github.com/lazynop/lazynf/internal/tui/drain"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

// Overlay identifies which modal/overlay is currently on top of the layout.
type Overlay int

const (
	// OverlayNone means the main layout is visible (no modal).
	OverlayNone Overlay = iota
	// OverlayHelp shows the help overlay.
	OverlayHelp
	// OverlayConfirm shows the confirm modal.
	OverlayConfirm
	// OverlayDoctor shows the doctor full-screen pane.
	OverlayDoctor
)

// Model is the root TUI model. It owns every sub-component, dispatches every
// message, and coordinates the op lifecycle (launch, drain, complete, confirm).
type Model struct {
	engine *engine.Engine
	ctx    context.Context
	cancel context.CancelFunc
	keys   keys.KeyMap

	width, height int
	focused       messages.Pane
	overlay       Overlay

	fontlist  fontlist.Model
	detail    detail.Model
	logpane   logpane.Model
	statusbar statusbar.Model
	confirm   confirm.Model
	doctor    doctor.Model
	help      help.Model

	pending  map[int64]tea.Msg
	tokens   atomic.Int64
	inFlight map[engine.OpID]engine.OpHandle
	bootErr  error
}

// New constructs the root model. eng must already be wired with its Deps.
func New(eng *engine.Engine) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	k := keys.Default()
	return &Model{
		engine:    eng,
		ctx:       ctx,
		cancel:    cancel,
		keys:      k,
		focused:   messages.PaneFontlist,
		fontlist:  fontlist.New(k),
		detail:    detail.New(),
		logpane:   logpane.New(logpane.NewFileLogger(stateDir())),
		statusbar: statusbar.New(k),
		confirm:   confirm.New(k, 0, "", ""),
		doctor:    doctor.New(k),
		help:      help.New(k),
		pending:   map[int64]tea.Msg{},
		inFlight:  map[engine.OpID]engine.OpHandle{},
	}
}

// Init returns the initial commands: load fonts and request the window size.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadFontsCmd(), tea.RequestWindowSize)
}

// loadFontsCmd asks the engine for the current font list and wraps the result
// in a FontsLoadedMsg so the fontlist component can adopt it.
func (m *Model) loadFontsCmd() tea.Cmd {
	return func() tea.Msg {
		fonts, err := m.engine.List(m.ctx)
		return messages.FontsLoadedMsg{Fonts: fonts, Err: err}
	}
}

// Update routes every incoming message to the appropriate sub-model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = x.Width, x.Height
		m.applyLayout()
		return m, nil
	case messages.FontsLoadedMsg:
		if x.Err != nil {
			m.bootErr = x.Err
			return m, nil
		}
		fl, c := m.fontlist.Update(x)
		m.fontlist = fl.(fontlist.Model)
		return m, c
	case messages.FontStateChangedMsg:
		fl, _ := m.fontlist.Update(x)
		m.fontlist = fl.(fontlist.Model)
		d, _ := m.detail.Update(messages.FontHighlightedMsg{Font: &x.Font})
		m.detail = d.(detail.Model)
		return m, nil
	case messages.FontHighlightedMsg:
		d, _ := m.detail.Update(x)
		m.detail = d.(detail.Model)
		return m, nil
	case messages.SelectionChangedMsg:
		m.statusbar.SelectionCount = x.Count
		return m, nil
	case messages.RequestInstallMsg:
		return m.launchPerTag(x.Tags, func(tag string) engine.OpHandle {
			return m.engine.Install(m.ctx, tag, engine.InstallOptions{})
		})
	case messages.RequestUpdateMsg:
		return m.startBatchOp(m.engine.Update(m.ctx, x.Tags, engine.UpdateOptions{}))
	case messages.RequestRemoveMsg:
		return m.confirmThen(x, "Remove "+joinShort(x.Tags)+"?", "Files will be deleted.")
	case messages.RequestImportMsg:
		return m.startBatchOp(m.engine.Import(m.ctx, x.Names, engine.ImportOptions{Detect: x.Detect}))
	case messages.RequestRefreshCatalogMsg:
		return m.startBatchOp(m.engine.RefreshCatalog(m.ctx))
	case messages.RequestDoctorMsg:
		m.overlay = OverlayDoctor
		m.doctor = doctor.New(m.keys)
		m.doctor.Width, m.doctor.Height = m.width, m.height
		return m.startBatchOp(m.engine.RunDoctor(m.ctx))
	case messages.ConfirmResultMsg:
		return m.handleConfirmResult(x)
	case messages.EngineEventMsg:
		return m.routeEngineEvent(x)
	case messages.OpDoneMsg:
		delete(m.inFlight, x.OpID)
		m.statusbar.InFlight = len(m.inFlight)
		return m, nil
	case spinner.TickMsg:
		lp, c := m.logpane.Update(x)
		m.logpane = lp.(logpane.Model)
		return m, c
	case tea.KeyPressMsg:
		return m.handleKey(x)
	}
	return m, nil
}

// handleKey dispatches keypresses through the precedence: open overlays first,
// then global bindings, then the focused pane.
func (m *Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.overlay == OverlayHelp && key.Matches(k, m.keys.Help):
		m.overlay = OverlayNone
		return m, nil
	case m.overlay == OverlayHelp:
		return m, nil
	case key.Matches(k, m.keys.Quit):
		if len(m.inFlight) > 0 {
			return m.confirmQuit()
		}
		return m, tea.Quit
	case key.Matches(k, m.keys.Help):
		m.overlay = OverlayHelp
		return m, nil
	case key.Matches(k, m.keys.Doctor):
		return m, sendMsg(messages.RequestDoctorMsg{})
	case key.Matches(k, m.keys.Refresh):
		return m, sendMsg(messages.RequestRefreshCatalogMsg{})
	case key.Matches(k, m.keys.FocusNext):
		m.focused = nextPane(m.focused)
		m.applyLayout()
		return m, nil
	case key.Matches(k, m.keys.ToggleLog):
		m.logpane.Visible = !m.logpane.Visible
		m.applyLayout()
		return m, nil
	}
	if m.overlay == OverlayConfirm {
		cm, c := m.confirm.Update(k)
		m.confirm = cm.(confirm.Model)
		return m, c
	}
	if m.overlay == OverlayDoctor {
		dm, c := m.doctor.Update(k)
		m.doctor = dm.(doctor.Model)
		return m, c
	}
	switch m.focused {
	case messages.PaneFontlist:
		fl, c := m.fontlist.Update(k)
		m.fontlist = fl.(fontlist.Model)
		return m, c
	}
	return m, nil
}

// launchPerTag starts one engine op per tag and wires its drain command.
func (m *Model) launchPerTag(tags []string, factory func(string) engine.OpHandle) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	for _, tag := range tags {
		h := factory(tag)
		opID := m.assignOp(h)
		cmds = append(cmds, drain.EngineCmdCtx(m.ctx, opID, h.Events))
	}
	m.statusbar.InFlight = len(m.inFlight)
	return m, tea.Batch(cmds...)
}

// startBatchOp registers a single batch op and returns its drain command.
func (m *Model) startBatchOp(h engine.OpHandle) (tea.Model, tea.Cmd) {
	opID := m.assignOp(h)
	m.statusbar.InFlight = len(m.inFlight)
	return m, drain.EngineCmdCtx(m.ctx, opID, h.Events)
}

// assignOp allocates a fresh OpID and stores the handle in the in-flight map.
func (m *Model) assignOp(h engine.OpHandle) engine.OpID {
	id := engine.OpID(m.tokens.Add(1))
	m.inFlight[id] = h
	return id
}

// confirmThen opens the confirm overlay and stashes req keyed by a fresh token
// so handleConfirmResult can re-dispatch it on ChoiceYes.
func (m *Model) confirmThen(req tea.Msg, title, body string) (tea.Model, tea.Cmd) {
	tok := m.tokens.Add(1)
	m.pending[tok] = req
	m.confirm = confirm.New(m.keys, tok, title, body)
	m.confirm.Width, m.confirm.Height = m.width, m.height
	m.overlay = OverlayConfirm
	return m, nil
}

// confirmQuit asks the user to confirm a quit while operations are in flight.
func (m *Model) confirmQuit() (tea.Model, tea.Cmd) {
	return m.confirmThen(quitMsg{}, "Quit?", "Operations are in flight. Quit anyway?")
}

// quitMsg is the internal sentinel stashed in pending for a confirmQuit cycle.
type quitMsg struct{}

// handleConfirmResult closes the overlay and dispatches the pending request
// based on the user's choice. For RequestRemoveMsg we call engine.Remove
// directly: re-emitting the message would route it back to the case that
// opened the modal, looping forever.
func (m *Model) handleConfirmResult(x messages.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	m.overlay = OverlayNone
	pending, ok := m.pending[x.Token]
	if !ok {
		return m, nil
	}
	delete(m.pending, x.Token)
	if x.Choice != messages.ChoiceYes {
		return m, nil
	}
	switch p := pending.(type) {
	case quitMsg:
		m.cancel()
		return m, tea.Quit
	case messages.RequestRemoveMsg:
		return m.startBatchOp(m.engine.Remove(m.ctx, p.Tags, engine.RemoveOptions{Purge: p.Purge}))
	default:
		// Future: other confirmable requests.
		return m, sendMsg(pending)
	}
}

// routeEngineEvent forwards an engine event to the logpane (always) and to the
// fontlist / detail / doctor panes when the event carries relevant state, then
// re-arms the drain command to read the next event from the same op.
func (m *Model) routeEngineEvent(x messages.EngineEventMsg) (tea.Model, tea.Cmd) {
	lp, _ := m.logpane.Update(x)
	m.logpane = lp.(logpane.Model)

	if ce, ok := x.Ev.(engine.CompletedEvent); ok && ce.NewState != nil {
		fl, _ := m.fontlist.Update(messages.FontStateChangedMsg{Font: *ce.NewState})
		m.fontlist = fl.(fontlist.Model)
		d, _ := m.detail.Update(messages.FontHighlightedMsg{Font: ce.NewState})
		m.detail = d.(detail.Model)
	}
	if ds, ok := x.Ev.(engine.DoctorSectionEvent); ok {
		dm, _ := m.doctor.Update(messages.DoctorSectionMsg{
			OpID: ds.OpID, Section: ds.Section, Title: ds.Title,
			Status: ds.Status, Detail: ds.Detail, Hint: ds.Hint, Action: ds.Action,
		})
		m.doctor = dm.(doctor.Model)
	}

	if h, ok := m.inFlight[x.OpID]; ok {
		return m, drain.EngineCmdCtx(m.ctx, x.OpID, h.Events)
	}
	return m, nil
}

// applyLayout recomputes pane sizes from the current width / height and pushes
// them into the sub-components along with their focus flags.
func (m *Model) applyLayout() {
	l := Compute(m.width, m.height, m.logpane.Visible)
	m.fontlist.Width, m.fontlist.Height = l.ListW, l.ListH
	m.detail.Width, m.detail.Height = l.DetailW, l.DetailH
	m.logpane.Width, m.logpane.Height = l.LogW, l.LogH
	m.statusbar.Width = l.StatusW
	m.confirm.Width, m.confirm.Height = m.width, m.height
	m.help.Width, m.help.Height = m.width, m.height
	m.doctor.Width, m.doctor.Height = m.width, m.height

	m.fontlist.Focused = m.focused == messages.PaneFontlist
	m.detail.Focused = m.focused == messages.PaneDetail
}

// View composes the layout. Overlays short-circuit the main composition.
func (m *Model) View() tea.View {
	if m.bootErr != nil {
		v := tea.NewView("error loading font list: " + m.bootErr.Error() + "\n\npress q to quit")
		v.AltScreen = true
		return v
	}
	switch m.overlay {
	case OverlayHelp:
		v := m.help.View()
		v.AltScreen = true
		return v
	case OverlayConfirm:
		v := m.confirm.View()
		v.AltScreen = true
		return v
	case OverlayDoctor:
		v := m.doctor.View()
		v.AltScreen = true
		return v
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top,
		m.fontlist.View().Content, m.detail.View().Content)
	v := tea.NewView(top + "\n" + m.logpane.View().Content + "\n" + m.statusbar.View().Content)
	v.AltScreen = true
	return v
}

// nextPane cycles focus between the fontlist and the detail pane.
func nextPane(p messages.Pane) messages.Pane {
	if p == messages.PaneFontlist {
		return messages.PaneDetail
	}
	return messages.PaneFontlist
}

// joinShort renders a comma-separated list of tags, collapsing long lists into
// "first, second, ... +N" so the confirm modal stays readable.
func joinShort(tags []string) string {
	if len(tags) > 3 {
		return tags[0] + ", " + tags[1] + ", ... +" + strconv.Itoa(len(tags)-2)
	}
	return strings.Join(tags, ", ")
}

// sendMsg wraps a tea.Msg into a tea.Cmd so it can be returned from Update.
func sendMsg(msg tea.Msg) tea.Cmd { return func() tea.Msg { return msg } }

// stateDir returns $XDG_STATE_HOME or the conventional ~/.local/state fallback.
func stateDir() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return home + "/.local/state"
}
