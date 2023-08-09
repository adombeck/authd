package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ubuntu/authd"
	"github.com/ubuntu/authd/internal/log"
)

type authModeSelectionModel struct {
	list.Model
	focused bool

	client authd.PAMClient

	availableAuthModes        []*authd.GAMResponse_AuthenticationMode
	currentAuthModeSelectedID string
}

type authModesReceived struct {
	authModes []*authd.GAMResponse_AuthenticationMode
}

type authModeSelected struct {
	id string
}

func (m *authModeSelectionModel) selectAuthMode(id string) tea.Cmd {
	m.currentAuthModeSelectedID = id
	return func() tea.Msg {
		return authModeSelected{
			id: id,
		}
	}
}

func newAuthModeSelectionModel(client authd.PAMClient) authModeSelectionModel {
	l := list.New(nil, itemLayout{}, 80, 24)
	l.Title = "Select your authentication method"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	l.Styles.Title = lipgloss.NewStyle()
	/*l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle*/

	return authModeSelectionModel{
		Model:  l,
		client: client,
	}
}

func (m authModeSelectionModel) Init() tea.Cmd {
	return nil
}

func (m authModeSelectionModel) Update(msg tea.Msg) (authModeSelectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case authModesReceived:
		m.availableAuthModes = msg.authModes

		var allAuthModes []list.Item
		for _, a := range m.availableAuthModes {
			allAuthModes = append(allAuthModes, authModeItem{
				id:    a.Id,
				label: a.Label,
			})
		}
		return m, m.SetItems(allAuthModes)

	case authModeSelected:
		// Ensure auth mode id is valid
		if !validAuthModeID(msg.id, m.availableAuthModes) {
			log.Infof(context.TODO(), "authentication mode %q is not part of currently available authentication mode", msg.id)
			return m, nil
		}
		// Select correct line to ensure model is synchronised
		for i, a := range m.Items() {
			if a.(authModeItem).id != msg.id {
				continue
			}
			m.Select(i)
		}

		return m, sendEvent(AuthModeSelected{
			ID: msg.id,
		})
	}

	// interaction events
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	// Key presses
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			item := m.SelectedItem()
			if item == nil {
				return m, nil
			}
			authMode := item.(authModeItem)
			cmd := m.selectAuthMode(authMode.id)
			return m, cmd
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// This is necessarily an integer, so above
			choice, _ := strconv.Atoi(msg.String())
			items := m.Items()
			if choice > len(items) {
				return m, nil
			}
			item := items[choice-1]
			authMode := item.(authModeItem)
			cmd := m.selectAuthMode(authMode.id)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m *authModeSelectionModel) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *authModeSelectionModel) Focused() bool {
	return m.focused
}

func (m *authModeSelectionModel) Blur() {
	m.focused = false
}

func (m authModeSelectionModel) WillCaptureEscape() bool {
	return m.FilterState() == list.Filtering
}

type authModeItem struct {
	id    string
	label string
}

func (i authModeItem) FilterValue() string { return "" }

// validAuthModeID returns if a authmode ID exists in the available list.
func validAuthModeID(id string, authModes []*authd.GAMResponse_AuthenticationMode) bool {
	for _, a := range authModes {
		if a.Id != id {
			continue
		}
		return true
	}
	return false
}

// getAuthenticationModes returns available authentication mode for this broker from authd.
func getAuthenticationModes(client authd.PAMClient, sessionID string) tea.Cmd {
	return func() tea.Msg {
		required, optional := "required", "optional"
		supportedEntries := "optional:chars,chars_password"
		requiredWithBooleans := "required:true,false"
		optionalWithBooleans := "optional:true,false"

		gamReq := &authd.GAMRequest{
			SessionId: sessionID,
			SupportedUiLayouts: []*authd.UILayout{
				{
					Type:   "form",
					Label:  &required,
					Entry:  &supportedEntries,
					Wait:   &optionalWithBooleans,
					Button: &optional,
				},
				{
					Type:    "qrcode",
					Content: &required,
					Wait:    &requiredWithBooleans,
					Label:   &optional,
				},
			},
		}

		gamResp, err := client.GetAuthenticationModes(context.Background(), gamReq)
		if err != nil {
			return pamSystemError{
				msg: fmt.Sprintf("could not get authentication modes: %v", err),
			}
		}

		authModes := gamResp.GetAuthenticationModes()
		if len(authModes) == 0 {
			return pamIgnore{
				// TODO: probably go back to broker selection here
				msg: "no supported authentication mode available for this provider",
			}
		}
		log.Info(context.TODO(), authModes)

		return authModesReceived{
			authModes: authModes,
		}
	}
}
func (m *authModeSelectionModel) Reset() {
	m.currentAuthModeSelectedID = ""
}
