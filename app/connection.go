package main

import (
	"errors"
	"grout/constants"
	"grout/romm"
	"grout/utils"
)

func validateConnection(host romm.Host) string {
	validationClient := utils.GetRommClient(host, constants.ValidationTimeout)
	err := validationClient.ValidateConnection()
	if err != nil {
		return classifyConnectionError(err)
	}
	return ""
}

func classifyConnectionError(err error) string {
	if err == nil {
		return ""
	}

	var protocolErr *romm.ProtocolError
	if errors.As(err, &protocolErr) {
		if protocolErr.CorrectProtocol == "https" {
			return "startup_error_use_https"
		}
		return "startup_error_use_http"
	}

	switch {
	case errors.Is(err, romm.ErrInvalidHostname):
		return "startup_error_invalid_hostname"
	case errors.Is(err, romm.ErrConnectionRefused):
		return "startup_error_connection_refused"
	case errors.Is(err, romm.ErrTimeout):
		return "startup_error_timeout"
	case errors.Is(err, romm.ErrWrongProtocol):
		return "startup_error_wrong_protocol"
	case errors.Is(err, romm.ErrUnauthorized):
		return "startup_error_credentials"
	case errors.Is(err, romm.ErrForbidden):
		return "startup_error_forbidden"
	case errors.Is(err, romm.ErrServerError):
		return "startup_error_server"
	default:
		return "error_loading_platforms"
	}
}
