package ui

import (
	"errors"
	"fmt"
	"grout/constants"
	"grout/utils"
	"os"
	"strconv"
	"strings"
	"time"

	"grout/romm"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
)

type loginInput struct {
	ExistingHost romm.Host
}

type loginOutput struct {
	Host   romm.Host
	Config *utils.Config
}

type loginAttemptResult struct {
	ErrorType string
	ErrorMsg  string
	Success   bool
}

type LoginScreen struct{}

func newLoginScreen() *LoginScreen {
	return &LoginScreen{}
}

func (s *LoginScreen) draw(input loginInput) (ScreenResult[loginOutput], error) {
	host := input.ExistingHost

	items := []gabagool.ItemWithOptions{
		{
			Item: gabagool.MenuItem{
				Text: i18n.GetString("login_protocol"),
			},
			Options: []gabagool.Option{
				{DisplayName: i18n.GetString("login_protocol_http"), Value: "http://"},
				{DisplayName: i18n.GetString("login_protocol_https"), Value: "https://"},
			},
			SelectedOption: func() int {
				if strings.Contains(host.RootURI, "https") {
					return 1
				}
				return 0
			}(),
		},
		{
			Item: gabagool.MenuItem{
				Text: i18n.GetString("login_hostname"),
			},
			Options: []gabagool.Option{
				{
					Type:           gabagool.OptionTypeKeyboard,
					KeyboardLayout: gabagool.KeyboardLayoutURL,
					URLShortcuts: []gabagool.URLShortcut{
						{Value: "romm.", SymbolValue: "romm."},
						{Value: ".com", SymbolValue: ".com"},
						{Value: ".org", SymbolValue: ".org"},
						{Value: ".net", SymbolValue: ".net"},
						{Value: ".local", SymbolValue: ".ts.net"},
					},
					DisplayName:    removeScheme(host.RootURI),
					KeyboardPrompt: removeScheme(host.RootURI),
					Value:          removeScheme(host.RootURI),
				},
			},
		},
		{
			Item: gabagool.MenuItem{
				Text: i18n.GetString("login_port"),
			},
			Options: []gabagool.Option{
				{
					Type:           gabagool.OptionTypeKeyboard,
					KeyboardLayout: gabagool.KeyboardLayoutNumeric,
					KeyboardPrompt: func() string {
						if host.Port == 0 {
							return ""
						}
						return strconv.Itoa(host.Port)
					}(),
					DisplayName: func() string {
						if host.Port == 0 {
							return ""
						}
						return strconv.Itoa(host.Port)
					}(),
					Value: func() string {
						if host.Port == 0 {
							return ""
						}
						return strconv.Itoa(host.Port)
					}(),
				},
			},
		},
		{
			Item: gabagool.MenuItem{
				Text: i18n.GetString("login_username"),
			},
			Options: []gabagool.Option{
				{
					Type:           gabagool.OptionTypeKeyboard,
					DisplayName:    host.Username,
					KeyboardPrompt: host.Username,
					Value:          host.Username,
				},
			},
		},
		{
			Item: gabagool.MenuItem{
				Text: i18n.GetString("login_password"),
			},
			Options: []gabagool.Option{
				{
					Type:           gabagool.OptionTypeKeyboard,
					Masked:         true,
					DisplayName:    host.Password,
					KeyboardPrompt: host.Password,
					Value:          host.Password,
				},
			},
		},
	}

	res, err := gabagool.OptionsList(
		i18n.GetString("login_title"),
		gabagool.OptionListSettings{
			DisableBackButton: false,
			FooterHelpItems: []gabagool.FooterHelpItem{
				{ButtonName: "B", HelpText: i18n.GetString("button_quit")},
				{ButtonName: "←→", HelpText: i18n.GetString("button_cycle")},
				{ButtonName: "Start", HelpText: i18n.GetString("button_login")},
			},
		},
		items,
	)

	if err != nil {
		return withCode(loginOutput{}, gabagool.ExitCodeCancel), nil
	}

	loginSettings := res.Items

	newHost := romm.Host{
		RootURI: fmt.Sprintf("%s%s", loginSettings[0].Value(), loginSettings[1].Value()),
		Port: func(s string) int {
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
			return 0
		}(loginSettings[2].Value().(string)),
		Username: loginSettings[3].Options[0].Value.(string),
		Password: loginSettings[4].Options[0].Value.(string),
	}

	return success(loginOutput{Host: newHost}), nil
}

func LoginFlow(existingHost romm.Host) (*utils.Config, error) {
	screen := newLoginScreen()

	for {
		result, err := screen.draw(loginInput{ExistingHost: existingHost})
		if err != nil {
			gabagool.ProcessMessage(i18n.GetString("login_error_unexpected"), gabagool.ProcessMessageOptions{}, func() (interface{}, error) {
				time.Sleep(3 * time.Second)
				return nil, nil
			})
			return nil, fmt.Errorf("unable to get login information: %w", err)
		}

		if result.ExitCode == gabagool.ExitCodeBack || result.ExitCode == gabagool.ExitCodeCancel {
			os.Exit(1)
		}

		host := result.Value.Host

		loginResult := attemptLogin(host)

		if loginResult.Success {
			config := &utils.Config{
				Hosts: []romm.Host{host},
			}
			return config, nil
		}

		gabagool.ConfirmationMessage(
			i18n.GetString(loginResult.ErrorMsg),
			[]gabagool.FooterHelpItem{
				{ButtonName: "A", HelpText: i18n.GetString("button_continue")},
			},
			gabagool.MessageOptions{},
		)
		existingHost = host
	}
}

func attemptLogin(host romm.Host) loginAttemptResult {
	// Phase 1: Quick validation with short timeout
	validationClient := utils.GetRommClient(host, constants.ValidationTimeout)

	result, _ := gabagool.ProcessMessage(
		i18n.GetString("login_validating"),
		gabagool.ProcessMessageOptions{},
		func() (interface{}, error) {
			err := validationClient.ValidateConnection()
			if err != nil {
				return classifyLoginError(err), nil
			}

			// Phase 2: Full login with normal timeout
			loginClient := utils.GetRommClient(host, constants.LoginTimeout)
			err = loginClient.Login(host.Username, host.Password)
			if err != nil {
				return classifyLoginError(err), nil
			}

			return loginAttemptResult{Success: true}, nil
		},
	)

	return result.(loginAttemptResult)
}

func classifyLoginError(err error) loginAttemptResult {
	if err == nil {
		return loginAttemptResult{Success: true}
	}

	var protocolErr *romm.ProtocolError
	if errors.As(err, &protocolErr) {
		if protocolErr.CorrectProtocol == "https" {
			return loginAttemptResult{
				ErrorType: "protocol",
				ErrorMsg:  "login_error_use_https",
			}
		}
		return loginAttemptResult{
			ErrorType: "protocol",
			ErrorMsg:  "login_error_use_http",
		}
	}

	switch {
	case errors.Is(err, romm.ErrInvalidHostname):
		return loginAttemptResult{
			ErrorType: "dns",
			ErrorMsg:  "login_error_invalid_hostname",
		}
	case errors.Is(err, romm.ErrConnectionRefused):
		return loginAttemptResult{
			ErrorType: "connection",
			ErrorMsg:  "login_error_connection_refused",
		}
	case errors.Is(err, romm.ErrTimeout):
		return loginAttemptResult{
			ErrorType: "timeout",
			ErrorMsg:  "login_error_timeout",
		}
	case errors.Is(err, romm.ErrWrongProtocol):
		return loginAttemptResult{
			ErrorType: "protocol",
			ErrorMsg:  "login_error_wrong_protocol",
		}
	case errors.Is(err, romm.ErrUnauthorized):
		return loginAttemptResult{
			ErrorType: "credentials",
			ErrorMsg:  "login_error_credentials",
		}
	case errors.Is(err, romm.ErrForbidden):
		return loginAttemptResult{
			ErrorType: "forbidden",
			ErrorMsg:  "login_error_forbidden",
		}
	case errors.Is(err, romm.ErrServerError):
		return loginAttemptResult{
			ErrorType: "server",
			ErrorMsg:  "login_error_server",
		}
	default:
		gabagool.GetLogger().Warn("Unclassified login error", "error", err)
		return loginAttemptResult{
			ErrorType: "unknown",
			ErrorMsg:  "login_error_unexpected",
		}
	}
}

func removeScheme(rawURL string) string {
	if strings.HasPrefix(rawURL, "https://") {
		return strings.TrimPrefix(rawURL, "https://")
	}
	if strings.HasPrefix(rawURL, "http://") {
		return strings.TrimPrefix(rawURL, "http://")
	}
	return rawURL
}
