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
	focusMethod focusArea = iota
	focusURL
	focusTabs
	focusConfig
	focusSend
	focusResponse
	focusCollection // Modal / Sidebar collection list
)

const totalMainFocusAreas = 6

type configTab int

const (
	tabHeaders configTab = iota
	tabBodyRaw
	tabBodyForm
)

const totalTabs = 3

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

	// Collection Management
	collection          Collection
	selectedColIndex    int
	showCollectionModal bool
	saveModalOpen       bool
	saveNameInput       textinput.Model

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
	urlIn.Width = 40

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
	sName.Placeholder = "Request Name (e.g. Get User List)"
	sName.CharLimit = 100

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Load Collections from Disk
	col := loadCollections()

	m := model{
		focus:         focusURL,
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

	m.updateFocusStates()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// --- Focus Management Helper ---
func (m *model) updateFocusStates() {
	// Blur all first
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
	// Set Method
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

// Simpan request aktif ke collection
func (m *model) saveCurrentToCollection(name string) {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s", m.methods[m.methodIndex], m.urlInput.Value())
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

	m.collection.Items = append(m.collection.Items, newItem)
	_ = saveCollections(m.collection)
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

	// Headers
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
				// Form-Data / File upload mode
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

		// Apply Headers
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

		// Pretty print JSON if applicable
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

		halfWidth := (m.width / 2) - 4
		if halfWidth < 30 {
			halfWidth = 30
		}
		vpHeight := m.height - 10
		if vpHeight < 5 {
			vpHeight = 5
		}

		if !m.viewportReady {
			m.viewport = viewport.New(halfWidth, vpHeight)
			m.viewport.SetContent("Waiting for request... Press [Ctrl+S] or Send button to execute.")
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

		// Modal Collection List handling
		if m.showCollectionModal {
			switch msg.String() {
			case "esc", "c", "C":
				m.showCollectionModal = false
				m.updateFocusStates()
				return m, nil
			case "up", "k":
				if m.selectedColIndex > 0 {
					m.selectedColIndex--
				}
				return m, nil
			case "down", "j":
				if m.selectedColIndex < len(m.collection.Items)-1 {
					m.selectedColIndex++
				}
				return m, nil
			case "enter":
				if len(m.collection.Items) > 0 {
					m.loadCollectionItem(m.collection.Items[m.selectedColIndex])
				}
				m.showCollectionModal = false
				m.updateFocusStates()
				return m, nil
			case "d", "backspace": // Hapus item collection
				if len(m.collection.Items) > 0 {
					idx := m.selectedColIndex
					m.collection.Items = append(m.collection.Items[:idx], m.collection.Items[idx+1:]...)
					if m.selectedColIndex >= len(m.collection.Items) && m.selectedColIndex > 0 {
						m.selectedColIndex--
					}
					_ = saveCollections(m.collection)
				}
				return m, nil
			}
			return m, nil
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

		case "ctrl+p": // Open Collection Picker
			m.showCollectionModal = true
			return m, nil

		case "ctrl+e": // Save request to collection
			m.saveModalOpen = true
			m.saveNameInput.SetValue(fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value())))
			m.updateFocusStates()
			return m, nil

		case "tab":
			m.focus = (m.focus + 1) % totalMainFocusAreas
			m.updateFocusStates()
			return m, nil

		case "shift+tab":
			m.focus = (m.focus - 1 + totalMainFocusAreas) % totalMainFocusAreas
			m.updateFocusStates()
			return m, nil

		case "ctrl+t": // Cycle config tabs quickly
			m.tab = (m.tab + 1) % totalTabs
			m.updateFocusStates()
			return m, nil
		}

		// Handle key per active focus area
		switch m.focus {
		case focusMethod:
			switch msg.String() {
			case "left", "h", "up", "k":
				if m.methodIndex > 0 {
					m.methodIndex--
				}
			case "right", "l", "down", "j":
				if m.methodIndex < len(m.methods)-1 {
					m.methodIndex++
				}
			case "enter":
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

	halfWidth := (m.width / 2) - 4
	if halfWidth < 30 {
		halfWidth = 30
	}
	panelHeight := m.height - 6
	if panelHeight < 10 {
		panelHeight = 10
	}

	// 1. Left Side: Request Builder
	leftContent := m.renderRequestBuilder(halfWidth)
	leftBorder := panelStyle.Width(halfWidth).Height(panelHeight)
	if m.focus != focusResponse {
		leftBorder = activePanelStyle.Width(halfWidth).Height(panelHeight)
	}
	leftPanel := leftBorder.Render(leftContent)

	// 2. Right Side: Response Viewer
	rightContent := m.renderResponseViewer(halfWidth)
	rightBorder := panelStyle.Width(halfWidth).Height(panelHeight)
	if m.focus == focusResponse {
		rightBorder = activePanelStyle.Width(halfWidth).Height(panelHeight)
	}
	rightPanel := rightBorder.Render(rightContent)

	// Combine Split View
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)

	// Header banner
	header := titleStyle.Render("⚡ MARTIS TUI - Ultra-Light REST Client")

	// Footer Help
	footer := helpStyle.Render(
		"[Tab] Focus • [Ctrl+P] Collections • [Ctrl+E] Save Request • [Ctrl+S] Send • [q] Quit",
	)

	rendered := lipgloss.JoinVertical(lipgloss.Left, header, mainBody, footer)

	// Render Modals if active
	if m.showCollectionModal {
		return m.renderCollectionModal(rendered)
	}
	if m.saveModalOpen {
		return m.renderSaveModal(rendered)
	}

	return rendered
}

// Render Collection Picker Modal
func (m model) renderCollectionModal(background string) string {
	var items []string
	items = append(items, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E676")).Render("📁 Collections / Saved Requests"))
	items = append(items, lipgloss.NewStyle().Foreground(subtleColor).Render("Use [↑/↓/j/k] to navigate, [Enter] to load, [d] to delete, [Esc] to close\n"))

	if len(m.collection.Items) == 0 {
		items = append(items, lipgloss.NewStyle().Foreground(subtleColor).Render("No saved requests yet. Press [Ctrl+E] to save current request."))
	} else {
		for i, it := range m.collection.Items {
			prefix := "  "
			cursor := lipgloss.NewStyle().Foreground(primaryColor)
			methodBadge := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("[%-6s]", it.Method))

			if i == m.selectedColIndex {
				prefix = "▶ "
				cursor = cursor.Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(primaryColor)
			}

			line := fmt.Sprintf("%s %s %-25s %s", prefix, methodBadge, it.Name, lipgloss.NewStyle().Foreground(subtleColor).Render(it.URL))
			items = append(items, cursor.Render(line))
		}
	}

	modalContent := modalBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, items...))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalContent)
}

// Render Save Request Modal
func (m model) renderSaveModal(background string) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("💾 Save Request to Collection"),
		"",
		labelStyle.Render("Enter a name for this request:"),
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

	// Send Button & Collection Quick Action
	sendBtn := sendBtnStyle.Render(" Send [Ctrl+S] ")
	if m.focus == focusSend {
		sendBtn = activeSendBtnStyle.Render("▶ Send [Ctrl+S] ")
	}
	if m.loading {
		sendBtn = lipgloss.NewStyle().Background(warningColor).Foreground(lipgloss.Color("#000000")).Render(
			fmt.Sprintf(" %s Sending... ", m.spinner.View()),
		)
	}

	saveBtn := lipgloss.NewStyle().Foreground(subtleColor).Render("[Ctrl+P] Load • [Ctrl+E] Save")
	topBar := lipgloss.JoinHorizontal(lipgloss.Center, methodRow, "  ", sendBtn, "  ", saveBtn)
	sections = append(sections, topBar)

	// URL Row
	urlPrompt := labelStyle.Render("URL: ")
	if m.focus == focusURL {
		urlPrompt = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("URL ▶ ")
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left, urlPrompt, m.urlInput.View()))

	// Tab Selector Row
	tabLabels := []string{"1. Headers", "2. Body (JSON)", "3. Form-Data / Upload"}
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
			lipgloss.NewStyle().Foreground(subtleColor).Render("Tip: Use Up/Down arrows to move between header fields."),
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
			lipgloss.NewStyle().Foreground(subtleColor).Render("Tip: Enter local file path to test multipart/form-data upload."),
		)
	}

	sections = append(sections, configContent)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) renderResponseViewer(width int) string {
	// Status & Metrics Bar
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

// Format bytes into readable format
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

// Global version variable injected via -ldflags="-X main.version=..."
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
		fmt.Printf("📁 Collections: %s (%d items)\n\n", col.Name, len(col.Items))
		for i, it := range col.Items {
			fmt.Printf("  %d. [%-6s] %-25s -> %s\n", i+1, it.Method, it.Name, it.URL)
		}
		return true

	case "help", "-h", "--help":
		fmt.Printf("Martis TUI - Ultra-Light REST Client (%s)\n\n", version)
		fmt.Println("Penggunaan:")
		fmt.Println("  martis             Buka Terminal User Interface")
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

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Martis TUI: %v\n", err)
		os.Exit(1)
	}
}
