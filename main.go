package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type step int

const (
	stepSelectLocalServices step = iota
	stepSelectBuildMethod
	stepInputBranch
	stepExecuting
	stepDone
)

type buildMethod int

const (
	buildHarbor buildMethod = iota
	buildLocal
)

type model struct {
	step          step
	servicesAll   []string
	servicesLocal map[string]bool
	buildMethod   buildMethod
	branchInput   textinput.Model
	hostsLine     string
	tag           string
	workDir       string
	currentTask   string
	spinner       spinner.Model
	err           error
	cursor        int
	width         int
	height        int
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(1, 0).
			MarginBottom(1)

	checkboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	buttonStyle   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3B82F6")).
			Padding(0, 3).
			Margin(1, 0)
	buttonSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#10B981")).
				Padding(0, 3).
				Margin(1, 0)
	buttonActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 3).
				Margin(1, 0)
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Margin(1, 0)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	hostsStyle   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#374151")).
			Padding(0, 2).
			MarginTop(1)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	taskStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
)

func getWorkDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return dir, nil
}

func getDockerComposeServices(workDir string) ([]string, error) {
	cmd := exec.Command("docker", "compose", "config", "--services")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker compose services: %w", err)
	}
	services := strings.Split(strings.TrimSpace(string(output)), "\n")
	sort.Strings(services)
	return services, nil
}

func initialModel() model {
	workDir, err := getWorkDir()
	if err != nil {
		return model{err: err}
	}

	services, err := getDockerComposeServices(workDir)
	if err != nil {
		return model{err: err}
	}

	ti := textinput.New()
	ti.Placeholder = "main"
	ti.SetValue("main")
	ti.Focus()

	servicesLocal := make(map[string]bool)
	for _, s := range services {
		servicesLocal[s] = false
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return model{
		step:          stepSelectLocalServices,
		servicesAll:   services,
		servicesLocal: servicesLocal,
		branchInput:   ti,
		cursor:        0,
		workDir:       workDir,
		spinner:       s,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
		return m, nil
	}

	var spinnerCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case execResult:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tag = msg.tag
		m.hostsLine = msg.hostsLine
		m.step = stepDone
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.step == stepSelectLocalServices {
				if m.cursor > 0 {
					m.cursor--
				}
			} else if m.step == stepSelectBuildMethod {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.step == stepSelectLocalServices {
				if m.cursor < len(m.servicesAll)-1 {
					m.cursor++
				}
			} else if m.step == stepSelectBuildMethod {
				if m.cursor < 1 {
					m.cursor++
				}
			}

		case " ":
			if m.step == stepSelectLocalServices {
				service := m.servicesAll[m.cursor]
				m.servicesLocal[service] = !m.servicesLocal[service]
			}

		case "enter":
			switch m.step {
			case stepSelectLocalServices:
				m.step = stepSelectBuildMethod
				m.cursor = 0
			case stepSelectBuildMethod:
				if m.cursor == 0 {
					m.buildMethod = buildHarbor
				} else {
					m.buildMethod = buildLocal
				}
				if m.buildMethod == buildHarbor {
					m.step = stepInputBranch
				} else {
					m.step = stepExecuting
					m.currentTask = "Получение хэша коммита..."
					return m, tea.Batch(m.spinner.Tick, m.execute())
				}
			case stepInputBranch:
				m.step = stepExecuting
				m.currentTask = "Получение хэша коммита..."
				return m, tea.Batch(m.spinner.Tick, m.execute())
			case stepDone:
				return m, tea.Quit
			}
		}
	}

	if m.step == stepInputBranch {
		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		return m, cmd
	}

	if m.step == stepExecuting {
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd
	}

	return m, nil
}

type execResult struct {
	err       error
	tag       string
	hostsLine string
}

type taskMsg struct {
	task string
}

func (m *model) setTask(task string) tea.Cmd {
	m.currentTask = task
	return func() tea.Msg {
		return taskMsg{task: task}
	}
}

func (m model) execute() tea.Cmd {
	return func() tea.Msg {
		servicesTag := ""

		// Get servicesTag
		if m.buildMethod == buildHarbor {
			branch := m.branchInput.Value()
			cmd := exec.Command("git", "fetch", "origin", branch)
			cmd.Dir = m.workDir
			if output, err := cmd.CombinedOutput(); err != nil {
				return execResult{err: fmt.Errorf("git fetch failed: %w\n%s", err, string(output))}
			}

			cmd = exec.Command("git", "rev-parse", "--short=8", "origin/"+branch)
			cmd.Dir = m.workDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				return execResult{err: fmt.Errorf("git rev-parse failed: %w\noutput: %s", err, string(output))}
			}
			servicesTag = strings.TrimSpace(string(output))
			if servicesTag == "" {
				return execResult{err: fmt.Errorf("git rev-parse returned empty tag for branch '%s', raw output: '%s'", branch, string(output))}
			}
		} else {
			cmd := exec.Command("git", "rev-parse", "--short=8", "HEAD")
			cmd.Dir = m.workDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				return execResult{err: fmt.Errorf("git rev-parse failed: %w\noutput: %s", err, string(output))}
			}
			servicesTag = strings.TrimSpace(string(output))
			if servicesTag == "" {
				return execResult{err: fmt.Errorf("git rev-parse returned empty tag, raw output: '%s'", string(output))}
			}
		}

		// Small delay to show task
		time.Sleep(300 * time.Millisecond)

		// Generate tmp.compose.json with TAG environment variable
		cmd := exec.Command("docker", "compose", "config", "--format", "json", "-o", "tmp.compose.json")
		cmd.Dir = m.workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("TAG=%s", servicesTag))
		if output, err := cmd.CombinedOutput(); err != nil {
			return execResult{err: fmt.Errorf("docker compose config failed: %w\n%s", err, string(output))}
		}

		// Create tmp.env
		envContent := fmt.Sprintf("TAG=%s\n", servicesTag)
		envPath := filepath.Join(m.workDir, "tmp.env")
		if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
			return execResult{err: fmt.Errorf("failed to create tmp.env: %w", err)}
		}

		// Modify tmp.compose.json
		if err := m.modifyComposeFile(); err != nil {
			return execResult{err: err}
		}

		// Run docker compose up
		if err := m.dockerComposeUp(); err != nil {
			return execResult{err: err}
		}

		// Generate hosts line for user
		servicesDocker := m.getServicesDocker()
		hostsLine := ""
		if len(servicesDocker) > 0 {
			hostsLine = fmt.Sprintf("127.0.0.1 %s", strings.Join(servicesDocker, " "))
		}

		return execResult{err: nil, tag: servicesTag, hostsLine: hostsLine}
	}
}

func (m model) dockerComposeUp() error {
	servicesDocker := m.getServicesDocker()

	var cmd *exec.Cmd
	if m.buildMethod == buildHarbor {
		args := []string{"compose", "-f", "tmp.compose.json", "--env-file", "tmp.env", "up", "-d", "--pull", "always", "--no-build"}
		args = append(args, servicesDocker...)
		cmd = exec.Command("docker", args...)
	} else {
		args := []string{"compose", "-f", "tmp.compose.json", "--env-file", "tmp.env", "up", "-d", "--build"}
		args = append(args, servicesDocker...)
		cmd = exec.Command("docker", args...)
	}
	cmd.Dir = m.workDir

	// Show output in real-time
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (m model) getServicesDocker() []string {
	var result []string
	for _, s := range m.servicesAll {
		if !m.servicesLocal[s] {
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

func (m model) getServicesLocal() []string {
	var result []string
	for _, s := range m.servicesAll {
		if m.servicesLocal[s] {
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

func (m model) modifyComposeFile() error {
	composePath := filepath.Join(m.workDir, "tmp.compose.json")
	data, err := os.ReadFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to read tmp.compose.json: %w", err)
	}

	var compose map[string]interface{}
	if err = json.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("failed to parse tmp.compose.json: %w", err)
	}

	servicesDocker := m.getServicesDocker()
	servicesLocal := m.getServicesLocal()

	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid compose file: missing services")
	}

	for _, svcName := range servicesDocker {
		svc, ok := services[svcName].(map[string]interface{})
		if !ok {
			continue
		}

		// Build extra_hosts for local services
		extraHosts := []string{}
		for _, localSvc := range servicesLocal {
			extraHosts = append(extraHosts, fmt.Sprintf("%s:host-gateway", localSvc))
		}

		if len(extraHosts) > 0 {
			svc["extra_hosts"] = extraHosts
		}
	}

	updatedData, err := json.MarshalIndent(compose, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}

	if err = os.WriteFile(composePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write tmp.compose.json: %w", err)
	}

	return nil
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Ошибка: %v", m.err)) + "\nНажмите любую клавишу для выхода."
	}

	var b strings.Builder

	switch m.step {
	case stepSelectLocalServices:
		b.WriteString(titleStyle.Render("MDGA - Настройка локальной разработки"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Директория: %s\n\n", m.workDir))
		b.WriteString("Какие из перечисленных сервисов НЕ нужно запускать в докере?\n")
		b.WriteString("(Используйте ↑/↓ для навигации, пробел для выбора, Enter для подтверждения)\n\n")

		for i, service := range m.servicesAll {
			checkbox := "[ ]"
			if m.servicesLocal[service] {
				checkbox = "[✓]"
			}
			if m.cursor == i {
				b.WriteString(selectedStyle.Render("> "))
				b.WriteString(checkboxStyle.Render(checkbox + " "))
				b.WriteString(selectedStyle.Render(service))
				b.WriteString("\n")
			} else {
				b.WriteString("  ")
				b.WriteString(checkboxStyle.Render(checkbox + " "))
				b.WriteString(service)
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(buttonActiveStyle.Render("Далее"))

	case stepSelectBuildMethod:
		b.WriteString(titleStyle.Render("MDGA - Настройка локальной разработки"))
		b.WriteString("\n")
		b.WriteString("Использовать готовые докер образы из Harbor, или запустить локальную сборку докер образов?\n\n")

		options := []string{
			"Взять образы из Harbor",
			"Собрать образы локально",
		}

		for i, opt := range options {
			if m.cursor == i {
				b.WriteString(buttonSelectedStyle.Render(opt))
			} else {
				b.WriteString(buttonStyle.Render(opt))
			}
			b.WriteString("\n")
		}

	case stepInputBranch:
		b.WriteString(titleStyle.Render("MDGA - Настройка локальной разработки"))
		b.WriteString("\n")
		b.WriteString(messageStyle.Render("Будет взят последний собранный образ из указанной ветки"))
		b.WriteString("\n\n")
		b.WriteString("Введите название ветки:\n")
		b.WriteString(m.branchInput.View())
		b.WriteString("\n\n")
		b.WriteString(buttonActiveStyle.Render("Далее"))

	case stepExecuting:
		b.WriteString(titleStyle.Render("MDGA - Настройка локальной разработки"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%s %s\n\n", m.spinner.View(), m.currentTask))
		b.WriteString(infoStyle.Render("Подготовка и запуск контейнеров...\n"))
		b.WriteString(infoStyle.Render("Вывод docker compose будет показан в терминале.\n"))

	case stepDone:
		b.WriteString(titleStyle.Render("MDGA - Настройка локальной разработки"))
		b.WriteString("\n")
		b.WriteString(successStyle.Render("✓ Docker контейнеры успешно запущены!"))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("TAG: %s", m.tag)))
		b.WriteString("\n\n")

		b.WriteString(infoStyle.Render("Сервисы:"))
		b.WriteString("\n")
		for _, svc := range m.servicesAll {
			if m.servicesLocal[svc] {
				b.WriteString(fmt.Sprintf("  %s %s (локально)\n", infoStyle.Render("○"), svc))
			} else {
				b.WriteString(fmt.Sprintf("  %s %s (в Docker)\n", successStyle.Render("●"), svc))
			}
		}
		b.WriteString("\n")

		if m.hostsLine != "" {
			b.WriteString(infoStyle.Render("Добавьте следующую строку в /etc/hosts:"))
			b.WriteString("\n")
			b.WriteString(hostsStyle.Render(m.hostsLine))
			b.WriteString("\n\n")
		}

		b.WriteString(buttonActiveStyle.Render("Выход"))
	}

	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}
