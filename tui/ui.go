package tui

import (
	"df/core"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3C3C3C")).
			Padding(0, 1)
)

type state int

const (
	menuState state = iota
	scanningState
	resultsState
	configState
	editingConfigState
	filesState
	addingPathState
	messageState
	errorState
)

type messageMsg struct {
	title   string
	content string
}

type item struct {
	title, desc string
	id          string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type fileItem struct {
	path string
	size string
}

func (i fileItem) Title() string       { return i.path }
func (i fileItem) Description() string { return i.size }
func (i fileItem) FilterValue() string { return i.path }

type configItem struct {
	key   string
	value string
	id    string
}

func (i configItem) Title() string       { return i.key }
func (i configItem) Description() string { return i.value }
func (i configItem) FilterValue() string { return i.key }

type mainModel struct {
	state           state
	list            list.Model
	filesList       list.Model
	configList      list.Model
	spinner         spinner.Model
	textInput       textinput.Model
	app             *core.App
	err             error
	results         []core.ResultList
	files           []*core.FileItem
	editingConfigID string
	msgTitle        string
	msgContent      string
	quitting        bool
	width           int
	height          int
}

func NewModel(app *core.App) mainModel {
	items := []list.Item{
		item{title: "Start Scan", desc: "Scan for duplicate files", id: "scan"},
		item{title: "Show Indexed Files", desc: "List all files in the database", id: "files"},
		item{title: "Show Duplicates", desc: "Show duplicates found in previous scans", id: "dupes"},
		item{title: "Purge Index", desc: "Remove non-existing files from database", id: "purge"},
		item{title: "Update Index", desc: "Update file hashes in the database", id: "update"},
		item{title: "Clear Index", desc: "Clear all files from the database", id: "clear"},
		item{title: "Config", desc: "Show current configuration", id: "config"},
		item{title: "Exit", desc: "Quit the application", id: "exit"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "DupeFiles TUI"
	l.Styles.Title = titleStyle

	fl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fl.Title = "Indexed Files"
	fl.Styles.Title = titleStyle
	fl.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "add path"),
			),
			key.NewBinding(
				key.WithKeys("x", "delete"),
				key.WithHelp("x", "remove path"),
			),
		}
	}

	cl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	cl.Title = "Configuration"
	cl.Styles.Title = titleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter path to add..."

	return mainModel{
		state:      menuState,
		list:       l,
		filesList:  fl,
		configList: cl,
		spinner:    s,
		textInput:  ti,
		app:        app,
	}
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

type scanFinishedMsg struct {
	results []core.ResultList
	err     error
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		m.filesList.SetSize(msg.Width, msg.Height-4)
		m.configList.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case scanFinishedMsg:
		m.state = resultsState
		m.results = msg.results
		m.err = msg.err
		return m, nil

	case messageMsg:
		m.state = messageState
		m.msgTitle = msg.title
		m.msgContent = msg.content
		return m, nil

	case filesLoadedMsg:
		m.filesList.SetItems(msg.items)
		return m, nil

	case configLoadedMsg:
		m.configList.SetItems(msg.items)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.state {
	case menuState:
		return m.updateMenu(msg)
	case scanningState:
		return m.updateScanning(msg)
	case resultsState:
		return m.updateSubView(msg)
	case filesState:
		return m.updateFiles(msg)
	case addingPathState:
		return m.updateAddingPath(msg)
	case configState:
		return m.updateConfig(msg)
	case editingConfigState:
		return m.updateEditingConfig(msg)
	case messageState:
		return m.updateSubView(msg)
	}

	return m, nil
}

func (m mainModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				switch i.id {
				case "scan":
					m.state = scanningState
					return m, tea.Batch(m.startScan(), m.spinner.Tick)
				case "files":
					m.state = filesState
					return m, m.refreshFiles()
				case "dupes":
					m.state = resultsState
					m.results = m.getDupes()
					return m, nil
				case "config":
					m.state = configState
					return m, m.refreshConfig()
				case "purge":
					return m, m.purgeIndex()
				case "update":
					return m, m.updateIndex()
				case "clear":
					return m, m.clearIndex()
				case "exit":
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m mainModel) updateFiles(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.state = menuState
			return m, nil
		case "a":
			m.state = addingPathState
			m.textInput.Focus()
			m.textInput.SetValue("")
			return m, nil
		case "x", "delete":
			i, ok := m.filesList.SelectedItem().(fileItem)
			if ok {
				return m, m.removePath(i.path)
			}
		}
	}

	var cmd tea.Cmd
	m.filesList, cmd = m.filesList.Update(msg)
	return m, cmd
}

func (m mainModel) updateAddingPath(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = filesState
			return m, nil
		case "enter":
			path := m.textInput.Value()
			if path != "" {
				return m, m.addPath(path)
			}
			m.state = filesState
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m mainModel) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.state = menuState
			return m, nil
		case "enter", " ":
			i, ok := m.configList.SelectedItem().(configItem)
			if ok {
				if i.id == "minfilesize" || i.id == "samplesize" {
					m.state = editingConfigState
					m.editingConfigID = i.id
					m.textInput.Focus()
					// Extract number from value string "1024 bytes"
					val := strings.Split(i.value, " ")[0]
					m.textInput.SetValue(val)
					return m, nil
				}
				return m, m.toggleConfig(i.id)
			}
		}
	}

	var cmd tea.Cmd
	m.configList, cmd = m.configList.Update(msg)
	return m, cmd
}

func (m mainModel) updateEditingConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = configState
			return m, nil
		case "enter":
			val := m.textInput.Value()
			return m, m.setConfig(m.editingConfigID, val)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m mainModel) setConfig(id, val string) tea.Cmd {
	return func() tea.Msg {
		c := m.app.GetConfig()
		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return messageMsg{title: "Error", content: "Invalid number"}
		}

		switch id {
		case "minfilesize":
			c.MinFileSize = num
		case "samplesize":
			c.SampleSizeBinaryCompare = int(num)
		}
		m.state = configState
		return m.refreshConfig()()
	}
}

func (m mainModel) refreshFiles() tea.Cmd {
	return func() tea.Msg {
		files := m.app.GetIndex().GetAllFiles()
		items := make([]list.Item, len(files))
		for i, f := range files {
			items[i] = fileItem{path: f.Path, size: f.HumanizedSize}
		}
		return filesLoadedMsg{items}
	}
}

type filesLoadedMsg struct{ items []list.Item }

func (m mainModel) refreshConfig() tea.Cmd {
	return func() tea.Msg {
		c := m.app.GetConfig()
		items := []list.Item{
			configItem{key: "Debug", value: fmt.Sprintf("%v", c.Debug), id: "debug"},
			configItem{key: "DryRun", value: fmt.Sprintf("%v", c.DryRun), id: "dryrun"},
			configItem{key: "MinFileSize", value: fmt.Sprintf("%d bytes", c.MinFileSize), id: "minfilesize"},
			configItem{key: "SampleSizeBinaryCompare", value: fmt.Sprintf("%d bytes", c.SampleSizeBinaryCompare), id: "samplesize"},
			configItem{key: "Database", value: m.app.GetIndex().GetIndexPath(), id: "database"},
		}
		return configLoadedMsg{items}
	}
}

type configLoadedMsg struct{ items []list.Item }

func (m mainModel) removePath(path string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.app.RemovePathFromIndex(path)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error removing path: %v", err)}
		}
		return m.refreshFiles()()
	}
}

func (m mainModel) addPath(path string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.app.AddPathToIndex(path, true, "")
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error adding path: %v", err)}
		}
		m.state = filesState
		return m.refreshFiles()()
	}
}

func (m mainModel) toggleConfig(id string) tea.Cmd {
	return func() tea.Msg {
		c := m.app.GetConfig()
		switch id {
		case "debug":
			c.Debug = !c.Debug
		case "dryrun":
			c.DryRun = !c.DryRun
		case "minfilesize":
			// For now just toggle common values or show message
			return messageMsg{title: "Config", content: "Editing numeric values not fully implemented yet."}
		case "samplesize":
			return messageMsg{title: "Config", content: "Editing numeric values not fully implemented yet."}
		default:
			return nil
		}
		return m.refreshConfig()()
	}
}

func (m mainModel) getDupes() []core.ResultList {
	// core.ResultList is not what GetAllDupes returns.
	// GetAllDupes returns []*FileItem.
	// We might need to reconstruct ResultList or just change how we display them.
	// Actually, let's just use the scanner to get the results again if needed,
	// or look at how App.ShowDupes does it.

	// App.ShowDupes just prints the files.
	// Let's see if we can get the groups.

	// For now, let's just return what we have in results or empty
	return m.results
}

func (m mainModel) updateScanning(msg tea.Msg) (tea.Model, tea.Cmd) {
	// For now, scanning is synchronous in the background goroutine
	// but we don't have real progress updates from core yet.
	return m, nil
}

func (m mainModel) updateSubView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "backspace" || msg.String() == "q" {
			m.state = menuState
			return m, nil
		}
	}
	return m, nil
}

func (m mainModel) View() string {
	if m.quitting {
		return ""
	}

	header := titleStyle.Render(" DupeFiles v0.1.4 ")

	// Get file count for footer
	fileCount := 0
	if m.app != nil && m.app.GetIndex() != nil {
		fileCount = len(m.app.GetIndex().GetAllFiles())
	}
	footer := statusStyle.Render(fmt.Sprintf(" %d files indexed ", fileCount))

	var content string
	switch m.state {
	case menuState:
		content = m.list.View()
	case scanningState:
		content = fmt.Sprintf("\n\n  %s Scanning for duplicates...\n\n  Press q to cancel", m.spinner.View())
	case resultsState:
		if m.err != nil {
			content = fmt.Sprintf("\n\n  Error: %v\n\n  Press esc to go back", m.err)
		} else if len(m.results) == 0 {
			content = "\n\n  No duplicates found.\n\n  Press esc to go back"
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\n  Found %d duplicate groups:\n\n", len(m.results)))
			for i, res := range m.results {
				if i > 15 {
					sb.WriteString("  ... and more\n")
					break
				}
				sb.WriteString(fmt.Sprintf("  Group %d (Hash: %s...)\n", i+1, res.HashSum[:8]))
				for _, guid := range res.FileGuids {
					file := m.app.GetIndex().GetFileByGuid(guid)
					if file != nil {
						sb.WriteString(fmt.Sprintf("    - %s (%s)\n", file.Path, file.HumanizedSize))
					}
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n  Press esc to go back")
			content = sb.String()
		}
	case filesState:
		content = m.filesList.View()
	case addingPathState:
		content = fmt.Sprintf("\n  Add Path\n\n%s\n\n(esc to cancel, enter to confirm)", m.textInput.View())
	case configState:
		content = m.configList.View()
	case messageState:
		content = fmt.Sprintf("\n  %s\n\n  %s\n\n  Press esc to go back", m.msgTitle, m.msgContent)
	}

	// Simple vertical join with some padding
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m mainModel) startScan() tea.Cmd {
	return func() tea.Msg {
		scanner := core.NewScanner(m.app.GetIndex())
		results, err := scanner.ScanForDuplicates()
		return scanFinishedMsg{results: results, err: err}
	}
}

func (m mainModel) purgeIndex() tea.Cmd {
	return func() tea.Msg {
		count, err := m.app.GetIndex().Purge()
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error purging index: %v", err)}
		}
		return messageMsg{title: "Success", content: fmt.Sprintf("Purged %d files from the database", count)}
	}
}

func (m mainModel) updateIndex() tea.Cmd {
	return func() tea.Msg {
		count, err := m.app.GetIndex().Update()
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error updating index: %v", err)}
		}
		return messageMsg{title: "Success", content: fmt.Sprintf("Updated %d files in the database", count)}
	}
}

func (m mainModel) clearIndex() tea.Cmd {
	return func() tea.Msg {
		// core.App.IndexClear() does a lot of things.
		// Let's just use the index method if possible or call App.IndexClear() but it prints.
		// For now let's just use a simple message as it might be destructive.
		return messageMsg{title: "Notice", content: "Clear Index not fully implemented in TUI yet for safety."}
	}
}
