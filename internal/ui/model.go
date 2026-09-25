package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"martis/internal/curlparser"
	"martis/internal/domain"
	"martis/internal/environment"
	"martis/internal/httpclient"
	"martis/internal/repository"
	"martis/internal/requestutil"
	"martis/internal/ui/styles"
)

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusMethod
	FocusURL
	FocusTabs
	FocusConfig
	FocusSend
	FocusResponse
)

const totalFocusAreas = 7

type ConfigTab int

const (
	TabHeaders ConfigTab = iota
	TabBodyRaw
	TabBodyForm
	TabQuery
	TabAuth
	TabAssertions
)

const totalTabs = 6

func configTabAtX(x int) ConfigTab {
	labels := []string{"Hdr", "JSON", "Form", "Query", "Auth", "Tests"}
	for i, label := range labels {
		width := lipgloss.Width(styles.InactiveTab.Render(label))
		if x < width {
			return ConfigTab(i)
		}
		x -= width
	}
	return TabAssertions
}

type modalKind int

const (
	modalNone modalKind = iota
	modalSave
	modalCurlImport
	modalSearch
	modalHistory
	modalEnvironment
	modalTheme
	modalBenchmark
)

type treeRowType int

const (
	rowFolder treeRowType = iota
	rowItem
)

type treeRow struct {
	rowType     treeRowType
	folderIndex int
	itemIndex   int
	name        string
	method      string
	isExpanded  bool
}

type Model struct {
	width  int
	height int

	focus FocusArea
	tab   ConfigTab

	repo       repository.CollectionRepository
	httpClient httpclient.Client

	collection        *domain.Collection
	selectedTreeIndex int
	sidebarRows       []treeRow
	saveModalOpen     bool
	saveNameInput     textinput.Model
	modal             modalKind
	modalInput        textarea.Model
	modalError        string
	modalIndex        int
	history           []domain.HistoryEntry
	envFiles          []string
	activeEnv         map[string]string
	activeEnvName     string
	status            string
	responseBody      string
	responseHeaders   http.Header
	lastStatus        int
	themeIndex        int
	responseTab       int
	responseSearch    string
	lastPayload       domain.RequestPayload

	methods      []string
	methodIndex  int
	urlInput     textinput.Model
	headerKey    textinput.Model
	headerVal    textinput.Model
	headerAuth   textinput.Model
	jsonBody     textarea.Model
	formKey      textinput.Model
	formFilePath textinput.Model
	configEditor textarea.Model
	extraHeaders []domain.KeyValue
	queryRows    []domain.KeyValue
	auth         domain.AuthConfig
	assertions   string

	headersFocusIndex int
	formFocusIndex    int

	loading  bool
	spinner  spinner.Model
	lastResp *domain.ResponseResult

	viewport      viewport.Model
	viewportReady bool
}

func NewModel(repo repository.CollectionRepository, client httpclient.Client) Model {
	urlIn := textinput.New()
	urlIn.Placeholder = "https://httpbin.org/anything"
	urlIn.SetValue("https://httpbin.org/anything")
	urlIn.CharLimit = 500
	urlIn.Width = 35

	hKey := textinput.New()
	hKey.Placeholder = "Header Name (e.g. Content-Type)"
	hKey.SetValue("Content-Type")
	hKey.CharLimit = 100

	hVal := textinput.New()
	hVal.Placeholder = "Header Value (e.g. application/json)"
	hVal.SetValue("application/json")
	hVal.CharLimit = 200

	hAuth := textinput.New()
	hAuth.Placeholder = "Bearer token or credentials"
	hAuth.CharLimit = 500

	ta := textarea.New()
	ta.Placeholder = "{\n  \"hello\": \"world\"\n}"
	ta.SetValue("{\n  \"message\": \"Hello from Martis!\",\n  \"status\": \"fast\"\n}")
	ta.ShowLineNumbers = true
	ta.SetHeight(8)

	fKey := textinput.New()
	fKey.Placeholder = "Field Name (e.g. file / upload)"
	fKey.SetValue("file")

	fPath := textinput.New()
	fPath.Placeholder = "File Path (e.g. ./test.txt)"

	sName := textinput.New()
	sName.Placeholder = "Request Name (e.g. Get User Profile)"
	sName.CharLimit = 100

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(styles.PrimaryColor)
	editor := textarea.New()
	editor.SetHeight(8)
	editor.ShowLineNumbers = true
	modalEditor := textarea.New()
	modalEditor.SetHeight(12)
	modalEditor.SetWidth(64)

	col, _ := repo.Load()

	m := Model{
		focus:         FocusSidebar,
		tab:           TabBodyRaw,
		repo:          repo,
		httpClient:    client,
		collection:    col,
		methods:       domain.SupportedMethods(),
		methodIndex:   1, // POST
		urlInput:      urlIn,
		headerKey:     hKey,
		headerVal:     hVal,
		headerAuth:    hAuth,
		jsonBody:      ta,
		formKey:       fKey,
		formFilePath:  fPath,
		saveNameInput: sName,
		spinner:       sp,
		configEditor:  editor,
		modalInput:    modalEditor,
		history:       nil,
		activeEnv:     map[string]string{},
	}
	m.history, _ = repository.LoadHistory(filepath.Join(repository.ConfigDir(), "history.json"))
	m.envFiles, _ = environment.Files(filepath.Join(repository.ConfigDir(), "environments"))
	if len(m.envFiles)==0 {m.envFiles,_=environment.Files(filepath.Join(repository.LegacyConfigDir(),"environments"))}
	if data,err:=os.ReadFile(filepath.Join(repository.ConfigDir(),"preferences.json"));err==nil {var prefs struct{Theme int `json:"theme"`};if json.Unmarshal(data,&prefs)==nil&&prefs.Theme>=0&&prefs.Theme<5 {m.themeIndex=prefs.Theme;m.applyTheme()}}

	m.rebuildSidebarRows()
	m.updateFocusStates()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *Model) rebuildSidebarRows() {
	var rows []treeRow
	if m.collection == nil {
		return
	}
	for fIdx, f := range m.collection.Folders {
		rows = append(rows, treeRow{
			rowType:     rowFolder,
			folderIndex: fIdx,
			name:        f.Name,
			isExpanded:  f.IsExpanded,
		})

		if f.IsExpanded {
			for iIdx, it := range f.Items {
				rows = append(rows, treeRow{
					rowType:     rowItem,
					folderIndex: fIdx,
					itemIndex:   iIdx,
					name:        it.Name,
					method:      it.Method,
				})
			}
		}
	}
	m.sidebarRows = rows
	if m.selectedTreeIndex >= len(m.sidebarRows) && len(m.sidebarRows) > 0 {
		m.selectedTreeIndex = len(m.sidebarRows) - 1
	}
}

func (m *Model) updateFocusStates() {
	m.urlInput.Blur()
	m.headerKey.Blur()
	m.headerVal.Blur()
	m.headerAuth.Blur()
	m.jsonBody.Blur()
	m.formKey.Blur()
	m.formFilePath.Blur()
	m.saveNameInput.Blur()
	m.configEditor.Blur()
	m.modalInput.Blur()

	if m.modal == modalSave {
		m.saveNameInput.Focus()
		return
	}
	if m.modal != modalNone {
		m.modalInput.Focus()
		return
	}

	switch m.focus {
	case FocusURL:
		m.urlInput.Focus()
	case FocusConfig:
		switch m.tab {
		case TabHeaders:
			switch m.headersFocusIndex {
			case 0:
				m.headerKey.Focus()
			case 1:
				m.headerVal.Focus()
			case 2:
				m.headerAuth.Focus()
			}
		case TabBodyRaw:
			m.jsonBody.Focus()
		case TabBodyForm:
			switch m.formFocusIndex {
			case 0:
				m.formKey.Focus()
			case 1:
				m.formFilePath.Focus()
			}
		}
		if m.tab == TabQuery || m.tab == TabAuth || m.tab == TabAssertions || m.tab == TabHeaders {
			m.configEditor.Focus()
		}
	}
}

func (m *Model) loadCollectionItem(item domain.CollectionItem) {
	for idx, meth := range m.methods {
		if strings.EqualFold(meth, item.Method) {
			m.methodIndex = idx
			break
		}
	}
	m.urlInput.SetValue(item.URL)
	m.headerKey.SetValue(item.HeaderKey)
	m.headerVal.SetValue(item.HeaderVal)
	m.headerAuth.SetValue(item.HeaderAuth)
	m.extraHeaders = item.Headers
	m.auth = item.Auth
	m.assertions = item.Assertions
	m.syncEditorFromTab()

	if item.BodyType == "form" {
		m.tab = TabBodyForm
		m.formKey.SetValue(item.FormKey)
		m.formFilePath.SetValue(item.FormPath)
	} else {
		m.tab = TabBodyRaw
		m.jsonBody.SetValue(item.BodyRaw)
	}
	if rows, err := requestutil.Query(item.URL); err == nil {
		m.queryRows = rows
	}
	m.syncEditorFromTab()
	m.updateFocusStates()
}

func (m *Model) syncEditorFromTab() {
	var text string
	switch m.tab {
	case TabHeaders:
		for _, h := range m.extraHeaders {
			text += h.Key + ": " + h.Value + "\n"
		}
	case TabQuery:
		for _, q := range m.queryRows {
			text += q.Key + "=" + q.Value + "\n"
		}
	case TabAuth:
		data, _ := json.MarshalIndent(m.auth, "", "  ")
		text = string(data)
	case TabAssertions:
		text = m.assertions
	}
	m.configEditor.SetValue(strings.TrimSuffix(text, "\n"))
}

func (m *Model) syncTabFromEditor() error {
	text := m.configEditor.Value()
	switch m.tab {
	case TabHeaders:
		m.extraHeaders = nil
		for i, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			k, v, ok := strings.Cut(line, ":")
			if !ok || strings.TrimSpace(k) == "" {
				return fmt.Errorf("header line %d must be Key: Value", i+1)
			}
			m.extraHeaders = append(m.extraHeaders, domain.KeyValue{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
		}
	case TabQuery:
		m.queryRows = nil
		for i, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(k) == "" {
				return fmt.Errorf("query line %d must be key=value", i+1)
			}
			m.queryRows = append(m.queryRows, domain.KeyValue{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
		}
	case TabAuth:
		var auth domain.AuthConfig
		if text != "" {
			if err := json.Unmarshal([]byte(text), &auth); err != nil {
				return fmt.Errorf("auth must be valid JSON: %w", err)
			}
		}
		m.auth = auth
	case TabAssertions:
		m.assertions = text
	}
	return nil
}

func (m *Model) toggleFolder(fIdx int) {
	if m.collection != nil && fIdx >= 0 && fIdx < len(m.collection.Folders) {
		m.collection.Folders[fIdx].IsExpanded = !m.collection.Folders[fIdx].IsExpanded
		if err := m.repo.Save(m.collection); err != nil {
			m.status = "save collection failed: " + err.Error()
		}
		m.rebuildSidebarRows()
	}
}

func (m *Model) saveCurrentToCollection(name string) {
	if m.collection == nil {
		m.collection = &domain.Collection{Name: "Default Workspace", Folders: []domain.Folder{{ID: "folder-default", Name: "My Requests", IsExpanded: true}}}
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value()))
	}

	bodyType := "raw"
	if m.tab == TabBodyForm {
		bodyType = "form"
	}

	newItem := domain.CollectionItem{
		Headers:    append([]domain.KeyValue(nil), m.extraHeaders...),
		Auth:       m.auth,
		Assertions: m.assertions,
		ID:         fmt.Sprintf("item-%d", time.Now().UnixNano()),
		Name:       name,
		Method:     m.methods[m.methodIndex],
		URL:        m.urlInput.Value(),
		HeaderKey:  m.headerKey.Value(),
		HeaderVal:  m.headerVal.Value(),
		HeaderAuth: m.headerAuth.Value(),
		BodyType:   bodyType,
		BodyRaw:    m.jsonBody.Value(),
		FormKey:    m.formKey.Value(),
		FormPath:   m.formFilePath.Value(),
	}

	if len(m.collection.Folders) == 0 {
		m.collection.Folders = append(m.collection.Folders, domain.Folder{
			ID:         "folder-default",
			Name:       "My Requests",
			IsExpanded: true,
		})
	}

	targetFolder := 0
	if m.selectedTreeIndex < len(m.sidebarRows) {
		targetFolder = m.sidebarRows[m.selectedTreeIndex].folderIndex
	}
	m.collection.Folders[targetFolder].Items = append(m.collection.Folders[targetFolder].Items, newItem)
	m.collection.Folders[targetFolder].IsExpanded = true
	if err := m.repo.Save(m.collection); err != nil {
		m.status = "save collection failed: " + err.Error()
	}
	m.rebuildSidebarRows()
}

func (m *Model) executeRequestCmd() tea.Cmd {
	bodyType := "raw"
	if m.tab == TabBodyForm {
		bodyType = "form"
	}

	payload := domain.RequestPayload{
		Method:     m.methods[m.methodIndex],
		URL:        m.urlInput.Value(),
		HeaderKey:  m.headerKey.Value(),
		HeaderVal:  m.headerVal.Value(),
		HeaderAuth: m.headerAuth.Value(),
		BodyType:   bodyType,
		BodyRaw:    m.jsonBody.Value(),
		FormKey:    m.formKey.Value(),
		FormPath:   m.formFilePath.Value(),
	}

	if m.tab == TabHeaders || m.tab == TabQuery || m.tab == TabAuth || m.tab == TabAssertions {
		if err := m.syncTabFromEditor(); err != nil {
			return func() tea.Msg { return domain.ResponseResult{Err: err} }
		}
	}
	payload.Headers = append([]domain.KeyValue(nil), m.extraHeaders...)
	payload.Auth = m.auth
	payload.Assertions = m.assertions
	if m.tab == TabQuery {
		if url, err := requestutil.WithQuery(payload.URL, m.queryRows); err == nil {
			payload.URL = url
		}
	}
	prepared, prepErr := requestutil.Prepare(payload, m.activeEnv)
	if prepErr != nil {
		return func() tea.Msg { return domain.ResponseResult{Err: prepErr} }
	}
	m.lastPayload = payload
	return func() tea.Msg {
		return m.httpClient.Do(prepared)
	}
}

func (m *Model) currentPayload() domain.RequestPayload {
	p := domain.RequestPayload{Method: m.methods[m.methodIndex], URL: m.urlInput.Value(), HeaderKey: m.headerKey.Value(), HeaderVal: m.headerVal.Value(), HeaderAuth: m.headerAuth.Value(), BodyType: "raw", BodyRaw: m.jsonBody.Value(), FormKey: m.formKey.Value(), FormPath: m.formFilePath.Value(), Headers: append([]domain.KeyValue(nil), m.extraHeaders...), Auth: m.auth, Assertions: m.assertions}
	if m.tab == TabBodyForm {
		p.BodyType = "form"
	}
	if m.tab == TabQuery {
		_ = m.syncTabFromEditor()
		if u, err := requestutil.WithQuery(p.URL, m.queryRows); err == nil {
			p.URL = u
		}
	}
	return p
}

func (m *Model) payloadToItem(p domain.RequestPayload) domain.CollectionItem {
	return domain.CollectionItem{ID: fmt.Sprintf("history-%d", time.Now().UnixNano()), Name: p.Method + " " + filepath.Base(p.URL), Method: p.Method, URL: p.URL, HeaderKey: p.HeaderKey, HeaderVal: p.HeaderVal, HeaderAuth: p.HeaderAuth, BodyType: p.BodyType, BodyRaw: p.BodyRaw, FormKey: p.FormKey, FormPath: p.FormPath, Headers: append([]domain.KeyValue(nil), p.Headers...), Auth: p.Auth, Assertions: p.Assertions}
}

func formatResponseHeaders(headers http.Header) string {
	var lines []string
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, key+": "+strings.Join(headers.Values(key), ", "))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) openModal(kind modalKind, initial string) {
	m.modal = kind
	m.modalError = ""
	m.modalIndex = 0
	m.modalInput.SetValue(initial)
	if kind == modalTheme {
		m.modalIndex = m.themeIndex
	}
	m.updateFocusStates()
}

func (m *Model) handleModalKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		m.modal = modalNone
		m.updateFocusStates()
		return nil
	}
	if m.modal == modalHistory || m.modal == modalEnvironment || m.modal == modalTheme {
		count := 0
		switch m.modal {
		case modalHistory:
			count = len(m.history)
		case modalEnvironment:
			count = len(m.envFiles)
		case modalTheme:
			count = 5
		}
		if msg.String() == "up" && count > 0 {
			m.modalIndex = (m.modalIndex - 1 + count) % count
			return nil
		}
		if msg.String() == "down" && count > 0 {
			m.modalIndex = (m.modalIndex + 1) % count
			return nil
		}
		if msg.String() != "enter" {
			return nil
		}
		switch m.modal {
		case modalHistory:
			if count > 0 {
				m.loadCollectionItem(m.history[m.modalIndex].Request)
				m.focus = FocusURL
				m.status = "Loaded history request"
			}
		case modalEnvironment:
			if count > 0 {
				path := m.envFiles[m.modalIndex]
				values, err := environment.Load(path)
				if err != nil {
					m.modalError = err.Error()
					return nil
				}
				m.activeEnv = values
				m.activeEnvName = filepath.Base(path)
				m.status = "Environment: " + m.activeEnvName
			}
		case modalTheme:
			m.themeIndex = m.modalIndex
			m.applyTheme()
			m.status = "Theme: " + []string{"Dracula", "Catppuccin", "Nord", "Tokyo Night", "Monokai"}[m.themeIndex]
		}
		m.modal = modalNone
		m.updateFocusStates()
		return nil
	}
	if msg.String() == "ctrl+enter" || (msg.String() == "enter" && m.modal == modalSave) {
		text := strings.TrimSpace(m.modalInput.Value())
		switch m.modal {
		case modalSave:
			m.saveCurrentToCollection(text)
			m.status = "Saved request to collection"
		case modalCurlImport:
			if err := m.importCurl(text); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.status = "Imported cURL request"
		case modalSearch:
			m.responseSearch = text
			m.status = "Search: " + text
			m.refreshResponse()
			m.refreshResponse()
		case modalBenchmark:
			count, err := strconv.Atoi(text)
			if err != nil || count < 1 || count > 1000 {
				m.modalError = "Enter a request count from 1 to 1000"
				return nil
			}
			p := m.currentPayload()
			prepared, err := requestutil.Prepare(p, m.activeEnv)
			if err != nil {
				m.modalError = err.Error()
				return nil
			}
			client := m.httpClient
			m.modal = modalNone
			m.loading = true
			m.updateFocusStates()
			return func() tea.Msg {
				r, err := requestutil.Benchmark(client.Do, prepared, count)
				return benchmarkMsg{result: r, err: err}
			}
		}
		m.modal = modalNone
		m.updateFocusStates()
		return nil
	}
	var cmd tea.Cmd
	m.modalInput, cmd = m.modalInput.Update(msg)
	return cmd
}

type benchmarkMsg struct {
	result requestutil.BenchmarkResult
	err    error
}

func (m *Model) applyTheme() {
	colors := []string{"#BD93F9", "#CBA6F7", "#88C0D0", "#7AA2F7", "#F92672"}
	color := lipgloss.Color(colors[m.themeIndex%len(colors)])
	styles.ActiveBorder = color
	styles.ActivePanel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color)
	styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(color).Padding(0, 1)
	styles.ActiveTab = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(color).Padding(0, 1)
}

func (m *Model) importCurl(raw string) error {
	parsed, err := curlparser.Parse(raw)
	if err != nil {
		return err
	}
	m.urlInput.SetValue(parsed.URL)
	for i, method := range m.methods {
		if method == parsed.Method {
			m.methodIndex = i
		}
	}
	m.extraHeaders = nil
	for key, value := range parsed.Headers {
		m.extraHeaders = append(m.extraHeaders, domain.KeyValue{Key: key, Value: value})
	}
	sort.Slice(m.extraHeaders, func(i, j int) bool { return m.extraHeaders[i].Key < m.extraHeaders[j].Key })
	m.headerAuth.SetValue(parsed.AuthHeader)
	if parsed.Body != "" {
		m.tab = TabBodyRaw
		m.jsonBody.SetValue(parsed.Body)
	}
	if parsed.FormPath != "" {
		m.tab = TabBodyForm
		m.formKey.SetValue(parsed.FormKey)
		m.formFilePath.SetValue(parsed.FormPath)
	}
	m.syncEditorFromTab()
	m.updateFocusStates()
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		sidebarWidth := 28
		remWidth := m.width - sidebarWidth - 6
		halfWidth := remWidth / 2
		if halfWidth < 30 {
			halfWidth = 30
		}
		vpHeight := m.height - 10
		if vpHeight < 5 {
			vpHeight = 5
		}

		if !m.viewportReady {
			m.viewport = viewport.New(halfWidth, vpHeight)
			m.viewport.SetContent("Waiting for request... Click an item or press [Ctrl+S] to send.")
			m.viewportReady = true
		} else {
			m.viewport.Width = halfWidth
			m.viewport.Height = vpHeight
		}

		m.urlInput.Width = halfWidth - 14
		m.headerKey.Width = (halfWidth / 2) - 3
		m.headerVal.Width = (halfWidth / 2) - 3
		m.headerAuth.Width = halfWidth - 6
		m.jsonBody.SetWidth(halfWidth - 4)
		m.jsonBody.SetHeight(vpHeight - 8)
		m.formKey.Width = halfWidth - 6
		m.formFilePath.Width = halfWidth - 6
		m.saveNameInput.Width = 35

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			sidebarWidth := 28
			if msg.X <= sidebarWidth+2 && msg.Y >= 2 {
				// View has a title, panel border, tree heading and help line above
				// the first row. Keep this aligned with renderSidebar's layout.
				clickedRow := msg.Y - 4
				if clickedRow >= 0 && clickedRow < len(m.sidebarRows) {
					m.selectedTreeIndex = clickedRow
					m.focus = FocusSidebar
					row := m.sidebarRows[clickedRow]

					if row.rowType == rowFolder {
						m.toggleFolder(row.folderIndex)
					} else {
						f := m.collection.Folders[row.folderIndex]
						if row.itemIndex < len(f.Items) {
							m.loadCollectionItem(f.Items[row.itemIndex])
						}
					}
					m.updateFocusStates()
					return m, nil
				}
			}
			requestWidth := (m.width - 34) / 2
			if msg.Y == 4 && requestWidth > 0 && msg.X >= 30 && msg.X < 30+requestWidth {
				m.tab = configTabAtX(msg.X - 30)
				if m.tab == TabQuery {
					if rows, err := requestutil.Query(m.urlInput.Value()); err == nil {
						m.queryRows = rows
					}
				}
				m.syncEditorFromTab()
				m.focus = FocusTabs
				m.updateFocusStates()
				return m, nil
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var spCmd tea.Cmd
			m.spinner, spCmd = m.spinner.Update(msg)
			cmds = append(cmds, spCmd)
		}

	case domain.ResponseResult:
		m.loading = false
		m.lastResp = &msg
		m.lastStatus = msg.StatusCode
		if msg.Headers != nil {
			m.responseHeaders = msg.Headers.Clone()
		}
		m.responseBody = msg.Body
		if msg.Err != nil {
			m.viewport.SetContent(fmt.Sprintf("❌ Request Error:\n\n%v\n\nDuration: %v", msg.Err, msg.Duration))
			m.status = msg.Err.Error()
		} else {
			if failures := requestutil.Assert(msg, m.lastPayload.Assertions); len(failures) > 0 {
				m.status = "Assertion failed: " + strings.Join(failures, ", ")
			} else {
				m.status = fmt.Sprintf("HTTP %d", msg.StatusCode)
			}
			item := m.payloadToItem(m.lastPayload)
			if err := repository.AppendHistory(filepath.Join(repository.ConfigDir(), "history.json"), domain.HistoryEntry{At: time.Now(), Request: item, Status: msg.StatusCode}); err != nil {
				m.status = "history save failed: " + err.Error()
			} else {
				m.history, _ = repository.LoadHistory(filepath.Join(repository.ConfigDir(), "history.json"))
			}
			headerLines := make([]string, 0, len(msg.Headers))
			for k, v := range msg.Headers {
				headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, strings.Join(v, ", ")))
			}
			content := fmt.Sprintf("// Response Headers\n%s\n\n// Response Body (%d bytes)\n%s",
				strings.Join(headerLines, "\n"),
				len(msg.Body),
				msg.Body,
			)
			m.viewport.SetContent(content)
		}
		m.viewport.GotoTop()
		m.refreshResponse()
	case benchmarkMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "Benchmark: " + msg.err.Error()
		} else {
			m.status = fmt.Sprintf("Benchmark %d requests: avg %s, min %s, max %s, failed %d", msg.result.Count, msg.result.Average.Round(time.Millisecond), msg.result.Min.Round(time.Millisecond), msg.result.Max.Round(time.Millisecond), msg.result.Failed)
		}

	case tea.KeyMsg:
		if m.modal != modalNone {
			return m, m.handleModalKey(msg)
		}
		if m.saveModalOpen {
			switch msg.String() {
			case "esc":
				m.saveModalOpen = false
				m.updateFocusStates()
				return m, nil
			case "enter":
				m.saveCurrentToCollection(m.saveNameInput.Value())
				m.saveModalOpen = false
				m.saveNameInput.SetValue("")
				m.updateFocusStates()
				return m, nil
			}
			var sCmd tea.Cmd
			m.saveNameInput, sCmd = m.saveNameInput.Update(msg)
			return m, sCmd
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+s":
			if !m.loading {
				m.loading = true
				m.viewport.SetContent("Sending HTTP request...")
				cmds = append(cmds, m.spinner.Tick, m.executeRequestCmd())
			}
			return m, tea.Batch(cmds...)

		case "ctrl+e":
			m.openModal(modalSave, fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value())))
			return m, nil
		case "ctrl+h":
			m.openModal(modalHistory, "")
			return m, nil
		case "ctrl+g":
			m.openModal(modalEnvironment, "")
			return m, nil
		case "ctrl+i":
			if copied, err := clipboard.ReadAll(); err == nil && strings.HasPrefix(strings.TrimSpace(copied), "curl") {
				if err := m.importCurl(copied); err != nil {
					m.openModal(modalCurlImport, copied)
					m.modalError = err.Error()
				} else {
					m.status = "Imported cURL from clipboard"
				}
				return m, nil
			}
			m.openModal(modalCurlImport, "")
			return m, nil
		case "ctrl+x":
			text := curlparser.Export(m.currentPayload())
			if err := clipboard.WriteAll(text); err != nil {
				m.status = "cURL: " + text
			} else {
				m.status = "cURL copied to clipboard"
			}
			return m, nil
		case "ctrl+y":
			text := m.responseBody
			if m.responseTab == 1 {
				text = formatResponseHeaders(m.responseHeaders)
			}
			if err := clipboard.WriteAll(text); err != nil {
				m.status = "clipboard: " + err.Error()
			} else {
				m.status = "Response copied"
			}
			return m, nil
		case "ctrl+o":
			ext := ".txt"
			if strings.Contains(strings.ToLower(m.responseHeaders.Get("Content-Type")), "json") {
				ext = ".json"
			}
			name := "response-" + time.Now().Format("20060102-150405") + ext
			if err := os.WriteFile(name, []byte(m.responseBody), 0600); err != nil {
				m.status = "save response: " + err.Error()
			} else {
				m.status = "Saved " + name
			}
			return m, nil
		case "ctrl+b":
			m.openModal(modalBenchmark, "10")
			return m, nil
		case "f2":
			m.openModal(modalTheme, "")
			return m, nil
		case "/":
			if m.focus == FocusResponse {
				m.openModal(modalSearch, "")
				return m, nil
			}

		case "tab":
			m.focus = (m.focus + 1) % totalFocusAreas
			m.updateFocusStates()
			return m, nil

		case "shift+tab":
			m.focus = (m.focus - 1 + totalFocusAreas) % totalFocusAreas
			m.updateFocusStates()
			return m, nil

		case "ctrl+t":
			old := m.tab
			m.tab = (m.tab + 1) % totalTabs
			if old == TabHeaders {
				_ = m.syncTabFromEditor()
			}
			if m.tab == TabQuery {
				if rows, err := requestutil.Query(m.urlInput.Value()); err == nil {
					m.queryRows = rows
				}
			}
			m.syncEditorFromTab()
			m.updateFocusStates()
			return m, nil
		}
		if m.focus == FocusConfig && m.configEditor.Focused() {
			if msg.String() == "esc" {
				_ = m.syncTabFromEditor()
				m.focus = FocusTabs
				m.updateFocusStates()
				return m, nil
			}
			var cCmd tea.Cmd
			m.configEditor, cCmd = m.configEditor.Update(msg)
			return m, cCmd
		}

		switch m.focus {
		case FocusSidebar:
			switch msg.String() {
			case "up", "k":
				if m.selectedTreeIndex > 0 {
					m.selectedTreeIndex--
				}
			case "down", "j":
				if m.selectedTreeIndex < len(m.sidebarRows)-1 {
					m.selectedTreeIndex++
				}
			case "enter", " ":
				if m.selectedTreeIndex < len(m.sidebarRows) {
					row := m.sidebarRows[m.selectedTreeIndex]
					if row.rowType == rowFolder {
						m.toggleFolder(row.folderIndex)
					} else {
						f := m.collection.Folders[row.folderIndex]
						if row.itemIndex < len(f.Items) {
							m.loadCollectionItem(f.Items[row.itemIndex])
							m.focus = FocusURL
							m.updateFocusStates()
						}
					}
				}
			case "right", "l":
				m.focus = FocusMethod
				m.updateFocusStates()
			}
			return m, nil

		case FocusMethod:
			switch msg.String() {
			case "left", "h":
				if m.methodIndex > 0 {
					m.methodIndex--
				} else {
					m.focus = FocusSidebar
					m.updateFocusStates()
				}
			case "right", "l":
				if m.methodIndex < len(m.methods)-1 {
					m.methodIndex++
				}
			case "enter", "down", "j":
				m.focus = FocusURL
				m.updateFocusStates()
			}
			return m, nil

		case FocusURL:
			if msg.String() == "enter" {
				if !m.loading {
					m.loading = true
					m.viewport.SetContent("Sending HTTP request...")
					cmds = append(cmds, m.spinner.Tick, m.executeRequestCmd())
				}
				return m, tea.Batch(cmds...)
			}
			var uCmd tea.Cmd
			m.urlInput, uCmd = m.urlInput.Update(msg)
			cmds = append(cmds, uCmd)
			return m, tea.Batch(cmds...)

		case FocusTabs:
			switch msg.String() {
			case "left", "h":
				m.tab = (m.tab - 1 + totalTabs) % totalTabs
				m.updateFocusStates()
			case "right", "l":
				m.tab = (m.tab + 1) % totalTabs
				m.updateFocusStates()
			case "enter", "down", "j":
				m.focus = FocusConfig
				m.updateFocusStates()
			}
			return m, nil

		case FocusConfig:
			switch m.tab {
			case TabHeaders:
				switch msg.String() {
				case "up":
					if m.headersFocusIndex > 0 {
						m.headersFocusIndex--
						m.updateFocusStates()
						return m, nil
					}
				case "down", "enter":
					if m.headersFocusIndex < 2 {
						m.headersFocusIndex++
						m.updateFocusStates()
						return m, nil
					}
				}
				var hCmd tea.Cmd
				switch m.headersFocusIndex {
				case 0:
					m.headerKey, hCmd = m.headerKey.Update(msg)
				case 1:
					m.headerVal, hCmd = m.headerVal.Update(msg)
				case 2:
					m.headerAuth, hCmd = m.headerAuth.Update(msg)
				}
				cmds = append(cmds, hCmd)
				return m, tea.Batch(cmds...)

			case TabBodyRaw:
				if msg.String() == "esc" {
					m.focus = FocusTabs
					m.updateFocusStates()
					return m, nil
				}
				var taCmd tea.Cmd
				m.jsonBody, taCmd = m.jsonBody.Update(msg)
				cmds = append(cmds, taCmd)
				return m, tea.Batch(cmds...)

			case TabBodyForm:
				switch msg.String() {
				case "up":
					if m.formFocusIndex > 0 {
						m.formFocusIndex--
						m.updateFocusStates()
						return m, nil
					}
				case "down", "enter":
					if m.formFocusIndex < 1 {
						m.formFocusIndex++
						m.updateFocusStates()
						return m, nil
					}
				}
				var fCmd tea.Cmd
				if m.formFocusIndex == 0 {
					m.formKey, fCmd = m.formKey.Update(msg)
				} else {
					m.formFilePath, fCmd = m.formFilePath.Update(msg)
				}
				cmds = append(cmds, fCmd)
				return m, tea.Batch(cmds...)
			}

		case FocusSend:
			if msg.String() == "enter" || msg.String() == " " {
				if !m.loading {
					m.loading = true
					m.viewport.SetContent("Sending HTTP request...")
					cmds = append(cmds, m.spinner.Tick, m.executeRequestCmd())
				}
				return m, tea.Batch(cmds...)
			}

		case FocusResponse:
			if msg.String() == "left" || msg.String() == "h" {
				m.responseTab = (m.responseTab + 2) % 3
				m.refreshResponse()
				return m, nil
			}
			if msg.String() == "right" || msg.String() == "l" {
				m.responseTab = (m.responseTab + 1) % 3
				m.refreshResponse()
				return m, nil
			}
			if msg.String() == "q" {
				return m, tea.Quit
			}
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
			return m, tea.Batch(cmds...)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Starting Martis..."
	}

	sidebarWidth := 28
	remWidth := m.width - sidebarWidth - 6
	halfWidth := remWidth / 2
	if halfWidth < 30 {
		halfWidth = 30
	}
	panelHeight := m.height - 6
	if panelHeight < 10 {
		panelHeight = 10
	}

	sidebarContent := m.renderSidebar(sidebarWidth)
	sidebarBorder := styles.Panel.Width(sidebarWidth).Height(panelHeight)
	if m.focus == FocusSidebar {
		sidebarBorder = styles.ActivePanel.Width(sidebarWidth).Height(panelHeight)
	}
	sidebarPanel := sidebarBorder.Render(sidebarContent)

	leftContent := m.renderRequestBuilder(halfWidth)
	leftBorder := styles.Panel.Width(halfWidth).Height(panelHeight)
	if m.focus != FocusResponse && m.focus != FocusSidebar {
		leftBorder = styles.ActivePanel.Width(halfWidth).Height(panelHeight)
	}
	leftPanel := leftBorder.Render(leftContent)

	rightContent := m.renderResponseViewer(halfWidth)
	rightBorder := styles.Panel.Width(halfWidth).Height(panelHeight)
	if m.focus == FocusResponse {
		rightBorder = styles.ActivePanel.Width(halfWidth).Height(panelHeight)
	}
	rightPanel := rightBorder.Render(rightContent)

	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, " ", leftPanel, " ", rightPanel)
	header := styles.Title.Render("⚡ MARTIS TUI - Ultra-Light REST Client")
	footer := styles.Help.Render(
		"[Ctrl+S] Send [Ctrl+E] Save [Ctrl+H] History [Ctrl+G] Env [Ctrl+I] cURL in [Ctrl+X] cURL out [Ctrl+B] Benchmark [Ctrl+O] Save response",
	)
	if m.activeEnvName != "" {
		footer += "  env=" + m.activeEnvName
	}
	if m.status != "" {
		footer += "  " + m.status
	}

	rendered := lipgloss.JoinVertical(lipgloss.Left, header, mainBody, footer)
	if m.modal != modalNone {
		return m.renderModal(rendered)
	}
	if m.saveModalOpen {
		return m.renderSaveModal(rendered)
	}
	return rendered
}

func (m Model) renderSidebar(width int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor).Render("📂 COLLECTIONS"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("Click or ↑/↓/Enter"))

	for i, row := range m.sidebarRows {
		prefix := "  "
		if i == m.selectedTreeIndex && m.focus == FocusSidebar {
			prefix = "▶ "
		}

		if row.rowType == rowFolder {
			icon := "📁 ▶"
			if row.isExpanded {
				icon = "📂 ▼"
			}
			folderText := ansi.Truncate(fmt.Sprintf("%s%s %s", prefix, icon, row.name), width-4, "…")
			if i == m.selectedTreeIndex && m.focus == FocusSidebar {
				lines = append(lines, styles.ActiveRow.Render(folderText))
			} else {
				lines = append(lines, styles.Folder.Render(folderText))
			}
		} else {
			methodColor := lipgloss.Color("#00D7D7")
			if row.method == "POST" {
				methodColor = lipgloss.Color("#00E676")
			} else if row.method == "DELETE" {
				methodColor = lipgloss.Color("#FF5252")
			}
			badge := lipgloss.NewStyle().Foreground(methodColor).Bold(true).Render(row.method)
			itemText := ansi.Truncate(fmt.Sprintf("%s  • %s %s", prefix, badge, row.name), width-4, "…")

			if i == m.selectedTreeIndex && m.focus == FocusSidebar {
				lines = append(lines, styles.ActiveRow.Render(itemText))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")).Render(itemText))
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderSaveModal(background string) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor).Render("💾 Save Request to Collection"),
		"",
		styles.Label.Render("Enter request name:"),
		m.saveNameInput.View(),
		"",
		lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("[Enter] Save • [Esc] Cancel"),
	)
	modal := styles.ModalBox.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderRequestBuilder(width int) string {
	var sections []string

	var methodBadges []string
	for i, meth := range m.methods {
		if i == m.methodIndex {
			if m.focus == FocusMethod {
				methodBadges = append(methodBadges, styles.ActiveMethod.Render("▶ "+meth))
			} else {
				methodBadges = append(methodBadges, styles.Method.Render(meth))
			}
		} else {
			methodBadges = append(methodBadges, lipgloss.NewStyle().Foreground(styles.SubtleColor).Render(" "+meth+" "))
		}
	}
	methodRow := lipgloss.JoinHorizontal(lipgloss.Center, methodBadges...)

	sendBtn := styles.SendBtn.Render(" Send [Ctrl+S] ")
	if m.focus == FocusSend {
		sendBtn = styles.ActiveSendBtn.Render("▶ Send [Ctrl+S] ")
	}
	if m.loading {
		sendBtn = lipgloss.NewStyle().Background(styles.WarningColor).Foreground(lipgloss.Color("#000000")).Render(
			fmt.Sprintf(" %s Sending... ", m.spinner.View()),
		)
	}

	saveBtn := lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("[Ctrl+E] Save")
	topBar := lipgloss.JoinHorizontal(lipgloss.Center, methodRow, "  ", sendBtn, "  ", saveBtn)
	sections = append(sections, topBar)

	urlPrompt := styles.Label.Render("URL: ")
	if m.focus == FocusURL {
		urlPrompt = lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true).Render("URL ▶ ")
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, urlPrompt, m.urlInput.View()))

	tabLabels := []string{"1.Headers", "2.JSON", "3.Form", "4.Query", "5.Auth", "6.Tests"}
	var renderedTabs []string
	for i, label := range tabLabels {
		if ConfigTab(i) == m.tab {
			if m.focus == FocusTabs {
				renderedTabs = append(renderedTabs, styles.ActiveTab.Copy().Background(styles.AccentColor).Foreground(lipgloss.Color("#000000")).Render("▶ "+label))
			} else {
				renderedTabs = append(renderedTabs, styles.ActiveTab.Render(label))
			}
		} else {
			renderedTabs = append(renderedTabs, styles.InactiveTab.Render(label))
		}
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...))

	var configContent string
	switch m.tab {
	case TabHeaders:
		kPrefix := "  "
		vPrefix := "  "
		aPrefix := "  "
		if m.focus == FocusConfig {
			switch m.headersFocusIndex {
			case 0:
				kPrefix = "▶ "
			case 1:
				vPrefix = "▶ "
			case 2:
				aPrefix = "▶ "
			}
		}
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.Label.Render("Custom Header (Key : Value):"),
			lipgloss.JoinHorizontal(lipgloss.Left, kPrefix, m.headerKey.View(), " : ", vPrefix, m.headerVal.View()),
			"",
			styles.Label.Render("Authorization Header:"),
			lipgloss.JoinHorizontal(lipgloss.Left, aPrefix, m.headerAuth.View()),
			"",
			lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("Tip: Up/Down arrows to move between header fields."),
			styles.Label.Render("Additional headers (one Key: Value per line):"),
			m.configEditor.View(),
		)

	case TabBodyRaw:
		prefix := "Payload (Raw JSON):"
		if m.focus == FocusConfig {
			prefix = "▶ Payload (Raw JSON): [Esc to unfocus textarea]"
		}
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.Label.Render(prefix),
			m.jsonBody.View(),
		)

	case TabBodyForm:
		kPrefix := "  "
		fPrefix := "  "
		if m.focus == FocusConfig {
			if m.formFocusIndex == 0 {
				kPrefix = "▶ "
			} else {
				fPrefix = "▶ "
			}
		}
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.Label.Render("Form-Data File Upload:"),
			styles.Label.Render("Field / Key Name:"),
			lipgloss.JoinHorizontal(lipgloss.Left, kPrefix, m.formKey.View()),
			"",
			styles.Label.Render("Local File Path:"),
			lipgloss.JoinHorizontal(lipgloss.Left, fPrefix, m.formFilePath.View()),
			"",
			lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("Tip: Enter local file path to test multipart upload."),
		)
	case TabQuery, TabAuth, TabAssertions:
		label := "Query Parameters (key=value per row)"
		if m.tab == TabAuth {
			label = "Authentication JSON (none, bearer, basic, api-key, oauth2)"
		}
		if m.tab == TabAssertions {
			label = "Assertions (Status == 200 / json.id != nil)"
		}
		configContent = lipgloss.JoinVertical(lipgloss.Left, styles.Label.Render(label), m.configEditor.View())
	}

	sections = append(sections, configContent)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderResponseViewer(width int) string {
	var statusBadge string
	var metaStats string

	if m.lastResp == nil {
		statusBadge = styles.MetaBadge.Render("STATUS: IDLE")
		metaStats = styles.MetaBadge.Render("Time: 0ms • Size: 0B")
	} else if m.lastResp.Err != nil {
		statusBadge = styles.StatusErr.Render("ERR: FAILED")
		metaStats = styles.MetaBadge.Render(fmt.Sprintf("Time: %dms", m.lastResp.Duration.Milliseconds()))
	} else {
		code := m.lastResp.StatusCode
		statusStr := fmt.Sprintf("%d %s", code, http.StatusText(code))
		if code >= 200 && code < 300 {
			statusBadge = styles.Status2xx.Render(statusStr)
		} else if code >= 300 && code < 400 {
			statusBadge = styles.Status3xx.Render(statusStr)
		} else {
			statusBadge = styles.StatusErr.Render(statusStr)
		}

		bodySize := len(m.lastResp.Body)
		metaStats = styles.MetaBadge.Render(
			fmt.Sprintf("Time: %dms • Size: %s", m.lastResp.Duration.Milliseconds(), domain.FormatBytes(bodySize)),
		)
	}

	statusBar := lipgloss.JoinHorizontal(lipgloss.Center, statusBadge, "  ", metaStats)
	var focusIndicator string
	if m.focus == FocusResponse {
		focusIndicator = lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true).Render(" [VIEWPORT ACTIVE: Scroll with ↑/↓/j/k]")
	} else {
		focusIndicator = lipgloss.NewStyle().Foreground(styles.SubtleColor).Render(" [Tab into Viewport to scroll]")
	}

	responseTabs := []string{"Body", "Headers", "Cookies"}
	for i, name := range responseTabs {
		if i == m.responseTab {
			responseTabs[i] = "▶ " + name
		}
	}
	responseContent := m.viewport.View()
	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		focusIndicator,
		lipgloss.JoinHorizontal(lipgloss.Left, responseTabs...),
		"",
		lipgloss.NewStyle().Width(width-4).Render(responseContent),
	)
}

func (m *Model) refreshResponse() {
	if !m.viewportReady {
		return
	}
	content := m.responseBody
	if m.responseTab == 1 {
		content = formatResponseHeaders(m.responseHeaders)
	}
	if m.responseTab == 2 {
		content = strings.Join(m.responseHeaders.Values("Set-Cookie"), "\n")
	}
	if m.responseSearch != "" {
		var matches []string
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(strings.ToLower(line), strings.ToLower(m.responseSearch)) {
				matches = append(matches, line)
			}
		}
		content = strings.Join(matches, "\n")
	}
	m.viewport.SetContent(content)
	m.viewport.GotoTop()
}

func (m Model) renderModal(background string) string {
	var title, body string
	switch m.modal {
	case modalSave:
		title = "Save Request"
		body = m.modalInput.View() + "\nEnter to save"
	case modalCurlImport:
		title = "Import cURL"
		body = m.modalInput.View() + "\nCtrl+Enter to import"
	case modalSearch:
		title = "Search Response"
		body = m.modalInput.View() + "\nCtrl+Enter to filter • clear search with empty input"
	case modalBenchmark:
		title = "Benchmark Runner"
		body = m.modalInput.View() + "\nRequest count, maximum 1000. Ctrl+Enter to run"
	case modalHistory:
		title = "Request History"
		for i, entry := range m.history {
			name := fmt.Sprintf("%s %s [%d]", entry.Request.Method, entry.Request.URL, entry.Status)
			if i == m.modalIndex {
				name = "▶ " + name
			}
			body += name + "\n"
		}
		body += "↑/↓ select • Enter load • Esc close"
	case modalEnvironment:
		title = "Environment"
		for i, path := range m.envFiles {
			name := filepath.Base(path)
			if i == m.modalIndex {
				name = "▶ " + name
			}
			body += name + "\n"
		}
		body += "Place .env files in ~/.config/martis/environments\n↑/↓ select • Enter activate"
	case modalTheme:
		title = "Theme"
		for i, name := range []string{"Dracula", "Catppuccin", "Nord", "Tokyo Night", "Monokai"} {
			if i == m.modalIndex {
				name = "▶ " + name
			}
			body += name + "\n"
		}
		body += "↑/↓ select • Enter apply"
	}
	if m.modalError != "" {
		body += "\nError: " + m.modalError
	}
	modal := styles.ModalBox.Render(lipgloss.JoinVertical(lipgloss.Left, styles.Label.Render(title), "", body, "", lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("Esc closes")))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
