package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Focus & Tab Enums ---
type focusArea int

const (
	focusSidebar focusArea = iota
	focusMethod
	focusURL
	focusTabs
	focusConfig
	focusSend
	focusResponse
)

const totalFocusAreas = 7

type configTab int

const (
	tabHeaders configTab = iota
	tabBodyRaw
	tabBodyForm
)

const totalTabs = 3

// --- Tree Node Row for Flattened Sidebar Display ---
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

// --- Response Message ---
type httpResponseMsg struct {
	statusCode int
	statusText string
	proto      string
	duration   time.Duration
	body       string
	headers    http.Header
	err        error
}

// --- Model Definition ---
type model struct {
	// Window dimensions
	width  int
	height int

	// Navigation & Focus
	focus focusArea
	tab   configTab

	// Collection Tree & Sidebar
	collection        Collection
	selectedTreeIndex int
	sidebarRows       []treeRow
	saveModalOpen     bool
	saveNameInput     textinput.Model

	// Request inputs
	methods      []string
	methodIndex  int
	urlInput     textinput.Model
	headerKey    textinput.Model
	headerVal    textinput.Model
	headerAuth   textinput.Model
	jsonBody     textarea.Model
	formKey      textinput.Model
	formFilePath textinput.Model

	// Focus inside config sections
	headersFocusIndex int // 0: Key, 1: Value, 2: Auth
	formFocusIndex    int // 0: FormKey, 1: FilePath

	// Execution state
	loading  bool
	spinner  spinner.Model
	lastResp *httpResponseMsg

	// Response Viewer
	viewport      viewport.Model
	viewportReady bool
}

// --- Styling with Lipgloss ---
var (
	subtleColor    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	primaryColor   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7D56F4"}
	accentColor    = lipgloss.AdaptiveColor{Light: "#02BA83", Dark: "#02BF87"}
	warningColor   = lipgloss.AdaptiveColor{Light: "#FFB000", Dark: "#FFA500"}
	dangerColor    = lipgloss.AdaptiveColor{Light: "#E85F5F", Dark: "#ED567A"}
	activeBorder   = lipgloss.Color("#7D56F4")
	inactiveBorder = lipgloss.Color("#3C3C3C")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(inactiveBorder)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(activeBorder)

	methodStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Background(lipgloss.Color("#2A2A38")).
			Foreground(lipgloss.Color("#00D7D7"))

	activeMethodStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				Background(primaryColor).
				Foreground(lipgloss.Color("#FFFFFF"))

	sendBtnStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 2).
			Background(lipgloss.Color("#2C7A4D")).
			Foreground(lipgloss.Color("#FFFFFF"))

	activeSendBtnStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 2).
				Background(lipgloss.Color("#00E676")).
				Foreground(lipgloss.Color("#000000"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#44475A")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0B0")).
			Bold(true)

	status2xx = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#2E7D32")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	status3xx = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#F57F17")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	statusErr = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#C62828")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	metaBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECEFF4")).
			Background(lipgloss.Color("#3B4252")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	folderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFA500"))

	activeRowStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#44475A")).
			Foreground(lipgloss.Color("#FFFFFF"))

	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(primaryColor).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 2)
)

// --- Initialization ---
func initialModel() model {
	// URL Input
	urlIn := textinput.New()
	urlIn.Placeholder = "https://httpbin.org/anything"
	urlIn.SetValue("https://httpbin.org/anything")
	urlIn.CharLimit = 500
	urlIn.Width = 35

	// Header Inputs
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

	// Body textarea
	ta := textarea.New()
	ta.Placeholder = "{\n  \"hello\": \"world\"\n}"
	ta.SetValue("{\n  \"message\": \"Hello from Martis!\",\n  \"status\": \"fast\"\n}")
	ta.ShowLineNumbers = true
	ta.SetHeight(8)

	// Form inputs
	fKey := textinput.New()
	fKey.Placeholder = "Field Name (e.g. file / upload)"
	fKey.SetValue("file")

	fPath := textinput.New()
	fPath.Placeholder = "File Path (e.g. ./test.txt)"

	// Save Modal Name input
	sName := textinput.New()
	sName.Placeholder = "Request Name (e.g. Get User Profile)"
	sName.CharLimit = 100

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Load Collections from Disk
	col := loadCollections()

	m := model{
		focus:         focusSidebar,
		tab:           tabBodyRaw,
		methods:       []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
		methodIndex:   1, // POST default
		urlInput:      urlIn,
		headerKey:     hKey,
		headerVal:     hVal,
		headerAuth:    hAuth,
		jsonBody:      ta,
		formKey:       fKey,
		formFilePath:  fPath,
		saveNameInput: sName,
		spinner:       sp,
		collection:    col,
	}

	m.rebuildSidebarRows()
	m.updateFocusStates()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// Membangun baris tampilan tree view berdasarkan folder dan status IsExpanded
func (m *model) rebuildSidebarRows() {
	var rows []treeRow
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

// --- Focus Management Helper ---
func (m *model) updateFocusStates() {
	m.urlInput.Blur()
	m.headerKey.Blur()
	m.headerVal.Blur()
	m.headerAuth.Blur()
	m.jsonBody.Blur()
	m.formKey.Blur()
	m.formFilePath.Blur()
	m.saveNameInput.Blur()

	if m.saveModalOpen {
		m.saveNameInput.Focus()
		return
	}

	switch m.focus {
	case focusURL:
		m.urlInput.Focus()
	case focusConfig:
		switch m.tab {
		case tabHeaders:
			switch m.headersFocusIndex {
			case 0:
				m.headerKey.Focus()
			case 1:
				m.headerVal.Focus()
			case 2:
				m.headerAuth.Focus()
			}
		case tabBodyRaw:
			m.jsonBody.Focus()
		case tabBodyForm:
			switch m.formFocusIndex {
			case 0:
				m.formKey.Focus()
			case 1:
				m.formFilePath.Focus()
			}
		}
	}
}

// Muat item collection ke form saat ini
func (m *model) loadCollectionItem(item CollectionItem) {
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

	if item.BodyType == "form" {
		m.tab = tabBodyForm
		m.formKey.SetValue(item.FormKey)
		m.formFilePath.SetValue(item.FormPath)
	} else {
		m.tab = tabBodyRaw
		m.jsonBody.SetValue(item.BodyRaw)
	}
	m.updateFocusStates()
}

// Toggle folder expand / collapse
func (m *model) toggleFolder(fIdx int) {
	if fIdx >= 0 && fIdx < len(m.collection.Folders) {
		m.collection.Folders[fIdx].IsExpanded = !m.collection.Folders[fIdx].IsExpanded
		_ = saveCollections(m.collection)
		m.rebuildSidebarRows()
	}
}

// Simpan request aktif ke folder yang dipilih atau folder pertama
func (m *model) saveCurrentToCollection(name string) {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value()))
	}

	bodyType := "raw"
	if m.tab == tabBodyForm {
		bodyType = "form"
	}

	newItem := CollectionItem{
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
		m.collection.Folders = append(m.collection.Folders, Folder{
			ID:         "folder-default",
			Name:       "My Requests",
			IsExpanded: true,
		})
	}

	// Masukkan ke folder saat ini jika sedang memilih folder
	targetFolder := 0
	if m.selectedTreeIndex < len(m.sidebarRows) {
		targetFolder = m.sidebarRows[m.selectedTreeIndex].folderIndex
	}
	m.collection.Folders[targetFolder].Items = append(m.collection.Folders[targetFolder].Items, newItem)
	m.collection.Folders[targetFolder].IsExpanded = true
	_ = saveCollections(m.collection)
	m.rebuildSidebarRows()
}

// --- HTTP Request Command (Asynchronous) ---
func (m model) sendHTTPRequest() tea.Cmd {
	targetURL := strings.TrimSpace(m.urlInput.Value())
	if targetURL == "" {
		targetURL = "https://httpbin.org/anything"
	}
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	method := m.methods[m.methodIndex]
	rawBody := m.jsonBody.Value()
	tab := m.tab

	customHKey := strings.TrimSpace(m.headerKey.Value())
	customHVal := strings.TrimSpace(m.headerVal.Value())
	authVal := strings.TrimSpace(m.headerAuth.Value())

	formKeyVal := strings.TrimSpace(m.formKey.Value())
	filePathVal := strings.TrimSpace(m.formFilePath.Value())

	return func() tea.Msg {
		startTime := time.Now()

		var reqBody io.Reader
		var contentType string

		if method != "GET" && method != "HEAD" {
			if tab == tabBodyRaw {
				reqBody = bytes.NewBufferString(rawBody)
			} else if tab == tabBodyForm && filePathVal != "" {
				bodyBuf := &bytes.Buffer{}
				writer := multipart.NewWriter(bodyBuf)

				file, err := os.Open(filePathVal)
				if err != nil {
					return httpResponseMsg{
						err: fmt.Errorf("open file error: %w", err),
					}
				}
				defer file.Close()

				fieldName := formKeyVal
				if fieldName == "" {
					fieldName = "file"
				}

				part, err := writer.CreateFormFile(fieldName, filepath.Base(filePathVal))
				if err != nil {
					return httpResponseMsg{
						err: fmt.Errorf("create form file error: %w", err),
					}
				}
				if _, err = io.Copy(part, file); err != nil {
					return httpResponseMsg{
						err: fmt.Errorf("copy file data error: %w", err),
					}
				}

				_ = writer.Close()
				contentType = writer.FormDataContentType()
				reqBody = bodyBuf
			}
		}

		req, err := http.NewRequest(method, targetURL, reqBody)
		if err != nil {
			return httpResponseMsg{
				err: fmt.Errorf("invalid request: %w", err),
			}
		}

		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		} else if customHKey != "" && customHVal != "" {
			req.Header.Set(customHKey, customHVal)
		}
		if authVal != "" {
			req.Header.Set("Authorization", authVal)
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "Martis-TUI-Client/1.0")
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		duration := time.Since(startTime)
		if err != nil {
			return httpResponseMsg{
				err:      err,
				duration: duration,
			}
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return httpResponseMsg{
				statusCode: resp.StatusCode,
				statusText: resp.Status,
				proto:      resp.Proto,
				duration:   duration,
				headers:    resp.Header,
				err:        fmt.Errorf("reading response: %w", err),
			}
		}

		bodyStr := string(respBytes)
		var prettyJSON bytes.Buffer
		if json.Indent(&prettyJSON, respBytes, "", "  ") == nil {
			bodyStr = prettyJSON.String()
		}

		return httpResponseMsg{
			statusCode: resp.StatusCode,
			statusText: resp.Status,
			proto:      resp.Proto,
			duration:   duration,
			headers:    resp.Header,
			body:       bodyStr,
		}
	}
}

// --- Update Function ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Perhitungan Layout Tiga Kolom (Sidebar: 28, Request: Sisa/2, Response: Sisa/2)
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
		// Menangani Mouse Click pada Sidebar Folder & Request Items!
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			sidebarWidth := 28
			// Jika mouse diklik di dalam area sidebar kiri
			if msg.X <= sidebarWidth+2 && msg.Y >= 2 {
				clickedRow := msg.Y - 2 // Offset dari header bar
				if clickedRow >= 0 && clickedRow < len(m.sidebarRows) {
					m.selectedTreeIndex = clickedRow
					m.focus = focusSidebar
					row := m.sidebarRows[clickedRow]

					if row.rowType == rowFolder {
						// Klik folder: toggle expand/collapse
						m.toggleFolder(row.folderIndex)
					} else {
						// Klik item: muat request langsung ke form!
						f := m.collection.Folders[row.folderIndex]
						if row.itemIndex < len(f.Items) {
							m.loadCollectionItem(f.Items[row.itemIndex])
						}
					}
					m.updateFocusStates()
					return m, nil
				}
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var spCmd tea.Cmd
			m.spinner, spCmd = m.spinner.Update(msg)
			cmds = append(cmds, spCmd)
		}

	case httpResponseMsg:
		m.loading = false
		m.lastResp = &msg
		if msg.err != nil {
			m.viewport.SetContent(fmt.Sprintf("❌ Request Error:\n\n%v\n\nDuration: %v", msg.err, msg.duration))
		} else {
			headerLines := make([]string, 0, len(msg.headers))
			for k, v := range msg.headers {
				headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, strings.Join(v, ", ")))
			}
			content := fmt.Sprintf("// Response Headers\n%s\n\n// Response Body (%d bytes)\n%s",
				strings.Join(headerLines, "\n"),
				len(msg.body),
				msg.body,
			)
			m.viewport.SetContent(content)
		}
		m.viewport.GotoTop()

	case tea.KeyMsg:
		// Modal Save Request handling
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

		// Global Keybindings
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+s":
			if !m.loading {
				m.loading = true
				m.viewport.SetContent("Sending HTTP request...")
				cmds = append(cmds, m.spinner.Tick, m.sendHTTPRequest())
			}
			return m, tea.Batch(cmds...)

		case "ctrl+e": // Save request to collection
			m.saveModalOpen = true
			m.saveNameInput.SetValue(fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value())))
			m.updateFocusStates()
			return m, nil

		case "tab":
			m.focus = (m.focus + 1) % totalFocusAreas
			m.updateFocusStates()
			return m, nil

		case "shift+tab":
			m.focus = (m.focus - 1 + totalFocusAreas) % totalFocusAreas
			m.updateFocusStates()
			return m, nil

		case "ctrl+t": // Cycle config tabs quickly
			m.tab = (m.tab + 1) % totalTabs
			m.updateFocusStates()
			return m, nil
		}

		// Handle key per active focus area
		switch m.focus {
		case focusSidebar:
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
							m.focus = focusURL
							m.updateFocusStates()
						}
					}
				}
			case "right", "l": // Pindah fokus ke Request Builder
				m.focus = focusMethod
				m.updateFocusStates()
			}
			return m, nil

		case focusMethod:
			switch msg.String() {
			case "left", "h":
				if m.methodIndex > 0 {
					m.methodIndex--
				} else {
					m.focus = focusSidebar
					m.updateFocusStates()
				}
			case "right", "l":
				if m.methodIndex < len(m.methods)-1 {
					m.methodIndex++
				}
			case "enter", "down", "j":
				m.focus = focusURL
				m.updateFocusStates()
			}
			return m, nil

		case focusURL:
			if msg.String() == "enter" {
				if !m.loading {
					m.loading = true
					m.viewport.SetContent("Sending HTTP request...")
					cmds = append(cmds, m.spinner.Tick, m.sendHTTPRequest())
				}
				return m, tea.Batch(cmds...)
			}
			var uCmd tea.Cmd
			m.urlInput, uCmd = m.urlInput.Update(msg)
			cmds = append(cmds, uCmd)
			return m, tea.Batch(cmds...)

		case focusTabs:
			switch msg.String() {
			case "left", "h":
				m.tab = (m.tab - 1 + totalTabs) % totalTabs
				m.updateFocusStates()
			case "right", "l":
				m.tab = (m.tab + 1) % totalTabs
				m.updateFocusStates()
			case "enter", "down", "j":
				m.focus = focusConfig
				m.updateFocusStates()
			}
			return m, nil

		case focusConfig:
			switch m.tab {
			case tabHeaders:
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

			case tabBodyRaw:
				if msg.String() == "esc" {
					m.focus = focusTabs
					m.updateFocusStates()
					return m, nil
				}
				var taCmd tea.Cmd
				m.jsonBody, taCmd = m.jsonBody.Update(msg)
				cmds = append(cmds, taCmd)
				return m, tea.Batch(cmds...)

			case tabBodyForm:
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

		case focusSend:
			if msg.String() == "enter" || msg.String() == " " {
				if !m.loading {
					m.loading = true
					m.viewport.SetContent("Sending HTTP request...")
					cmds = append(cmds, m.spinner.Tick, m.sendHTTPRequest())
				}
				return m, tea.Batch(cmds...)
			}

		case focusResponse:
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

// --- View Rendering ---
func (m model) View() string {
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

	// 1. Leftmost: Clickable Tree View Sidebar
	sidebarContent := m.renderSidebar(sidebarWidth)
	sidebarBorder := panelStyle.Width(sidebarWidth).Height(panelHeight)
	if m.focus == focusSidebar {
		sidebarBorder = activePanelStyle.Width(sidebarWidth).Height(panelHeight)
	}
	sidebarPanel := sidebarBorder.Render(sidebarContent)

	// 2. Middle: Request Builder
	leftContent := m.renderRequestBuilder(halfWidth)
	leftBorder := panelStyle.Width(halfWidth).Height(panelHeight)
	if m.focus != focusResponse && m.focus != focusSidebar {
		leftBorder = activePanelStyle.Width(halfWidth).Height(panelHeight)
	}
	leftPanel := leftBorder.Render(leftContent)

	// 3. Rightmost: Response Viewer
	rightContent := m.renderResponseViewer(halfWidth)
	rightBorder := panelStyle.Width(halfWidth).Height(panelHeight)
	if m.focus == focusResponse {
		rightBorder = activePanelStyle.Width(halfWidth).Height(panelHeight)
	}
	rightPanel := rightBorder.Render(rightContent)

	// Combine 3-Column Split View
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, " ", leftPanel, " ", rightPanel)

	// Header banner
	header := titleStyle.Render("⚡ MARTIS TUI - Ultra-Light REST Client")

	// Footer Help
	footer := helpStyle.Render(
		"[Click/Enter] Toggle Folder/Load Request • [Tab] Focus • [Ctrl+E] Save Request • [Ctrl+S] Send • [q] Quit",
	)

	rendered := lipgloss.JoinVertical(lipgloss.Left, header, mainBody, footer)

	// Render Save Modal if active
	if m.saveModalOpen {
		return m.renderSaveModal(rendered)
	}

	return rendered
}

// Render Clickable Sidebar Tree
func (m model) renderSidebar(width int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("📂 COLLECTIONS"))
	lines = append(lines, lipgloss.NewStyle().Foreground(subtleColor).Render("Click or ↑/↓/Enter\n"))

	for i, row := range m.sidebarRows {
		prefix := "  "
		if i == m.selectedTreeIndex && m.focus == focusSidebar {
			prefix = "▶ "
		}

		if row.rowType == rowFolder {
			icon := "📁 ▶"
			if row.isExpanded {
				icon = "📂 ▼"
			}
			folderText := fmt.Sprintf("%s%s %s", prefix, icon, row.name)
			if i == m.selectedTreeIndex && m.focus == focusSidebar {
				lines = append(lines, activeRowStyle.Render(folderText))
			} else {
				lines = append(lines, folderStyle.Render(folderText))
			}
		} else {
			// Item Request di dalam Folder
			methodColor := lipgloss.Color("#00D7D7")
			if row.method == "POST" {
				methodColor = lipgloss.Color("#00E676")
			} else if row.method == "DELETE" {
				methodColor = lipgloss.Color("#FF5252")
			}
			badge := lipgloss.NewStyle().Foreground(methodColor).Bold(true).Render(row.method)
			itemText := fmt.Sprintf("%s  • %s %s", prefix, badge, row.name)

			if i == m.selectedTreeIndex && m.focus == focusSidebar {
				lines = append(lines, activeRowStyle.Render(itemText))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")).Render(itemText))
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Render Save Request Modal
func (m model) renderSaveModal(background string) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("💾 Save Request to Collection"),
		"",
		labelStyle.Render("Enter request name:"),
		m.saveNameInput.View(),
		"",
		lipgloss.NewStyle().Foreground(subtleColor).Render("[Enter] Save • [Esc] Cancel"),
	)

	modal := modalBoxStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m model) renderRequestBuilder(width int) string {
	var sections []string

	// Method Selector
	var methodBadges []string
	for i, meth := range m.methods {
		if i == m.methodIndex {
			if m.focus == focusMethod {
				methodBadges = append(methodBadges, activeMethodStyle.Render("▶ "+meth))
			} else {
				methodBadges = append(methodBadges, methodStyle.Render(meth))
			}
		} else {
			methodBadges = append(methodBadges, lipgloss.NewStyle().Foreground(subtleColor).Render(" "+meth+" "))
		}
	}
	methodRow := lipgloss.JoinHorizontal(lipgloss.Center, methodBadges...)

	// Send Button & Save
	sendBtn := sendBtnStyle.Render(" Send [Ctrl+S] ")
	if m.focus == focusSend {
		sendBtn = activeSendBtnStyle.Render("▶ Send [Ctrl+S] ")
	}
	if m.loading {
		sendBtn = lipgloss.NewStyle().Background(warningColor).Foreground(lipgloss.Color("#000000")).Render(
			fmt.Sprintf(" %s Sending... ", m.spinner.View()),
		)
	}

	saveBtn := lipgloss.NewStyle().Foreground(subtleColor).Render("[Ctrl+E] Save")
	topBar := lipgloss.JoinHorizontal(lipgloss.Center, methodRow, "  ", sendBtn, "  ", saveBtn)
	sections = append(sections, topBar)

	// URL Row
	urlPrompt := labelStyle.Render("URL: ")
	if m.focus == focusURL {
		urlPrompt = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("URL ▶ ")
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, urlPrompt, m.urlInput.View()))

	// Tab Selector Row
	tabLabels := []string{"1. Headers", "2. Body (JSON)", "3. Form-Data"}
	var renderedTabs []string
	for i, label := range tabLabels {
		if configTab(i) == m.tab {
			if m.focus == focusTabs {
				renderedTabs = append(renderedTabs, activeTabStyle.Copy().Background(accentColor).Foreground(lipgloss.Color("#000000")).Render("▶ "+label))
			} else {
				renderedTabs = append(renderedTabs, activeTabStyle.Render(label))
			}
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(label))
		}
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...))

	// Tab Content Body
	var configContent string
	switch m.tab {
	case tabHeaders:
		kPrefix := "  "
		vPrefix := "  "
		aPrefix := "  "
		if m.focus == focusConfig {
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
			labelStyle.Render("Custom Header (Key : Value):"),
			lipgloss.JoinHorizontal(lipgloss.Left, kPrefix, m.headerKey.View(), " : ", vPrefix, m.headerVal.View()),
			"",
			labelStyle.Render("Authorization Header:"),
			lipgloss.JoinHorizontal(lipgloss.Left, aPrefix, m.headerAuth.View()),
			"",
			lipgloss.NewStyle().Foreground(subtleColor).Render("Tip: Up/Down arrows to move between header fields."),
		)

	case tabBodyRaw:
		prefix := "Payload (Raw JSON):"
		if m.focus == focusConfig {
			prefix = "▶ Payload (Raw JSON): [Esc to unfocus textarea]"
		}
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			labelStyle.Render(prefix),
			m.jsonBody.View(),
		)

	case tabBodyForm:
		kPrefix := "  "
		fPrefix := "  "
		if m.focus == focusConfig {
			if m.formFocusIndex == 0 {
				kPrefix = "▶ "
			} else {
				fPrefix = "▶ "
			}
		}
		configContent = lipgloss.JoinVertical(
			lipgloss.Left,
			labelStyle.Render("Form-Data File Upload:"),
			labelStyle.Render("Field / Key Name:"),
			lipgloss.JoinHorizontal(lipgloss.Left, kPrefix, m.formKey.View()),
			"",
			labelStyle.Render("Local File Path:"),
			lipgloss.JoinHorizontal(lipgloss.Left, fPrefix, m.formFilePath.View()),
			"",
			lipgloss.NewStyle().Foreground(subtleColor).Render("Tip: Enter local file path to test multipart upload."),
		)
	}

	sections = append(sections, configContent)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) renderResponseViewer(width int) string {
	var statusBadge string
	var metaStats string

	if m.lastResp == nil {
		statusBadge = metaBadge.Render("STATUS: IDLE")
		metaStats = metaBadge.Render("Time: 0ms • Size: 0B")
	} else if m.lastResp.err != nil {
		statusBadge = statusErr.Render("ERR: FAILED")
		metaStats = metaBadge.Render(fmt.Sprintf("Time: %dms", m.lastResp.duration.Milliseconds()))
	} else {
		code := m.lastResp.statusCode
		statusStr := fmt.Sprintf("%d %s", code, http.StatusText(code))
		if code >= 200 && code < 300 {
			statusBadge = status2xx.Render(statusStr)
		} else if code >= 300 && code < 400 {
			statusBadge = status3xx.Render(statusStr)
		} else {
			statusBadge = statusErr.Render(statusStr)
		}

		bodySize := len(m.lastResp.body)
		metaStats = metaBadge.Render(
			fmt.Sprintf("Time: %dms • Size: %s", m.lastResp.duration.Milliseconds(), formatBytes(bodySize)),
		)
	}

	statusBar := lipgloss.JoinHorizontal(lipgloss.Center, statusBadge, "  ", metaStats)
	var focusIndicator string
	if m.focus == focusResponse {
		focusIndicator = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(" [VIEWPORT ACTIVE: Scroll with ↑/↓/j/k]")
	} else {
		focusIndicator = lipgloss.NewStyle().Foreground(subtleColor).Render(" [Tab into Viewport to scroll]")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		focusIndicator,
		"",
		m.viewport.View(),
	)
}

func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

var version = "dev"

func handleCLIArgs() bool {
	if len(os.Args) < 2 {
		return false
	}

	cmd := os.Args[1]
	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("martis %s\n", version)
		return true

	case "update", "--update":
		fmt.Println("⚡ Memeriksa dan memperbarui Martis dari upstream...")
		if _, err := os.Stat(".git"); err == nil {
			fmt.Println("Repo git terdeteksi. Menjalankan make update-upstream...")
			c := "make update-upstream"
			fmt.Printf("Menjalankan: %s\n", c)
		} else {
			fmt.Println("Untuk update langsung tanpa repositori lokal, gunakan:")
			fmt.Println("  go install github.com/wahyuakbarwibowo/martis@latest")
		}
		return true

	case "collections", "col":
		col := loadCollections()
		fmt.Printf("📁 Collections: %s (%d folders)\n\n", col.Name, len(col.Folders))
		for _, f := range col.Folders {
			fmt.Printf("📂 %s (%d requests)\n", f.Name, len(f.Items))
			for i, it := range f.Items {
				fmt.Printf("   %d. [%-6s] %-25s -> %s\n", i+1, it.Method, it.Name, it.URL)
			}
			fmt.Println()
		}
		return true

	case "help", "-h", "--help":
		fmt.Printf("Martis TUI - Ultra-Light REST Client (%s)\n\n", version)
		fmt.Println("Penggunaan:")
		fmt.Println("  martis             Buka Terminal User Interface (dengan Mouse Click Tree View)")
		fmt.Println("  martis collections Tampilkan daftar request di collection")
		fmt.Println("  martis version     Tampilkan versi aplikasi")
		fmt.Println("  martis update      Perbarui aplikasi dari upstream")
		fmt.Println("  martis help        Tampilkan bantuan ini")
		return true
	}

	return false
}

// --- Main Program ---
func main() {
	if handleCLIArgs() {
		return
	}

	// tea.WithMouseCellMotion() mengaktifkan event klik mouse pada terminal!
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Martis TUI: %v\n", err)
		os.Exit(1)
	}
}
