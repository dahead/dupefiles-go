package tui

import (
	"df/core"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
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

	pinkTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#FF5FD7")).
			Padding(0, 1)

	greenTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#00AF5F")).
			Padding(0, 1)

	blueTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#005FDF")).
			Padding(0, 1)

	orangeTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#FFAF00")).
				Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3C3C3C")).
			Padding(0, 1)

	hashColors = []lipgloss.Color{
		lipgloss.Color("#88d2ff"),
		lipgloss.Color("#ff88d2"),
		lipgloss.Color("#d2ff88"),
		lipgloss.Color("#ff9d88"),
		lipgloss.Color("#88ff9d"),
		lipgloss.Color("#9d88ff"),
		lipgloss.Color("#ffd288"),
		lipgloss.Color("#88ffd2"),
	}
)

type state int

const (
	menuState state = iota
	scanningState
	resultsState
	configState
	editingConfigState
	filesState
	indexState
	addingPathState
	removingPathState
	confirmClearState
	messageState
	errorState
	dupesState
	filteringState
	movingState
	browsingState
)

type messageMsg struct {
	title   string
	content string
	results []core.ResultList
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

type resultItem struct {
	isHeader bool
	hash     string
	file     *core.FileItem
	group    *core.ResultList
}

func (i configItem) Title() string       { return i.key }
func (i configItem) Description() string { return i.value }
func (i configItem) FilterValue() string { return i.key }

type mainModel struct {
	state           state
	list            list.Model
	indexList       list.Model
	filesList       list.Model
	configList      list.Model
	dupesList       list.Model
	spinner         spinner.Model
	textInput       textinput.Model
	paginator       paginator.Model
	app             *core.App
	err             error
	results         []core.ResultList
	files           []*core.FileItem
	selectedDupes   map[string]bool
	dupeFilter      string
	dupeCursor      int
	editingConfigID string
	msgTitle        string
	msgContent      string
	quitting        bool
	width           int
	height          int

	// Browser state
	browserDir      string
	browserFiles    []browserItem
	browserCursor   int
	browserSelected map[string]bool
}

type browserItem struct {
	name  string
	isDir bool
	size  int64
}

func customDelegate(color lipgloss.Color) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(color).BorderForeground(color)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(color).BorderForeground(color)
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(color)
	return d
}

func NewModel(app *core.App) mainModel {
	items := []list.Item{
		item{title: "[1] Start Scan", desc: "Scan for duplicate files", id: "scan"},
		item{title: "[2] Database (Index)", desc: "Manage indexed files and folders", id: "index"},
		item{title: "[3] Results", desc: "Show duplicates found in previous scans", id: "dupes"},
		item{title: "[4] Config", desc: "Show current configuration", id: "config"},
		item{title: "[Q] Exit", desc: "Quit the application", id: "exit"},
	}

	l := list.New(items, customDelegate(lipgloss.Color("#7D56F4")), 0, 0)
	l.Title = "DupeFiles TUI"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)

	indexItems := []list.Item{
		item{title: "[1] Add Path (Manual)", desc: "Enter a file or folder path manually", id: "add_path"},
		item{title: "[B] Add Path (Browse)", desc: "Browse for files or folders to add", id: "browse_path"},
		item{title: "[2] Remove Path", desc: "Remove a file or folder from the index", id: "remove_path"},
		item{title: "[3] Show Indexed Files", desc: "List all files in the database", id: "files"},
		item{title: "[4] Purge Index", desc: "Remove non-existing files from database", id: "purge"},
		item{title: "[5] Update Index", desc: "Update file hashes in the database", id: "update"},
		item{title: "[6] Clear Index", desc: "Clear all files from the database", id: "clear"},
	}
	il := list.New(indexItems, customDelegate(lipgloss.Color("#00AF5F")), 0, 0)
	il.Title = "Database (Index) Management"
	il.Styles.Title = greenTitleStyle
	il.SetShowStatusBar(false)

	fl := list.New([]list.Item{}, customDelegate(lipgloss.Color("#00AF5F")), 0, 0)
	fl.Title = "Indexed Files"
	fl.Styles.Title = greenTitleStyle
	fl.SetShowStatusBar(false)
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

	cl := list.New([]list.Item{}, customDelegate(lipgloss.Color("#FFAF00")), 0, 0)
	cl.Title = "Configuration"
	cl.Styles.Title = orangeTitleStyle
	cl.SetShowStatusBar(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter path to add..."

	pg := paginator.New()
	pg.Type = paginator.Dots
	pg.PerPage = 10

	return mainModel{
		state:           menuState,
		list:            l,
		indexList:       il,
		filesList:       fl,
		configList:      cl,
		spinner:         s,
		textInput:       ti,
		paginator:       pg,
		app:             app,
		selectedDupes:   make(map[string]bool),
		browserSelected: make(map[string]bool),
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
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.state == menuState {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		m.indexList.SetSize(msg.Width, msg.Height-4)
		m.filesList.SetSize(msg.Width, msg.Height-4)
		m.configList.SetSize(msg.Width, msg.Height-4)
		m.paginator.PerPage = (m.height - 12) / 4
		if m.paginator.PerPage < 1 {
			m.paginator.PerPage = 1
		}
		return m, nil

	case scanFinishedMsg:
		m.state = resultsState
		m.results = msg.results
		m.err = msg.err
		// Auto select: keep first, select others
		for _, group := range m.results {
			for i, guid := range group.FileGuids {
				if i == 0 {
					m.selectedDupes[guid] = false
				} else {
					m.selectedDupes[guid] = true
				}
			}
		}
		return m, nil

	case messageMsg:
		m.state = messageState
		m.msgTitle = msg.title
		m.msgContent = msg.content
		if msg.results != nil {
			m.results = msg.results
		}
		return m, nil

	case filesLoadedMsg:
		m.filesList.SetItems(msg.items)
		m.state = filesState
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
	case indexState:
		return m.updateIndexMenu(msg)
	case scanningState:
		return m.updateScanning(msg)
	case resultsState:
		return m.updateSubView(msg)
	case filesState:
		return m.updateFiles(msg)
	case addingPathState:
		return m.updateAddingPath(msg)
	case removingPathState:
		return m.updateRemovingPath(msg)
	case configState:
		return m.updateConfig(msg)
	case editingConfigState:
		return m.updateEditingConfig(msg)
	case confirmClearState:
		return m.updateConfirmClear(msg)
	case filteringState:
		return m.updateFiltering(msg)
	case movingState:
		return m.updateMoving(msg)
	case browsingState:
		return m.updateBrowsing(msg)
	case messageState:
		return m.updateSubView(msg)
	}

	return m, nil
}

func (m mainModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.state = scanningState
			return m, tea.Batch(m.startScan(), m.spinner.Tick)
		case "2":
			m.state = indexState
			return m, nil
		case "3":
			m.state = resultsState
			m.results = m.getDupes()
			// Auto select: keep first, select others
			for _, group := range m.results {
				for i, guid := range group.FileGuids {
					if i == 0 {
						m.selectedDupes[guid] = false
					} else {
						m.selectedDupes[guid] = true
					}
				}
			}
			return m, nil
		case "4":
			m.state = configState
			return m, m.refreshConfig()
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				switch i.id {
				case "scan":
					m.state = scanningState
					return m, tea.Batch(m.startScan(), m.spinner.Tick)
				case "index":
					m.state = indexState
					return m, nil
				case "dupes":
					m.state = resultsState
					m.results = m.getDupes()
					m.paginator.Page = 0
					m.dupeCursor = 0
					// Auto select: keep first, select others
					for _, group := range m.results {
						for i, guid := range group.FileGuids {
							if i == 0 {
								m.selectedDupes[guid] = false
							} else {
								m.selectedDupes[guid] = true
							}
						}
					}
					return m, nil
				case "config":
					m.state = configState
					return m, m.refreshConfig()
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

func (m mainModel) updateIndexMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.state = menuState
			return m, nil
		case "1":
			m.state = addingPathState
			m.textInput.Placeholder = "Enter path to add..."
			m.textInput.Focus()
			m.textInput.SetValue("")
			return m, nil
		case "b", "B":
			m.state = browsingState
			dir, _ := os.Getwd()
			m.browserDir = dir
			return m, m.loadDir(dir)
		case "2":
			m.state = removingPathState
			m.textInput.Placeholder = "Enter text to match for removal..."
			m.textInput.Focus()
			m.textInput.SetValue("")
			return m, nil
		case "3":
			m.state = filesState
			return m, m.refreshFiles()
		case "4":
			return m, m.purgeIndex()
		case "5":
			return m, m.updateIndex()
		case "6":
			m.state = confirmClearState
			return m, nil
		case "enter":
			i, ok := m.indexList.SelectedItem().(item)
			if ok {
				switch i.id {
				case "add_path":
					m.state = addingPathState
					m.textInput.Placeholder = "Enter path to add..."
					m.textInput.Focus()
					m.textInput.SetValue("")
					return m, nil
				case "browse_path":
					m.state = browsingState
					dir, _ := os.Getwd()
					m.browserDir = dir
					return m, m.loadDir(dir)
				case "remove_path":
					m.state = removingPathState
					m.textInput.Placeholder = "Enter text to match for removal..."
					m.textInput.Focus()
					m.textInput.SetValue("")
					return m, nil
				case "files":
					m.state = filesState
					return m, m.refreshFiles()
				case "purge":
					return m, m.purgeIndex()
				case "update":
					return m, m.updateIndex()
				case "clear":
					m.state = confirmClearState
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	m.indexList, cmd = m.indexList.Update(msg)
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
			m.state = indexState
			return m, nil
		case "enter":
			path := m.textInput.Value()
			if path != "" {
				m.state = indexState
				return m, m.addPath(path)
			}
			m.state = indexState
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m mainModel) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = indexState
			return m, nil
		case "up", "k":
			if m.browserCursor > 0 {
				m.browserCursor--
			}
		case "down", "j":
			if m.browserCursor < len(m.browserFiles)-1 {
				m.browserCursor++
			}
		case " ":
			path := filepath.Join(m.browserDir, m.browserFiles[m.browserCursor].name)
			if m.browserFiles[m.browserCursor].name == ".." {
				return m, nil
			}
			m.browserSelected[path] = !m.browserSelected[path]
		case "enter":
			item := m.browserFiles[m.browserCursor]
			if item.name == ".." {
				m.browserDir = filepath.Dir(m.browserDir)
				return m, m.loadDir(m.browserDir)
			}
			path := filepath.Join(m.browserDir, item.name)
			if item.isDir {
				m.browserDir = path
				return m, m.loadDir(m.browserDir)
			}
		case "a":
			return m, m.addSelectedPaths()
		}
	case dirLoadedMsg:
		m.browserFiles = msg.files
		m.browserCursor = 0
		return m, nil
	}
	return m, nil
}

type dirLoadedMsg struct {
	files []browserItem
}

func (m mainModel) loadDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error reading directory: %v", err)}
		}

		var items []browserItem
		// Add parent directory option
		items = append(items, browserItem{name: "..", isDir: true})

		// Sort entries: directories first, then files, both alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() && !entries[j].IsDir() {
				return true
			}
			if !entries[i].IsDir() && entries[j].IsDir() {
				return false
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})

		for _, entry := range entries {
			info, _ := entry.Info()
			items = append(items, browserItem{
				name:  entry.Name(),
				isDir: entry.IsDir(),
				size:  info.Size(),
			})
		}
		return dirLoadedMsg{items}
	}
}

func (m mainModel) addSelectedPaths() tea.Cmd {
	return func() tea.Msg {
		var paths []string
		for path, selected := range m.browserSelected {
			if selected {
				paths = append(paths, path)
			}
		}

		if len(paths) == 0 {
			// If nothing selected, maybe add current item if it's not ".."
			item := m.browserFiles[m.browserCursor]
			if item.name != ".." {
				paths = append(paths, filepath.Join(m.browserDir, item.name))
			}
		}

		if len(paths) == 0 {
			return messageMsg{title: "Add Path", content: "No paths selected to add."}
		}

		totalAdded := 0
		for _, path := range paths {
			added, err := m.app.AddPathToIndex(path, true, "")
			if err != nil {
				return messageMsg{title: "Error", content: fmt.Sprintf("Error adding path %s: %v", path, err)}
			}
			totalAdded += added
		}

		// Clear selection after adding
		m.browserSelected = make(map[string]bool)

		m.state = filesState
		return messageMsg{title: "Success", content: fmt.Sprintf("Added %d files from %d paths", totalAdded, len(paths))}
	}
}

func (m mainModel) updateRemovingPath(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = indexState
			return m, nil
		case "enter":
			path := m.textInput.Value()
			if path != "" {
				m.state = indexState
				return m, m.removePath(path)
			}
			m.state = indexState
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
		case "1":
			return m, m.toggleConfig("debug")
		case "2":
			return m, m.toggleConfig("dryrun")
		case "3":
			m.state = editingConfigState
			m.editingConfigID = "minfilesize"
			m.textInput.Focus()
			// Need current value... this is tricky without finding it in list
			// But we can just set it to empty or try to get it from app
			c := m.app.GetConfig()
			m.textInput.SetValue(fmt.Sprintf("%d", c.MinFileSize))
			return m, nil
		case "4":
			m.state = editingConfigState
			m.editingConfigID = "samplesize"
			m.textInput.Focus()
			c := m.app.GetConfig()
			m.textInput.SetValue(fmt.Sprintf("%d", c.SampleSizeBinaryCompare))
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
			configItem{key: "[1] Debug", value: fmt.Sprintf("%v", c.Debug), id: "debug"},
			configItem{key: "[2] DryRun", value: fmt.Sprintf("%v", c.DryRun), id: "dryrun"},
			configItem{key: "[3] MinFileSize", value: fmt.Sprintf("%d bytes", c.MinFileSize), id: "minfilesize"},
			configItem{key: "[4] SampleSizeBinaryCompare", value: fmt.Sprintf("%d bytes", c.SampleSizeBinaryCompare), id: "samplesize"},
			configItem{key: "Database", value: m.app.GetIndex().GetIndexPath(), id: "database"},
		}
		return configLoadedMsg{items}
	}
}

type configLoadedMsg struct{ items []list.Item }

func (m mainModel) removePath(path string) tea.Cmd {
	return func() tea.Msg {
		count, err := m.app.RemovePathFromIndex(path)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error removing path: %v", err)}
		}
		return messageMsg{title: "Success", content: fmt.Sprintf("Removed %d files from the database", count)}
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
	scanner := core.NewScanner(m.app.GetIndex())
	results, _ := scanner.ScanForDuplicates()
	return results
}

func (m mainModel) updateScanning(msg tea.Msg) (tea.Model, tea.Cmd) {
	// For now, scanning is synchronous in the background goroutine
	// but we don't have real progress updates from core yet.
	return m, nil
}

func (m mainModel) updateConfirmClear(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.state = menuState
			return m, m.clearIndex()
		case "n", "N", "esc", "backspace":
			m.state = menuState
			return m, nil
		}
	}
	return m, nil
}

func (m mainModel) updateSubView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == resultsState {
			return m.updateResults(msg)
		}
		if msg.String() == "esc" || msg.String() == "backspace" || msg.String() == "q" {
			if m.state == indexState {
				m.state = menuState
			} else if m.state == messageState {
				m.state = menuState // Default back to menu from message
			} else {
				m.state = menuState
			}
			return m, nil
		}
	}
	return m, nil
}

func (m mainModel) getFilteredGroups() []core.ResultList {
	var filtered []core.ResultList
	for _, group := range m.results {
		match := false
		for _, guid := range group.FileGuids {
			file := m.app.GetIndex().GetFileByGuid(guid)
			if file != nil {
				if m.dupeFilter == "" || strings.Contains(strings.ToLower(file.Path), strings.ToLower(m.dupeFilter)) {
					match = true
					break
				}
			}
		}
		if match {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func (m mainModel) flattenGroups(groups []core.ResultList) ([]resultItem, []*core.FileItem) {
	var allItems []resultItem
	var allFiles []*core.FileItem

	for i := range groups {
		group := &groups[i]
		groupMatch := false
		var groupItems []resultItem

		for _, guid := range group.FileGuids {
			file := m.app.GetIndex().GetFileByGuid(guid)
			if file != nil {
				if m.dupeFilter == "" || strings.Contains(strings.ToLower(file.Path), strings.ToLower(m.dupeFilter)) {
					groupMatch = true
					groupItems = append(groupItems, resultItem{
						isHeader: false,
						file:     file,
						group:    group,
					})
					allFiles = append(allFiles, file)
				}
			}
		}

		if groupMatch {
			allItems = append(allItems, resultItem{
				isHeader: true,
				hash:     group.HashSum,
				group:    group,
			})
			allItems = append(allItems, groupItems...)
		}
	}
	return allItems, allFiles
}

func (m mainModel) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filteredGroups := m.getFilteredGroups()

	// Update paginator
	m.paginator.PerPage = (m.height - 12) / 4
	if m.paginator.PerPage < 1 {
		m.paginator.PerPage = 1
	}
	m.paginator.TotalPages = (len(filteredGroups) + m.paginator.PerPage - 1) / m.paginator.PerPage

	// Current page items
	start, end := m.paginator.GetSliceBounds(len(filteredGroups))
	pageGroups := filteredGroups[start:end]
	pagedItems, _ := m.flattenGroups(pageGroups)

	// Total filtered items (for actions like trash/delete/export)
	_, allFiles := m.flattenGroups(filteredGroups)

	oldPage := m.paginator.Page

	switch msg.String() {
	case "esc", "q":
		m.state = menuState
		return m, nil
	case "up", "k":
		if m.dupeCursor > 0 {
			m.dupeCursor--
		} else if m.paginator.Page > 0 {
			m.paginator.Page--
			// Need to calculate dupeCursor for the new page (last item)
			newStart, newEnd := m.paginator.GetSliceBounds(len(filteredGroups))
			newPagedItems, _ := m.flattenGroups(filteredGroups[newStart:newEnd])
			m.dupeCursor = len(newPagedItems) - 1
		}
		return m, nil
	case "down", "j":
		if m.dupeCursor < len(pagedItems)-1 {
			m.dupeCursor++
		} else if m.paginator.Page < m.paginator.TotalPages-1 {
			m.paginator.Page++
			m.dupeCursor = 0
		}
		return m, nil
	case "f":
		m.state = filteringState
		m.textInput.Placeholder = "Enter filter text..."
		m.textInput.Focus()
		m.textInput.SetValue(m.dupeFilter)
		return m, nil
	case " ":
		if len(pagedItems) > 0 && m.dupeCursor < len(pagedItems) {
			item := pagedItems[m.dupeCursor]
			if item.isHeader {
				// Toggle whole group (except first one which is original)
				anyUnchecked := false
				for i, guid := range item.group.FileGuids {
					if i == 0 {
						continue
					}
					if !m.selectedDupes[guid] {
						anyUnchecked = true
						break
					}
				}

				for i, guid := range item.group.FileGuids {
					if i == 0 {
						continue
					}
					m.selectedDupes[guid] = anyUnchecked
				}
			} else {
				guid := item.file.Guid
				m.selectedDupes[guid] = !m.selectedDupes[guid]
			}
		}
		return m, nil
	case "a":
		// Auto select: keep first, select others
		for _, group := range m.results {
			for i, guid := range group.FileGuids {
				if i == 0 {
					m.selectedDupes[guid] = false
				} else {
					m.selectedDupes[guid] = true
				}
			}
		}
		return m, nil
	case "A":
		// Toggle auto switch: everything checked gets unchecked and vice versa
		for _, group := range m.results {
			for _, guid := range group.FileGuids {
				m.selectedDupes[guid] = !m.selectedDupes[guid]
			}
		}
		return m, nil
	case "x", "X":
		if len(pagedItems) > 0 && m.dupeCursor < len(pagedItems) {
			item := pagedItems[m.dupeCursor]
			if msg.String() == "X" {
				// Shift+X: Toggle only current group
				anyUnchecked := false
				for _, guid := range item.group.FileGuids {
					if !m.selectedDupes[guid] {
						anyUnchecked = true
						break
					}
				}
				for _, guid := range item.group.FileGuids {
					m.selectedDupes[guid] = anyUnchecked
				}
			} else {
				// x: Toggle all items
				anyUnchecked := false
				for _, f := range allFiles {
					if !m.selectedDupes[f.Guid] {
						anyUnchecked = true
						break
					}
				}
				for _, f := range allFiles {
					m.selectedDupes[f.Guid] = anyUnchecked
				}
			}
		}
		return m, nil
	case "t":
		return m, m.trashSelected(allFiles)
	case "d":
		return m, m.deleteSelected(allFiles)
	case "e":
		return m, m.exportSelected(allFiles)
	case "m":
		m.state = movingState
		m.textInput.Placeholder = "Enter destination path..."
		m.textInput.Focus()
		m.textInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.paginator, cmd = m.paginator.Update(msg)
	if m.paginator.Page != oldPage {
		m.dupeCursor = 0
	}
	return m, cmd
}

func (m mainModel) updateFiltering(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = resultsState
			return m, nil
		case "enter":
			m.dupeFilter = m.textInput.Value()
			m.state = resultsState
			m.paginator.Page = 0
			m.dupeCursor = 0
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m mainModel) updateMoving(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = resultsState
			return m, nil
		case "enter":
			dest := m.textInput.Value()
			if dest != "" {
				m.state = resultsState
				return m, m.moveSelected(dest)
			}
			m.state = resultsState
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m mainModel) moveSelected(dest string) tea.Cmd {
	return func() tea.Msg {
		// Get flattened files for navigation and selection
		var selected []*core.FileItem
		for _, group := range m.results {
			for _, guid := range group.FileGuids {
				file := m.app.GetIndex().GetFileByGuid(guid)
				if file != nil {
					if m.selectedDupes[file.Guid] {
						selected = append(selected, file)
					}
				}
			}
		}

		if len(selected) == 0 {
			return messageMsg{title: "Move", content: "No files selected for move."}
		}

		count, err := m.app.MoveFilesToDirectory(selected, dest)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error moving files: %v", err)}
		}

		// Clear selection after action
		for _, file := range selected {
			delete(m.selectedDupes, file.Guid)
		}

		// Update results without full re-scan
		var newResults []core.ResultList
		for _, group := range m.results {
			var newGuids []string
			for _, guid := range group.FileGuids {
				deleted := false
				for _, s := range selected {
					if s.Guid == guid {
						deleted = true
						break
					}
				}
				if !deleted {
					newGuids = append(newGuids, guid)
				}
			}
			if len(newGuids) >= 2 {
				newResults = append(newResults, core.ResultList{
					HashSum:   group.HashSum,
					FileGuids: newGuids,
				})
			}
		}

		return messageMsg{
			title:   "Success",
			content: fmt.Sprintf("Moved %d files to %s.", count, dest),
			results: newResults,
		}
	}
}

func (m mainModel) browsingView() string {
	s := greenTitleStyle.Render("Browse Path") + "\n\n"
	s += fmt.Sprintf("Current Directory: %s\n\n", m.browserDir)

	maxItems := m.height - 10
	if maxItems < 1 {
		maxItems = 1
	}

	start := 0
	if m.browserCursor >= maxItems {
		start = m.browserCursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(m.browserFiles) {
		end = len(m.browserFiles)
	}

	for i := start; i < end; i++ {
		item := m.browserFiles[i]
		cursor := " "
		if m.browserCursor == i {
			cursor = ">"
		}

		selected := "[ ]"
		path := filepath.Join(m.browserDir, item.name)
		if m.browserSelected[path] {
			selected = "[x]"
		}

		if item.name == ".." {
			selected = "   "
		}

		name := item.name
		if item.isDir {
			name += "/"
		}

		line := fmt.Sprintf("%s %s %s", cursor, selected, name)
		if m.browserCursor == i {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#00AF5F")).Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if len(m.browserFiles) > maxItems {
		s += fmt.Sprintf("\n ... (%d more) ...\n", len(m.browserFiles)-maxItems)
	}

	s += "\n (space) select • (enter) navigate • (a) add selected • (esc) back"
	return s
}

func (m mainModel) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.state {
	case menuState:
		content = m.list.View()
	case indexState:
		content = m.indexList.View()
	case scanningState:
		content = fmt.Sprintf("\n  %s\n\n  %s Scanning for duplicates...\n\n  Press q to cancel", pinkTitleStyle.Render(" Scanning "), m.spinner.View())
	case resultsState:
		content = m.resultsView()
	case filesState:
		content = m.filesList.View()
	case addingPathState:
		content = fmt.Sprintf("\n  %s\n\n%s\n\n(esc to cancel, enter to confirm)", greenTitleStyle.Render(" Add Path "), m.textInput.View())
	case removingPathState:
		content = fmt.Sprintf("\n  %s\n\n%s\n\n(esc to cancel, enter to confirm)", greenTitleStyle.Render(" Remove Path "), m.textInput.View())
	case configState:
		content = m.configList.View()
	case confirmClearState:
		content = fmt.Sprintf("\n  %s\n\n  Are you sure you want to clear the entire index?\n\n  (y)es / (n)o", greenTitleStyle.Render(" Clear Index "))
	case filteringState:
		content = fmt.Sprintf("\n  %s\n\n%s\n\n(esc to cancel, enter to confirm)", blueTitleStyle.Render(" Filter Duplicates "), m.textInput.View())
	case movingState:
		content = fmt.Sprintf("\n  %s\n\n%s\n\n(esc to cancel, enter to confirm)", blueTitleStyle.Render(" Move Selected Files "), m.textInput.View())
	case browsingState:
		content = m.browsingView()
	case editingConfigState:
		content = fmt.Sprintf("\n  %s\n\n%s\n\n(esc to cancel, enter to confirm)", orangeTitleStyle.Render(" Edit Config "), m.textInput.View())
	case messageState:
		content = fmt.Sprintf("\n  %s\n\n  %s\n\n  Press esc to go back", m.msgTitle, m.msgContent)
	}

	// Simple vertical join with some padding
	return content
}

func (m mainModel) startScan() tea.Cmd {
	return func() tea.Msg {
		scanner := core.NewScanner(m.app.GetIndex())
		results, err := scanner.ScanForDuplicates()
		return scanFinishedMsg{results: results, err: err}
	}
}

func (m mainModel) resultsView() string {
	if m.err != nil {
		return fmt.Sprintf("\n\n  Error: %v\n\n  Press esc to go back", m.err)
	}

	if len(m.results) == 0 {
		return "\n\n  No duplicates found.\n\n  Press esc to go back"
	}

	var sb strings.Builder
	sb.WriteString("\n  " + blueTitleStyle.Render(" Results (Interactive) ") + "\n\n")

	// Filtering
	if m.dupeFilter != "" {
		sb.WriteString(fmt.Sprintf("  Filter: %s\n", m.dupeFilter))
	}

	filteredGroups := m.getFilteredGroups()
	m.paginator.PerPage = (m.height - 12) / 4
	if m.paginator.PerPage < 1 {
		m.paginator.PerPage = 1
	}
	m.paginator.TotalPages = (len(filteredGroups) + m.paginator.PerPage - 1) / m.paginator.PerPage

	if m.paginator.Page >= m.paginator.TotalPages && m.paginator.TotalPages > 0 {
		m.paginator.Page = m.paginator.TotalPages - 1
	}

	// Current page items
	start, end := m.paginator.GetSliceBounds(len(filteredGroups))
	pageGroups := filteredGroups[start:end]
	pagedItems, _ := m.flattenGroups(pageGroups)

	// Total filtered items
	allItems, _ := m.flattenGroups(filteredGroups)

	if len(allItems) == 0 {
		sb.WriteString("  No files match filter.\n")
	} else {
		// Calculate starting group counter for the current page
		groupCounter := m.paginator.Page * m.paginator.PerPage

		var lastGroup *core.ResultList

		for i, item := range pagedItems {
			if item.group != lastGroup {
				lastGroup = item.group
				groupCounter++
			}

			cursor := " "
			if i == m.dupeCursor {
				cursor = ">"
			}

			colorIdx := (groupCounter - 1) % len(hashColors)
			if colorIdx < 0 {
				colorIdx = 0
			}
			style := lipgloss.NewStyle().Foreground(hashColors[colorIdx])

			if item.isHeader {
				// Get size from first file in group
				sizeStr := ""
				if len(item.group.FileGuids) > 0 {
					firstFile := m.app.GetIndex().GetFileByGuid(item.group.FileGuids[0])
					if firstFile != nil {
						sizeStr = fmt.Sprintf(" (%s)", firstFile.HumanizedSize)
					}
				}
				sb.WriteString(fmt.Sprintf(" %s %s%s\n", cursor, style.Render(fmt.Sprintf("Hash: %s", item.hash)), sizeStr))
			} else {
				checked := " "
				if m.selectedDupes[item.file.Guid] {
					checked = "x"
				}

				fileStyle := lipgloss.NewStyle()
				// Check if it's the "original" (first in group)
				isOriginal := item.file.Guid == item.group.FileGuids[0]
				if isOriginal {
					fileStyle = fileStyle.Bold(true).Foreground(lipgloss.Color("2")) // Green for original
				} else {
					fileStyle = fileStyle.Foreground(lipgloss.Color("7")) // Gray/White for dupes
				}

				sb.WriteString(fmt.Sprintf(" %s   [%s] %s\n", cursor, checked, fileStyle.Render(item.file.Path)))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n  Page %d of %d (%d items total)\n", m.paginator.Page+1, m.paginator.TotalPages, len(allItems)))
	sb.WriteString("\n  [↑/↓] navigate • [←/→] page • [space] select • [x/X] toggle all/group • [a/A] auto/inv • [f] filter • [e] export • [m] move • [t] trash • [d] delete • [esc] back\n")

	return sb.String()
}

func (m mainModel) exportSelected(allFiles []*core.FileItem) tea.Cmd {
	return func() tea.Msg {
		var selected []*core.FileItem
		for _, file := range allFiles {
			if m.selectedDupes[file.Guid] {
				selected = append(selected, file)
			}
		}
		if len(selected) == 0 {
			return messageMsg{title: "Export", content: "No files selected for export."}
		}

		// Re-use Export functionality but filtered
		// Actually, App.Export works on ALL dupes.
		// Let's just create a quick report for the selected ones.
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("dupefiles_selection_%s.txt", timestamp)

		f, err := os.Create(filename)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Failed to create export file: %v", err)}
		}
		defer f.Close()

		fmt.Fprintf(f, "# DupeFiles Selection Export - %d files\n", len(selected))
		for _, file := range selected {
			fmt.Fprintf(f, "%s (%s)\n", file.Path, file.HumanizedSize)
		}

		return messageMsg{title: "Success", content: fmt.Sprintf("Exported selection to %s", filename)}
	}
}

func (m mainModel) trashSelected(allFiles []*core.FileItem) tea.Cmd {
	return func() tea.Msg {
		var selected []*core.FileItem
		for _, file := range allFiles {
			if m.selectedDupes[file.Guid] {
				selected = append(selected, file)
			}
		}
		if len(selected) == 0 {
			return messageMsg{title: "Trash", content: "No files selected for trash."}
		}

		count, err := m.app.MoveFilesToTrash(selected)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error moving to trash: %v", err)}
		}

		// Clear selection after action
		for _, file := range selected {
			delete(m.selectedDupes, file.Guid)
		}

		// Update results without full re-scan
		var newResults []core.ResultList
		for _, group := range m.results {
			var newGuids []string
			for _, guid := range group.FileGuids {
				deleted := false
				for _, s := range selected {
					if s.Guid == guid {
						deleted = true
						break
					}
				}
				if !deleted {
					newGuids = append(newGuids, guid)
				}
			}
			if len(newGuids) >= 2 {
				newResults = append(newResults, core.ResultList{
					HashSum:   group.HashSum,
					FileGuids: newGuids,
				})
			}
		}

		return messageMsg{
			title:   "Success",
			content: fmt.Sprintf("Moved %d files to trash.", count),
			results: newResults,
		}
	}
}

func (m mainModel) deleteSelected(allFiles []*core.FileItem) tea.Cmd {
	return func() tea.Msg {
		var selected []*core.FileItem
		for _, file := range allFiles {
			if m.selectedDupes[file.Guid] {
				selected = append(selected, file)
			}
		}
		if len(selected) == 0 {
			return messageMsg{title: "Delete", content: "No files selected for deletion."}
		}

		count, err := m.app.DeleteFiles(selected)
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error deleting files: %v", err)}
		}

		// Clear selection after action
		for _, file := range selected {
			delete(m.selectedDupes, file.Guid)
		}

		// Update results without full re-scan
		var newResults []core.ResultList
		for _, group := range m.results {
			var newGuids []string
			for _, guid := range group.FileGuids {
				deleted := false
				for _, s := range selected {
					if s.Guid == guid {
						deleted = true
						break
					}
				}
				if !deleted {
					newGuids = append(newGuids, guid)
				}
			}
			if len(newGuids) >= 2 {
				newResults = append(newResults, core.ResultList{
					HashSum:   group.HashSum,
					FileGuids: newGuids,
				})
			}
		}

		return messageMsg{
			title:   "Success",
			content: fmt.Sprintf("Deleted %d files.", count),
			results: newResults,
		}
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
		err := m.app.GetIndex().Clear()
		if err != nil {
			return messageMsg{title: "Error", content: fmt.Sprintf("Error clearing index: %v", err)}
		}
		return messageMsg{title: "Success", content: "Database cleared"}
	}
}
