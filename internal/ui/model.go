package ui

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"martis/internal/domain"
	"martis/internal/httpclient"
	"martis/internal/repository"
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
)

const totalTabs = 3

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

	methods      []string
	methodIndex  int
	urlInput     textinput.Model
	headerKey    textinput.Model
	headerVal    textinput.Model
	headerAuth   textinput.Model
	jsonBody     textarea.Model
	formKey      textinput.Model
	formFilePath textinput.Model

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
	m.saveNameInput.Blur()

	if m.saveModalOpen {
		m.saveNameInput.Focus()
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

	if item.BodyType == "form" {
		m.tab = TabBodyForm
		m.formKey.SetValue(item.FormKey)
		m.formFilePath.SetValue(item.FormPath)
	} else {
		m.tab = TabBodyRaw
		m.jsonBody.SetValue(item.BodyRaw)
	}
	m.updateFocusStates()
}

func (m *Model) toggleFolder(fIdx int) {
	if m.collection != nil && fIdx >= 0 && fIdx < len(m.collection.Folders) {
		m.collection.Folders[fIdx].IsExpanded = !m.collection.Folders[fIdx].IsExpanded
		_ = m.repo.Save(m.collection)
		m.rebuildSidebarRows()
	}
}

func (m *Model) saveCurrentToCollection(name string) {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s %s", m.methods[m.methodIndex], filepath.Base(m.urlInput.Value()))
	}

	bodyType := "raw"
	if m.tab == TabBodyForm {
		bodyType = "form"
	}

	newItem := domain.CollectionItem{
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
	_ = m.repo.Save(m.collection)
	m.rebuildSidebarRows()
}

func (m Model) executeRequestCmd() tea.Cmd {
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

	return func() tea.Msg {
		return m.httpClient.Do(payload)
	}
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
				clickedRow := msg.Y - 2
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
		if msg.Err != nil {
			m.viewport.SetContent(fmt.Sprintf("❌ Request Error:\n\n%v\n\nDuration: %v", msg.Err, msg.Duration))
		} else {
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

	case tea.KeyMsg:
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

		case "ctrl+t":
			m.tab = (m.tab + 1) % totalTabs
			m.updateFocusStates()
			return m, nil
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
		"[Click/Enter] Toggle Folder/Load Request • [Tab] Focus • [Ctrl+E] Save Request • [Ctrl+S] Send • [q] Quit",
	)

	rendered := lipgloss.JoinVertical(lipgloss.Left, header, mainBody, footer)
	if m.saveModalOpen {
		return m.renderSaveModal(rendered)
	}
	return rendered
}

func (m Model) renderSidebar(width int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor).Render("📂 COLLECTIONS"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.SubtleColor).Render("Click or ↑/↓/Enter\n"))

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
			folderText := fmt.Sprintf("%s%s %s", prefix, icon, row.name)
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
			itemText := fmt.Sprintf("%s  • %s %s", prefix, badge, row.name)

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

	tabLabels := []string{"1. Headers", "2. Body (JSON)", "3. Form-Data"}
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

	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		focusIndicator,
		"",
		m.viewport.View(),
	)
}
