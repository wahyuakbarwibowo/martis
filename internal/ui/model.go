package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
	TabGraphQL
)

const totalTabs = 7

// usesConfigEditor reports whether the tab is edited through the shared configEditor.
func (t ConfigTab) usesConfigEditor() bool {
	switch t {
	case TabHeaders, TabQuery, TabAuth, TabAssertions, TabGraphQL:
		return true
	}
	return false
}

// gqlSeparator splits the GraphQL query from its JSON variables in the GQL tab.
const gqlSeparator = "### variables"

func configTabAtX(x int) ConfigTab {
	labels := []string{"Hdr", "JSON", "Form", "Query", "Auth", "Tests", "GQL"}
	for i, label := range labels {
		width := lipgloss.Width(styles.InactiveTab.Render(label))
		if x < width {
			return ConfigTab(i)
		}
		x -= width
	}
	return TabGraphQL
}

var responseTabNames = []string{"Body", "Headers", "Cookies"}

func responseTabAtX(x int) int {
	for i, name := range responseTabNames {
		width := lipgloss.Width(styles.InactiveTab.Render(name))
		if x < width {
			return i
		}
		x -= width
	}
	return len(responseTabNames) - 1
}

func (m Model) collectionItems() []treeRow {
	var rows []treeRow
	if m.collection == nil {
		return rows
	}
	for folderIndex, folder := range m.collection.Folders {
		for itemIndex, item := range folder.Items {
			rows = append(rows, treeRow{rowType: rowItem, folderIndex: folderIndex, itemIndex: itemIndex, name: folder.Name + " / " + item.Name, method: item.Method})
		}
	}
	return rows
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
	modalAuth
	modalCollections
	modalFolder
	modalFilePicker
	modalRename
	modalMove
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

type filePickerResultMsg struct {
	path string
	err  error
}

func nativeFilePickerCmd() tea.Msg {
	if runtime.GOOS != "darwin" {
		return filePickerResultMsg{err: fmt.Errorf("native picker unavailable")}
	}
	// macOS presents the standard Finder-backed file chooser. The user can
	// browse folders, preview files, and select any readable local file.
	script := `try
set pickedFile to choose file with prompt "Choose form-data file"
return POSIX path of pickedFile
on error number -128
error "user canceled"
end try`
	out, err := osexec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return filePickerResultMsg{err: fmt.Errorf("%s", strings.TrimSpace(string(out)))}
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return filePickerResultMsg{err: fmt.Errorf("user canceled")}
	}
	return filePickerResultMsg{path: path}
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
	renameFolderIndex int
	renameItemIndex   int
	renameIsFolder    bool
	moveFolderIndex   int
	moveItemFolder    int
	moveItemIndex     int
	fileCandidates    []string
	filePickerDir     string
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
	gqlQuery          string
	gqlVars           string
	graphql           bool // body is sent as GraphQL (last body tab used was GQL)
	streamCancel      context.CancelFunc
	streamCh          chan tea.Msg
	streamEvents      int
	streamText        *strings.Builder
	prevResponseBody  string
	responseDiff      bool
	showLargeBody     bool
	lastPayload       domain.RequestPayload

	methods      []string
	methodIndex  int
	urlInput     textinput.Model
	headerKey    textinput.Model
	headerVal    textinput.Model
	headerAuth   textinput.Model
	jsonBody     textarea.Model
	bodyFile     string
	formKey      textinput.Model
	formFilePath textinput.Model
	formEditor   textarea.Model
	formFields   []domain.KeyValue
	formFiles    []domain.KeyValue
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
	urlIn.Prompt = ""

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
	sp.Style = lipgloss.NewStyle().Foreground(styles.WarningColor)
	editor := textarea.New()
	editor.SetHeight(8)
	editor.ShowLineNumbers = true
	formEditor := textarea.New()
	formEditor.Placeholder = "field=value\n@file=/path/to/file"
	formEditor.SetHeight(6)
	formEditor.ShowLineNumbers = true
	modalEditor := textarea.New()
	modalEditor.SetHeight(12)
	modalEditor.SetWidth(64)
	for _, t := range []*textarea.Model{&ta, &editor, &formEditor, &modalEditor} {
		quietTextarea(t)
	}

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
		formEditor:    formEditor,
		modalInput:    modalEditor,
		history:       nil,
		activeEnv:     map[string]string{},
	}
	m.history, _ = repository.LoadHistory(filepath.Join(repository.ConfigDir(), "history.json"))
	m.envFiles, _ = environment.Files(filepath.Join(repository.ConfigDir(), "environments"))
	if len(m.envFiles) == 0 {
		m.envFiles, _ = environment.Files(filepath.Join(repository.LegacyConfigDir(), "environments"))
	}
	if data, err := os.ReadFile(filepath.Join(repository.ConfigDir(), "preferences.json")); err == nil {
		var prefs struct {
			Theme int `json:"theme"`
		}
		if json.Unmarshal(data, &prefs) == nil && prefs.Theme >= 0 && prefs.Theme < 5 {
			m.themeIndex = prefs.Theme
			m.applyTheme()
		}
	}

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
	m.formEditor.Blur()
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
			m.configEditor.Focus()
		case TabBodyRaw:
			m.jsonBody.Focus()
		case TabBodyForm:
			m.formEditor.Focus()
		}
		if m.tab.usesConfigEditor() {
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
	// Keep legacy single-file fields while restoring newer multipart entries.
	m.auth = item.Auth
	m.assertions = item.Assertions
	m.syncEditorFromTab()

	m.graphql = item.BodyType == "graphql"
	if m.graphql {
		m.tab = TabGraphQL
		m.gqlQuery, m.gqlVars = item.BodyRaw, item.Variables
	} else if item.BodyType == "form" {
		m.tab = TabBodyForm
		m.formKey.SetValue(item.FormKey)
		m.formFilePath.SetValue(item.FormPath)
		m.formFields = append([]domain.KeyValue(nil), item.FormFields...)
		m.formFiles = append([]domain.KeyValue(nil), item.FormFiles...)
		m.syncFormEditorView()
	} else {
		m.tab = TabBodyRaw
		m.jsonBody.SetValue(item.BodyRaw)
		m.bodyFile = item.BodyFile
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
		if m.headerKey.Value() != "" {
			text += m.headerKey.Value() + ": " + m.headerVal.Value() + "\n"
		}
		for _, h := range m.extraHeaders {
			text += h.Key + ": " + h.Value + "\n"
		}
		if m.headerAuth.Value() != "" {
			text += "Authorization: " + m.headerAuth.Value() + "\n"
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
	case TabGraphQL:
		text = m.gqlQuery + "\n" + gqlSeparator + "\n" + m.gqlVars
	}
	m.configEditor.SetValue(strings.TrimSuffix(text, "\n"))
}

func (m *Model) syncFormEditorView() {
	var lines []string
	for _, field := range m.formFields {
		lines = append(lines, field.Key+"="+field.Value)
	}
	if m.formKey.Value() != "" && m.formFilePath.Value() != "" {
		lines = append(lines, "@"+m.formKey.Value()+"="+m.formFilePath.Value())
	}
	for _, field := range m.formFiles {
		lines = append(lines, "@"+field.Key+"="+field.Value)
	}
	m.formEditor.SetValue(strings.Join(lines, "\n"))
}

func (m *Model) syncFormEditor() error {
	m.formFields, m.formFiles = nil, nil
	for i, line := range strings.Split(m.formEditor.Value(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return fmt.Errorf("form line %d must be field=value or @file=path", i+1)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if strings.HasPrefix(key, "@") {
			m.formFiles = append(m.formFiles, domain.KeyValue{Key: strings.TrimPrefix(key, "@"), Value: value})
		} else {
			m.formFields = append(m.formFields, domain.KeyValue{Key: key, Value: value})
		}
	}
	if len(m.formFiles) > 0 {
		m.formKey.SetValue(m.formFiles[0].Key)
		m.formFilePath.SetValue(m.formFiles[0].Value)
		m.formFiles = m.formFiles[1:]
	}
	return nil
}

func (m *Model) syncTabFromEditor() error {
	text := m.configEditor.Value()
	switch m.tab {
	case TabHeaders:
		m.extraHeaders = nil
		m.headerKey.SetValue("")
		m.headerVal.SetValue("")
		m.headerAuth.SetValue("")
		for i, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			k, v, ok := strings.Cut(line, ":")
			if !ok || strings.TrimSpace(k) == "" {
				return fmt.Errorf("header line %d must be Key: Value", i+1)
			}
			key, value := strings.TrimSpace(k), strings.TrimSpace(v)
			if strings.EqualFold(key, "Authorization") {
				m.headerAuth.SetValue(value)
			} else if m.headerKey.Value() == "" {
				m.headerKey.SetValue(key)
				m.headerVal.SetValue(value)
			} else {
				m.extraHeaders = append(m.extraHeaders, domain.KeyValue{Key: key, Value: value})
			}
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
		if updated, err := requestutil.WithQuery(m.urlInput.Value(), m.queryRows); err == nil {
			m.urlInput.SetValue(updated)
		} else {
			return err
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
	case TabGraphQL:
		query, vars, _ := strings.Cut(text, gqlSeparator)
		m.gqlQuery, m.gqlVars = strings.TrimSpace(query), strings.TrimSpace(vars)
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
	if m.tab.usesConfigEditor() {
		if err := m.syncTabFromEditor(); err != nil {
			m.status = err.Error()
			return
		}
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value()))
	}

	bodyType := "raw"
	if m.tab == TabBodyForm {
		bodyType = "form"
	}
	m.trackBodyMode()
	bodyRaw, variables := m.jsonBody.Value(), ""
	if m.graphql {
		bodyType, bodyRaw, variables = "graphql", m.gqlQuery, m.gqlVars
	}

	newItem := domain.CollectionItem{
		Variables:  variables,
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
		BodyRaw:    bodyRaw,
		BodyFile:   m.bodyFile,
		FormKey:    m.formKey.Value(),
		FormPath:   m.formFilePath.Value(),
		FormFields: append([]domain.KeyValue(nil), m.formFields...),
		FormFiles:  append([]domain.KeyValue(nil), m.formFiles...),
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
	if m.tab == TabBodyForm {
		if err := m.syncFormEditor(); err != nil {
			return func() tea.Msg { return domain.ResponseResult{Err: err} }
		}
	}
	bodyType := "raw"
	if m.tab == TabBodyForm {
		bodyType = "form"
	}
	m.trackBodyMode()

	payload := domain.RequestPayload{
		Method:     m.methods[m.methodIndex],
		URL:        m.urlInput.Value(),
		HeaderKey:  m.headerKey.Value(),
		HeaderVal:  m.headerVal.Value(),
		HeaderAuth: m.headerAuth.Value(),
		BodyType:   bodyType,
		BodyRaw:    m.jsonBody.Value(),
		BodyFile:   m.bodyFile,
		FormKey:    m.formKey.Value(),
		FormPath:   m.formFilePath.Value(),
		FormFields: append([]domain.KeyValue(nil), m.formFields...),
		FormFiles:  append([]domain.KeyValue(nil), m.formFiles...),
	}

	if m.tab.usesConfigEditor() {
		if err := m.syncTabFromEditor(); err != nil {
			return func() tea.Msg { return domain.ResponseResult{Err: err} }
		}
	}
	payload.HeaderKey = m.headerKey.Value()
	payload.HeaderVal = m.headerVal.Value()
	payload.HeaderAuth = m.headerAuth.Value()
	payload.Headers = append([]domain.KeyValue(nil), m.extraHeaders...)
	payload.Auth = m.auth
	payload.Assertions = m.assertions
	if m.graphql {
		payload.BodyType, payload.BodyRaw, payload.Variables, payload.BodyFile = "graphql", m.gqlQuery, m.gqlVars, ""
	}
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

	// Run through Stream so Server-Sent Events show up as they arrive.
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan tea.Msg, 64)
	m.streamCancel, m.streamCh, m.streamEvents, m.streamText = cancel, ch, 0, &strings.Builder{}
	client := m.httpClient
	go func() {
		defer close(ch)
		defer cancel()
		ch <- httpclient.Stream(ctx, client, prepared, func(event string) { ch <- sseEventMsg(event) })
	}()
	return waitStream(ch)
}

// sseEventMsg carries one Server-Sent Event while a stream is open.
type sseEventMsg string

// waitStream delivers the next message from a running request.
func waitStream(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// trackBodyMode remembers whether the last body tab used was GQL or JSON/Form.
func (m *Model) trackBodyMode() {
	switch m.tab {
	case TabGraphQL:
		m.graphql = true
	case TabBodyRaw, TabBodyForm:
		m.graphql = false
	}
}

func (m *Model) currentPayload() domain.RequestPayload {
	if m.tab == TabBodyForm {
		_ = m.syncFormEditor()
	}
	p := domain.RequestPayload{Method: m.methods[m.methodIndex], URL: m.urlInput.Value(), HeaderKey: m.headerKey.Value(), HeaderVal: m.headerVal.Value(), HeaderAuth: m.headerAuth.Value(), BodyType: "raw", BodyRaw: m.jsonBody.Value(), FormKey: m.formKey.Value(), FormPath: m.formFilePath.Value(), FormFields: append([]domain.KeyValue(nil), m.formFields...), FormFiles: append([]domain.KeyValue(nil), m.formFiles...), Headers: append([]domain.KeyValue(nil), m.extraHeaders...), Auth: m.auth, Assertions: m.assertions}
	p.BodyFile = m.bodyFile
	if m.tab.usesConfigEditor() {
		_ = m.syncTabFromEditor()
		p.HeaderKey = m.headerKey.Value()
		p.HeaderVal = m.headerVal.Value()
		p.HeaderAuth = m.headerAuth.Value()
		p.Headers = append([]domain.KeyValue(nil), m.extraHeaders...)
		p.Auth = m.auth
		p.Assertions = m.assertions
	}
	if m.tab == TabBodyForm {
		p.BodyType = "form"
	}
	m.trackBodyMode()
	if m.graphql {
		p.BodyType, p.BodyRaw, p.Variables, p.BodyFile = "graphql", m.gqlQuery, m.gqlVars, ""
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
	return domain.CollectionItem{ID: fmt.Sprintf("history-%d", time.Now().UnixNano()), Name: p.Method + " " + filepath.Base(p.URL), Method: p.Method, URL: p.URL, HeaderKey: p.HeaderKey, HeaderVal: p.HeaderVal, HeaderAuth: p.HeaderAuth, BodyType: p.BodyType, BodyRaw: p.BodyRaw, BodyFile: p.BodyFile, FormKey: p.FormKey, FormPath: p.FormPath, FormFields: append([]domain.KeyValue(nil), p.FormFields...), FormFiles: append([]domain.KeyValue(nil), p.FormFiles...), Headers: append([]domain.KeyValue(nil), p.Headers...), Auth: p.Auth, Assertions: p.Assertions, Variables: p.Variables}
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
	if m.modal == modalCollections {
		items := m.collectionItems()
		if msg.String() == "up" && len(items) > 0 {
			m.modalIndex = (m.modalIndex - 1 + len(items)) % len(items)
			return nil
		}
		if msg.String() == "down" && len(items) > 0 {
			m.modalIndex = (m.modalIndex + 1) % len(items)
			return nil
		}
		if (msg.String() == "d" || msg.String() == "backspace") && len(items) > 0 {
			row := items[m.modalIndex]
			folder := &m.collection.Folders[row.folderIndex]
			folder.Items = append(folder.Items[:row.itemIndex], folder.Items[row.itemIndex+1:]...)
			if err := m.repo.Save(m.collection); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.rebuildSidebarRows()
			if m.modalIndex >= len(m.collectionItems()) {
				m.modalIndex = len(m.collectionItems()) - 1
			}
			if m.modalIndex < 0 {
				m.modalIndex = 0
			}
			return nil
		}
		if msg.String() == "y" && len(items) > 0 {
			row := items[m.modalIndex]
			folder := &m.collection.Folders[row.folderIndex]
			copyItem := folder.Items[row.itemIndex]
			copyItem.ID = fmt.Sprintf("item-%d", time.Now().UnixNano())
			copyItem.Name += " (copy)"
			folder.Items = append(folder.Items, copyItem)
			if err := m.repo.Save(m.collection); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.rebuildSidebarRows()
			m.status = "Duplicated request"
			return nil
		}
		if msg.String() == "m" && len(items) > 0 {
			row := items[m.modalIndex]
			m.moveItemFolder, m.moveItemIndex = row.folderIndex, row.itemIndex
			m.moveFolderIndex = row.folderIndex
			m.modal = modalMove
			m.modalIndex = row.folderIndex
			m.updateFocusStates()
			return nil
		}
		if msg.String() == "enter" && len(items) > 0 {
			row := items[m.modalIndex]
			m.loadCollectionItem(m.collection.Folders[row.folderIndex].Items[row.itemIndex])
			m.focus = FocusURL
			m.modal = modalNone
			m.updateFocusStates()
			return nil
		}
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
			if err := repository.WriteJSON(filepath.Join(repository.ConfigDir(), "preferences.json"), struct {
				Theme int `json:"theme"`
			}{Theme: m.themeIndex}); err != nil {
				m.status = "save theme preference: " + err.Error()
			}
			m.status = "Theme: " + []string{"Dracula", "Catppuccin", "Nord", "Tokyo Night", "Monokai"}[m.themeIndex]
		}
		m.modal = modalNone
		m.updateFocusStates()
		return nil
	}
	if m.modal == modalAuth {
		if msg.String() != "up" && msg.String() != "down" && msg.String() != "enter" {
			return nil
		}
		presets := []domain.AuthConfig{{Mode: "none"}, {Mode: "bearer"}, {Mode: "basic"}, {Mode: "api-key", Location: "header"}, {Mode: "api-key", Location: "query"}, {Mode: "oauth2"}}
		if msg.String() == "up" {
			m.modalIndex = (m.modalIndex - 1 + len(presets)) % len(presets)
			return nil
		}
		if msg.String() == "down" {
			m.modalIndex = (m.modalIndex + 1) % len(presets)
			return nil
		}
		m.auth = presets[m.modalIndex]
		m.tab = TabAuth
		m.focus = FocusTabs
		m.syncEditorFromTab()
		m.modal = modalNone
		m.updateFocusStates()
		m.status = "Authentication preset: " + m.auth.Mode
		return nil
	}
	if m.modal == modalFilePicker {
		count := len(m.fileCandidates)
		switch msg.String() {
		case "up":
			if count > 0 {
				m.modalIndex = (m.modalIndex - 1 + count) % count
			}
			return nil
		case "down":
			if count > 0 {
				m.modalIndex = (m.modalIndex + 1) % count
			}
			return nil
		case "enter":
			if count > 0 {
				m.formFilePath.SetValue(filepath.Join(m.filePickerDir, m.fileCandidates[m.modalIndex]))
				m.syncFormEditorView()
				m.tab = TabBodyForm
				m.focus = FocusConfig
				m.formFocusIndex = 1
				m.modal = modalNone
				m.updateFocusStates()
			}
			return nil
		}
		return nil
	}
	if m.modal == modalMove {
		count := len(m.collection.Folders)
		switch msg.String() {
		case "up":
			if count > 0 {
				m.modalIndex = (m.modalIndex - 1 + count) % count
			}
			return nil
		case "down":
			if count > 0 {
				m.modalIndex = (m.modalIndex + 1) % count
			}
			return nil
		case "enter":
			if count == 0 || m.moveItemFolder < 0 || m.moveItemFolder >= count {
				return nil
			}
			if m.modalIndex == m.moveItemFolder {
				m.modalError = "Choose a different folder"
				return nil
			}
			from := &m.collection.Folders[m.moveItemFolder]
			if m.moveItemIndex < 0 || m.moveItemIndex >= len(from.Items) {
				m.modalError = "Invalid request target"
				return nil
			}
			item := from.Items[m.moveItemIndex]
			from.Items = append(from.Items[:m.moveItemIndex], from.Items[m.moveItemIndex+1:]...)
			m.collection.Folders[m.modalIndex].Items = append(m.collection.Folders[m.modalIndex].Items, item)
			m.collection.Folders[m.modalIndex].IsExpanded = true
			if err := m.repo.Save(m.collection); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.rebuildSidebarRows()
			m.modal = modalNone
			m.status = "Moved request to " + m.collection.Folders[m.modalIndex].Name
			m.updateFocusStates()
			return nil
		}
		return nil
	}
	if msg.String() == "ctrl+enter" || (msg.String() == "enter" && (m.modal == modalSave || m.modal == modalFolder || m.modal == modalRename)) {
		text := strings.TrimSpace(m.modalInput.Value())
		switch m.modal {
		case modalSave:
			m.saveCurrentToCollection(text)
			m.status = "Saved request to collection"
		case modalFolder:
			name := strings.TrimSpace(text)
			if name == "" {
				m.modalError = "Folder name is required"
				return nil
			}
			if m.collection == nil {
				m.collection = &domain.Collection{Name: "Default Workspace"}
			}
			m.collection.Folders = append(m.collection.Folders, domain.Folder{ID: fmt.Sprintf("folder-%d", time.Now().UnixNano()), Name: name, IsExpanded: true})
			if err := m.repo.Save(m.collection); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.rebuildSidebarRows()
			m.selectedTreeIndex = len(m.sidebarRows) - 1
			m.status = "Created folder " + name
		case modalRename:
			name := strings.TrimSpace(text)
			if name == "" {
				m.modalError = "Name is required"
				return nil
			}
			if m.collection == nil || m.renameFolderIndex < 0 || m.renameFolderIndex >= len(m.collection.Folders) {
				m.modalError = "Invalid collection target"
				return nil
			}
			if m.renameIsFolder {
				m.collection.Folders[m.renameFolderIndex].Name = name
			} else if m.renameItemIndex >= 0 && m.renameItemIndex < len(m.collection.Folders[m.renameFolderIndex].Items) {
				m.collection.Folders[m.renameFolderIndex].Items[m.renameItemIndex].Name = name
			} else {
				m.modalError = "Invalid request target"
				return nil
			}
			if err := m.repo.Save(m.collection); err != nil {
				m.modalError = err.Error()
				return nil
			}
			m.rebuildSidebarRows()
			m.status = "Renamed collection item"
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
	styles.SetAccent(lipgloss.Color(colors[m.themeIndex%len(colors)]))
}

// ImportCurl fills the request editor from a cURL command before the TUI starts.
func (m *Model) ImportCurl(raw string) error { return m.importCurl(raw) }

func (m *Model) importCurl(raw string) error {
	parsed, err := curlparser.Parse(raw)
	if err != nil {
		return err
	}
	m.urlInput.SetValue(parsed.URL)
	m.headerKey.SetValue("")
	m.headerVal.SetValue("")
	m.headerAuth.SetValue("")
	m.auth = domain.AuthConfig{}
	m.extraHeaders = nil
	m.jsonBody.SetValue("")
	m.bodyFile = ""
	m.formKey.SetValue("")
	m.formFilePath.SetValue("")
	m.formEditor.SetValue("")
	m.formFields = nil
	m.formFiles = nil
	for i, method := range m.methods {
		if method == parsed.Method {
			m.methodIndex = i
		}
	}
	for key, value := range parsed.Headers {
		m.extraHeaders = append(m.extraHeaders, domain.KeyValue{Key: key, Value: value})
	}
	sort.Slice(m.extraHeaders, func(i, j int) bool { return m.extraHeaders[i].Key < m.extraHeaders[j].Key })
	m.headerAuth.SetValue(parsed.AuthHeader)
	if parsed.Body != "" {
		m.tab = TabBodyRaw
		m.jsonBody.SetValue(parsed.Body)
	}
	if parsed.BodyFile != "" {
		m.tab = TabBodyRaw
		m.bodyFile = parsed.BodyFile
	}
	if parsed.FormPath != "" {
		m.tab = TabBodyForm
		m.formKey.SetValue(parsed.FormKey)
		m.formFilePath.SetValue(parsed.FormPath)
	}
	for key, value := range parsed.FormFields {
		m.formFields = append(m.formFields, domain.KeyValue{Key: key, Value: value})
	}
	for key, value := range parsed.FormFiles {
		if key == parsed.FormKey && value == parsed.FormPath {
			continue // legacy FormKey/FormPath carries the first file
		}
		m.formFiles = append(m.formFiles, domain.KeyValue{Key: key, Value: value})
	}
	sort.Slice(m.formFields, func(i, j int) bool { return m.formFields[i].Key < m.formFields[j].Key })
	sort.Slice(m.formFiles, func(i, j int) bool { return m.formFiles[i].Key < m.formFiles[j].Key })
	m.syncFormEditorView()
	m.syncEditorFromTab()
	m.updateFocusStates()
	return nil
}

func (m *Model) openFilePicker() {
	dir, err := os.Getwd()
	if err != nil {
		m.status = "file picker: " + err.Error()
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.status = "file picker: " + err.Error()
		return
	}
	m.fileCandidates = m.fileCandidates[:0]
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		m.fileCandidates = append(m.fileCandidates, entry.Name())
	}
	sort.Strings(m.fileCandidates)
	m.filePickerDir = dir
	m.modalIndex = 0
	m.modalError = ""
	m.modal = modalFilePicker
	m.updateFocusStates()
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
		m.formEditor.SetWidth(halfWidth - 4)
		m.formEditor.SetHeight(vpHeight - 8)
		m.configEditor.SetWidth(halfWidth - 4)
		m.configEditor.SetHeight(vpHeight - 8)
		m.saveNameInput.Width = 35

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			step := -1
			if msg.Button == tea.MouseButtonWheelDown {
				step = 1
			}
			if msg.X < 28 && len(m.sidebarRows) > 0 {
				m.selectedTreeIndex += step
				if m.selectedTreeIndex < 0 {
					m.selectedTreeIndex = 0
				}
				if m.selectedTreeIndex >= len(m.sidebarRows) {
					m.selectedTreeIndex = len(m.sidebarRows) - 1
				}
				m.focus = FocusSidebar
			} else if msg.X > 30+(m.width-34)/2 && m.viewportReady {
				if step < 0 {
					m.viewport.LineUp(3)
				} else {
					m.viewport.LineDown(3)
				}
				m.focus = FocusResponse
			}
			m.updateFocusStates()
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.modal == modalFilePicker {
				if len(m.fileCandidates) > 0 {
					// The picker is centered; the first item is close to the modal top.
					row := msg.Y - (m.height-len(m.fileCandidates))/2
					if row >= 0 && row < len(m.fileCandidates) {
						m.formFilePath.SetValue(filepath.Join(m.filePickerDir, m.fileCandidates[row]))
						m.syncFormEditorView()
						m.modal = modalNone
						m.tab = TabBodyForm
						m.focus = FocusConfig
						m.formFocusIndex = 1
						m.updateFocusStates()
					}
				}
				return m, nil
			}
			sidebarWidth := 28
			if msg.X < sidebarWidth && msg.Y >= 4 {
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
				if m.tab.usesConfigEditor() {
					if err := m.syncTabFromEditor(); err != nil {
						m.status = err.Error()
					}
				}
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
			responseStart := 31 + requestWidth
			if msg.Y == 4 && msg.X >= responseStart {
				m.responseTab = responseTabAtX(msg.X - responseStart)
				m.refreshResponse()
				m.focus = FocusResponse
				return m, nil
			}
			if msg.X >= responseStart && msg.Y >= 5 {
				m.focus = FocusResponse
				return m, nil
			}
			if msg.X >= 30 && msg.X < 30+requestWidth && msg.Y == 3 {
				m.focus = FocusURL
				m.updateFocusStates()
				return m, nil
			}
			if msg.X >= 30 && msg.X < 30+requestWidth && msg.Y == 2 {
				pos := msg.X - 30
				if pos < 1+7*len(m.methods) {
					m.methodIndex = max(0, pos-1) / 7
					if m.methodIndex >= len(m.methods) {
						m.methodIndex = len(m.methods) - 1
					}
					m.focus = FocusMethod
				} else {
					m.focus = FocusSend
				}
				m.updateFocusStates()
				return m, nil
			}
			if msg.X >= 30 && msg.X < 30+requestWidth && msg.Y >= 5 {
				m.focus = FocusConfig
				if m.tab == TabBodyForm {
					if msg.Y <= 7 {
						m.formFocusIndex = 0
					} else if msg.Y <= 11 {
						m.formFocusIndex = 1
						if runtime.GOOS == "darwin" {
							return m, nativeFilePickerCmd
						}
						m.openFilePicker()
						return m, nil
					}
				}
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

	case sseEventMsg:
		m.streamEvents++
		m.streamText.WriteString(string(msg) + "\n")
		if m.streamText.Len() > largeResponseBytes { // keep the view light on long streams
			kept := m.streamText.String()[m.streamText.Len()-largeResponseBytes/2:]
			m.streamText.Reset()
			m.streamText.WriteString(kept)
		}
		m.status = fmt.Sprintf("Streaming · %d events · ^S stop", m.streamEvents)
		if m.viewportReady {
			m.viewport.SetContent(m.streamText.String())
			m.viewport.GotoBottom()
		}
		return m, waitStream(m.streamCh)
	case domain.ResponseResult:
		m.streamCancel = nil
		m.loading = false
		if msg.Err == nil && m.lastResp != nil && m.lastResp.Err == nil {
			m.prevResponseBody = m.lastResp.Body
		}
		m.responseDiff, m.showLargeBody = false, false
		m.lastResp = &msg
		m.lastStatus = msg.StatusCode
		if msg.Headers != nil {
			m.responseHeaders = msg.Headers.Clone()
		}
		m.responseBody = msg.Body
		if msg.Err != nil {
			m.viewport.SetContent(fmt.Sprintf("Request failed\n\n%v\n\nDuration: %v", msg.Err, msg.Duration))
			m.status = msg.Err.Error()
		} else {
			captured, captureFailures := requestutil.Capture(msg, m.lastPayload.Assertions)
			// ponytail: captured values live in memory only, lost on restart or env switch.
			for k, v := range captured {
				m.activeEnv[k] = v
			}
			if failures := append(requestutil.Assert(msg, m.lastPayload.Assertions), captureFailures...); len(failures) > 0 {
				m.status = "Assertion failed: " + strings.Join(failures, ", ")
			} else if len(captured) > 0 {
				m.status = fmt.Sprintf("HTTP %d, captured %d variable(s)", msg.StatusCode, len(captured))
			} else {
				m.status = fmt.Sprintf("HTTP %d", msg.StatusCode)
			}
			item := m.payloadToItem(m.lastPayload)
			if err := repository.AppendHistory(filepath.Join(repository.ConfigDir(), "history.json"), domain.HistoryEntry{At: time.Now(), Request: item, Status: msg.StatusCode}); err != nil {
				m.status = "history save failed: " + err.Error()
			} else {
				m.history, _ = repository.LoadHistory(filepath.Join(repository.ConfigDir(), "history.json"))
			}
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
	case filePickerResultMsg:
		if msg.err != nil {
			if msg.err.Error() != "user canceled" {
				m.status = "file picker: " + msg.err.Error()
			}
			return m, nil
		}
		m.formFilePath.SetValue(msg.path)
		m.syncFormEditorView()
		m.tab = TabBodyForm
		m.focus = FocusConfig
		m.formFocusIndex = 1
		m.updateFocusStates()
		m.status = "Selected file: " + filepath.Base(msg.path)
		return m, nil

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
			if m.loading && m.streamCancel != nil {
				m.streamCancel() // stop an open event stream
				return m, nil
			}
			if !m.loading {
				m.loading = true
				m.viewport.SetContent("Sending HTTP request...")
				cmds = append(cmds, m.spinner.Tick, m.executeRequestCmd())
			}
			return m, tea.Batch(cmds...)

		case "ctrl+e":
			m.openModal(modalSave, fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value())))
			return m, nil
		case "ctrl+p":
			m.openModal(modalCollections, "")
			return m, nil
		case "ctrl+n":
			m.openModal(modalFolder, "")
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
		case "ctrl+f":
			if m.tab == TabBodyForm {
				if runtime.GOOS == "darwin" {
					return m, nativeFilePickerCmd
				}
				m.openFilePicker()
			}
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
			if path, err := m.downloadResponse(); err != nil {
				m.status = "Download gagal: " + err.Error()
			} else {
				m.status = "Tersimpan di " + path
			}
			return m, nil
		case "ctrl+b":
			m.openModal(modalBenchmark, "10")
			return m, nil
		case "f2":
			m.openModal(modalTheme, "")
			return m, nil
		case "f4":
			if m.tab == TabBodyRaw {
				formatted := requestutil.IndentJSON(m.jsonBody.Value())
				if formatted == m.jsonBody.Value() && !json.Valid([]byte(formatted)) {
					m.status = "Body bukan JSON valid"
				} else {
					m.jsonBody.SetValue(formatted)
					m.status = "Body dirapikan"
				}
			}
			return m, nil
		case "f3":
			m.openModal(modalAuth, "")
			return m, nil
		case "ctrl+d":
			if m.prevResponseBody == "" {
				m.status = "Belum ada response sebelumnya untuk dibandingkan"
				return m, nil
			}
			m.responseDiff = !m.responseDiff
			m.responseTab = 0
			m.refreshResponse()
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
			if old == TabHeaders || old == TabQuery {
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
			case "r":
				if m.selectedTreeIndex < len(m.sidebarRows) {
					row := m.sidebarRows[m.selectedTreeIndex]
					m.renameFolderIndex, m.renameItemIndex = row.folderIndex, row.itemIndex
					m.renameIsFolder = row.rowType == rowFolder
					name := row.name
					m.openModal(modalRename, name)
				}
			case "m":
				if m.selectedTreeIndex < len(m.sidebarRows) {
					row := m.sidebarRows[m.selectedTreeIndex]
					if row.rowType == rowItem && len(m.collection.Folders) > 1 {
						m.moveItemFolder, m.moveItemIndex = row.folderIndex, row.itemIndex
						m.moveFolderIndex = row.folderIndex
						m.openModal(modalMove, "")
						m.modalIndex = row.folderIndex
					}
				}
			case "d", "backspace":
				if m.selectedTreeIndex < len(m.sidebarRows) && m.collection != nil {
					row := m.sidebarRows[m.selectedTreeIndex]
					folder := &m.collection.Folders[row.folderIndex]
					if row.rowType == rowFolder {
						m.collection.Folders = append(m.collection.Folders[:row.folderIndex], m.collection.Folders[row.folderIndex+1:]...)
						m.status = "Deleted folder"
					} else if row.itemIndex < len(folder.Items) {
						folder.Items = append(folder.Items[:row.itemIndex], folder.Items[row.itemIndex+1:]...)
						m.status = "Deleted request"
					}
					if err := m.repo.Save(m.collection); err != nil {
						m.status = "save collection failed: " + err.Error()
					}
					m.rebuildSidebarRows()
				}
			case "y":
				if m.selectedTreeIndex < len(m.sidebarRows) && m.collection != nil {
					row := m.sidebarRows[m.selectedTreeIndex]
					if row.rowType == rowItem && row.itemIndex < len(m.collection.Folders[row.folderIndex].Items) {
						item := m.collection.Folders[row.folderIndex].Items[row.itemIndex]
						item.ID = fmt.Sprintf("item-%d", time.Now().UnixNano())
						item.Name += " (copy)"
						m.collection.Folders[row.folderIndex].Items = append(m.collection.Folders[row.folderIndex].Items, item)
						if err := m.repo.Save(m.collection); err != nil {
							m.status = "save collection failed: " + err.Error()
						} else {
							m.status = "Duplicated request"
						}
						m.rebuildSidebarRows()
					}
				}
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
				if msg.String() == "esc" {
					if err := m.syncFormEditor(); err != nil {
						m.status = err.Error()
					}
					m.focus = FocusTabs
					m.updateFocusStates()
					return m, nil
				}
				var fCmd tea.Cmd
				m.formEditor, fCmd = m.formEditor.Update(msg)
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
			if msg.String() == "enter" && !m.showLargeBody && m.bodyPrompt() != "" && utf8.ValidString(m.responseBody) {
				m.showLargeBody = true
				m.refreshResponse()
				return m, nil
			}
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

	panel := func(focused bool, width int, content string) string {
		style := styles.Panel
		if focused {
			style = styles.ActivePanel
		}
		return style.Width(width).Height(panelHeight).Render(content)
	}
	sidebarPanel := panel(m.focus == FocusSidebar, sidebarWidth, m.renderSidebar(sidebarWidth))
	leftPanel := panel(m.focus != FocusResponse && m.focus != FocusSidebar, halfWidth, m.renderRequestBuilder(halfWidth))
	rightPanel := panel(m.focus == FocusResponse, halfWidth, m.renderResponseViewer(halfWidth))
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, " ", leftPanel, " ", rightPanel)

	left := " " + styles.Title.Render("martis") + styles.Faint.Render("  rest client")
	env := "no environment"
	if m.activeEnvName != "" {
		env = strings.TrimSuffix(m.activeEnvName, ".env")
	}
	right := styles.Muted.Render("env ") + styles.Pill.Foreground(styles.TextColor).Render(env) + " "
	header := left + strings.Repeat(" ", max(1, lipgloss.Width(mainBody)-lipgloss.Width(left)-lipgloss.Width(right))) + right

	hints := []string{"^S send", "^E save", "^H history", "^G env", "^I import", "^X curl", "^B bench", "^O save body", "^D diff", "F4 format", "F2 theme"}
	footer := styles.Help.Render(" " + strings.Join(hints, "  "))
	if m.status != "" {
		footer += styles.Faint.Render("   ·   ") + styles.Muted.Render(m.status)
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

// quietTextarea swaps the heavy default gutter for a faint hairline.
func quietTextarea(t *textarea.Model) {
	t.Prompt = "│ "
	for _, st := range []*textarea.Style{&t.FocusedStyle, &t.BlurredStyle} {
		st.Prompt = styles.Faint
		st.LineNumber = styles.Faint
		st.CursorLineNumber = styles.Muted
	}
	t.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(styles.SurfaceColor)
}

// listRow renders one selectable row in the sidebar and modals.
func listRow(text string, selected bool) string {
	if selected {
		return styles.ActiveRow.Render(" " + ansi.Strip(text) + " ")
	}
	return " " + text
}

func (m Model) renderSidebar(width int) string {
	lines := []string{
		styles.Muted.Bold(true).Render(" COLLECTIONS"),
		styles.Faint.Render(" click or ↑/↓ enter"),
	}
	for i, row := range m.sidebarRows {
		selected := i == m.selectedTreeIndex && m.focus == FocusSidebar
		var text string
		if row.rowType == rowFolder {
			chevron := "▸"
			if row.isExpanded {
				chevron = "▾"
			}
			text = styles.Faint.Render(chevron) + " " + styles.Folder.Render(row.name)
		} else {
			text = "  " + styles.MethodStyle(row.method).Width(7).Render(row.method) + styles.Row.Render(row.name)
		}
		lines = append(lines, listRow(ansi.Truncate(text, width-3, "…"), selected))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderSaveModal(background string) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Save request"),
		"",
		styles.Label.Render("Name"),
		m.saveNameInput.View(),
		"",
		styles.Faint.Render("enter save  ·  esc cancel"),
	)
	modal := styles.ModalBox.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderRequestBuilder(width int) string {
	var sections []string

	// Each method occupies a fixed 7-cell slot; mouse clicks map x/7 to the method.
	var methodRow string
	for i, meth := range m.methods {
		if i == m.methodIndex {
			style := styles.ActiveMethod.Foreground(styles.MethodStyle(meth).GetForeground())
			if m.focus == FocusMethod {
				style = style.Underline(true)
			}
			methodRow += style.Render(meth)
		} else {
			methodRow += styles.Method.Render(meth)
		}
	}

	sendBtn := styles.SendBtn.Render("Send ^S")
	if m.focus == FocusSend {
		sendBtn = styles.ActiveSendBtn.Render("Send ^S")
	}
	if m.loading {
		sendBtn = styles.SendBtn.Foreground(styles.WarningColor).Render(m.spinner.View() + " Sending")
	}
	topBar := " " + methodRow
	// Only show Send when it fits; a wrapped row would shift the mouse map.
	if lipgloss.Width(topBar+" "+sendBtn) <= width {
		topBar += " " + sendBtn
	}
	sections = append(sections, topBar)

	urlPrompt := styles.Label.Render(" URL ")
	if m.focus == FocusURL {
		urlPrompt = lipgloss.NewStyle().Foreground(styles.AccentColor).Bold(true).Render(" URL ")
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, urlPrompt, m.urlInput.View()))

	tabLabels := []string{"Hdr", "JSON", "Form", "Query", "Auth", "Tests", "GQL"}
	var renderedTabs []string
	for i, label := range tabLabels {
		switch {
		case ConfigTab(i) == m.tab && m.focus == FocusTabs:
			renderedTabs = append(renderedTabs, styles.FocusedTab.Render(label))
		case ConfigTab(i) == m.tab:
			renderedTabs = append(renderedTabs, styles.ActiveTab.Render(label))
		default:
			renderedTabs = append(renderedTabs, styles.InactiveTab.Render(label))
		}
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...))

	var configContent string
	switch m.tab {
	case TabHeaders:
		configContent = lipgloss.JoinVertical(lipgloss.Left, styles.Label.Render(" Headers · Key: Value per line"), m.configEditor.View())
	case TabBodyRaw:
		label := " Body · raw JSON"
		if m.focus == FocusConfig {
			label += styles.Faint.Render("  esc to leave editor")
		}
		configContent = lipgloss.JoinVertical(lipgloss.Left, styles.Label.Render(label), m.jsonBody.View())
	case TabBodyForm:
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			styles.Label.Render(" Form data · field=value, @file=/path"),
			m.formEditor.View(),
			styles.Faint.Render(" ^F choose a file, or edit rows directly"),
		)
	case TabQuery, TabAuth, TabAssertions, TabGraphQL:
		label := " Query · key=value per line"
		if m.tab == TabAuth {
			label = " Auth · JSON (none, bearer, basic, api-key, oauth2) · F3 presets"
		}
		if m.tab == TabAssertions {
			label = " Tests · Status == 200 · json.id != nil · set token = json.access_token"
		}
		if m.tab == TabGraphQL {
			label = " GraphQL · query, then '" + gqlSeparator + "' and JSON variables"
		}
		configContent = lipgloss.JoinVertical(lipgloss.Left, styles.Label.Render(ansi.Truncate(label, width-2, "…")), m.configEditor.View())
	}

	sections = append(sections, configContent)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderResponseViewer(width int) string {
	status := styles.Muted.Render("No response yet")
	var meta string
	if m.lastResp != nil && m.lastResp.Err != nil {
		status = lipgloss.NewStyle().Foreground(styles.ErrorColor).Bold(true).Render("Request failed")
		meta = fmt.Sprintf("%d ms", m.lastResp.Duration.Milliseconds())
	} else if m.lastResp != nil {
		code := m.lastResp.StatusCode
		color := styles.ErrorColor
		if code < 300 {
			color = styles.SuccessColor
		} else if code < 400 {
			color = styles.WarningColor
		}
		status = lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%d %s", code, http.StatusText(code)))
		meta = fmt.Sprintf("%d ms  ·  %s", m.lastResp.Duration.Milliseconds(), domain.FormatBytes(len(m.lastResp.Body)))
	}
	if m.responseDiff {
		meta += "  ·  diff"
	}
	statusBar := " " + status + "  " + styles.Muted.Render(meta)

	hint := styles.Faint.Render(" tab to focus and scroll")
	if m.focus == FocusResponse {
		hint = styles.Faint.Render(" ↑/↓ scroll  ·  ←/→ tabs  ·  / filter  ·  ^D diff")
	}

	var tabs []string
	for i, name := range responseTabNames {
		if i == m.responseTab {
			tabs = append(tabs, styles.ActiveTab.Render(name))
		} else {
			tabs = append(tabs, styles.InactiveTab.Render(name))
		}
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		hint,
		lipgloss.JoinHorizontal(lipgloss.Left, tabs...),
		"",
		lipgloss.NewStyle().Width(width-4).Render(m.viewport.View()),
	)
}

// largeResponseBytes is the body size above which the viewer asks before rendering.
const largeResponseBytes = 1 << 20

// bodyPrompt asks whether to download or show a large or file response; "" renders the body.
func (m Model) bodyPrompt() string {
	size := domain.FormatBytes(len(m.responseBody))
	file, isFile := requestutil.DetectFile(m.responseHeaders, m.responseBody)
	switch {
	case isFile && !utf8.ValidString(m.responseBody):
		return fmt.Sprintf("File %s · %s · %s\n\nIsi file tidak ditampilkan.\n\n[Ctrl+O] Download ke ~/Downloads/%s", file.Kind, file.Name, size, file.Name)
	case isFile:
		return fmt.Sprintf("File %s · %s · %s\n\n[Ctrl+O] Download ke ~/Downloads/%s\n[Enter]  Tampilkan sebagai teks", file.Kind, file.Name, size, file.Name)
	case len(m.responseBody) > largeResponseBytes:
		return fmt.Sprintf("Response besar (%s) tidak langsung ditampilkan agar TUI tetap ringan.\n\n[Ctrl+O] Download ke ~/Downloads\n[Enter]  Tampilkan\n[/]      Filter teks atau json.path", size)
	}
	return ""
}

// downloadResponse saves the raw response bytes to ~/Downloads (or the working
// directory), using the server's file name and never overwriting a file.
func (m Model) downloadResponse() (string, error) {
	name := "response-" + time.Now().Format("20060102-150405") + ".txt"
	if file, ok := requestutil.DetectFile(m.responseHeaders, m.responseBody); ok {
		name = file.Name
	} else if strings.Contains(strings.ToLower(m.responseHeaders.Get("Content-Type")), "json") {
		name = strings.TrimSuffix(name, ".txt") + ".json"
	}
	dir := "."
	if home, err := os.UserHomeDir(); err == nil {
		if info, err := os.Stat(filepath.Join(home, "Downloads")); err == nil && info.IsDir() {
			dir = filepath.Join(home, "Downloads")
		}
	}
	path := filepath.Join(dir, name)
	ext := filepath.Ext(name)
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(name, ext), i, ext))
	}
	return path, os.WriteFile(path, []byte(m.responseBody), 0600)
}

func (m *Model) refreshResponse() {
	if !m.viewportReady {
		return
	}
	content := m.responseBody
	if len(content) <= largeResponseBytes || m.showLargeBody {
		content = requestutil.IndentJSON(content)
	}
	if m.responseTab == 0 && m.responseDiff {
		content = requestutil.Diff(m.prevResponseBody, m.responseBody)
	} else if m.responseTab == 0 && m.responseSearch == "" && !m.showLargeBody {
		if prompt := m.bodyPrompt(); prompt != "" {
			content = prompt
		}
	}
	if m.responseTab == 1 {
		content = formatResponseHeaders(m.responseHeaders)
	}
	if m.responseTab == 2 {
		content = strings.Join(m.responseHeaders.Values("Set-Cookie"), "\n")
	}
	if strings.HasPrefix(m.responseSearch, "json.") {
		if value, ok := requestutil.JSONPath(m.responseBody, m.responseSearch); ok {
			content = value
		} else {
			content = "No value at " + m.responseSearch
		}
	} else if m.responseSearch != "" {
		var matches []string
		term := strings.ToLower(m.responseSearch)
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(strings.ToLower(line), term) {
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
		body = m.modalInput.View() + "\nText or json.path (json.data.0.id) • Ctrl+Enter to filter • empty clears"
	case modalBenchmark:
		title = "Benchmark Runner"
		body = m.modalInput.View() + "\nRequest count, maximum 1000. Ctrl+Enter to run"
	case modalHistory:
		title = "Request History"
		for i, entry := range m.history {
			name := fmt.Sprintf("%s %s [%d]", entry.Request.Method, entry.Request.URL, entry.Status)
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "↑/↓ select • Enter load • Esc close"
	case modalEnvironment:
		title = "Environment"
		for i, path := range m.envFiles {
			name := filepath.Base(path)
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "Place .env files in ~/martis/environments\n↑/↓ select • Enter activate"
	case modalTheme:
		title = "Theme"
		for i, name := range []string{"Dracula", "Catppuccin", "Nord", "Tokyo Night", "Monokai"} {
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "↑/↓ select • Enter apply"
	case modalCollections:
		title = "Saved requests"
		for i, row := range m.collectionItems() {
			name := "[" + row.method + "] " + row.name
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "↑/↓ select • Enter load • y duplicate • d delete"
	case modalFolder:
		title = "New collection folder"
		body = m.modalInput.View() + "\nEnter to create"
	case modalRename:
		title = "Rename collection item"
		body = m.modalInput.View() + "\nEnter to rename"
	case modalMove:
		title = "Move request to folder"
		for i, folder := range m.collection.Folders {
			name := folder.Name
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "↑/↓ select • Enter move"
	case modalFilePicker:
		title = "Choose form-data file"
		if len(m.fileCandidates) == 0 {
			body = "No files found in " + m.filePickerDir
		} else {
			for i, name := range m.fileCandidates {
				body += listRow(name, i == m.modalIndex) + "\n"
			}
			body += "Enter/click to use file"
		}
	case modalAuth:
		title = "Authentication preset"
		for i, name := range []string{"None", "Bearer Token", "Basic Auth", "API Key in Header", "API Key in Query", "OAuth 2.0 Client Credentials"} {
			body += listRow(name, i == m.modalIndex) + "\n"
		}
		body += "Select a preset, then fill its fields in the Auth tab (JSON)."
	}
	if m.modalError != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(styles.ErrorColor).Render(m.modalError)
	}
	modal := styles.ModalBox.Render(lipgloss.JoinVertical(lipgloss.Left, styles.Title.Render(title), "", body, "", styles.Faint.Render("esc close")))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
