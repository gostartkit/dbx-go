package util

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Layer   string `json:"layer,omitempty"`
}

func DescribeError(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	layer := deepestLayer(err)
	root := rootCause(err)
	lowerRoot := strings.ToLower(root.Error())

	switch {
	case errors.Is(err, os.ErrNotExist):
		return &ErrorInfo{Code: "CONFIG_NOT_FOUND", Message: "config not found", Layer: coalesceLayer(layer, "config")}
	case layer == "validation" && strings.Contains(lowerRoot, "confirmation required"):
		return &ErrorInfo{Code: "CONFIRMATION_REQUIRED", Message: "confirmation required: rerun with --yes", Layer: layer}
	case layer == "validation":
		return &ErrorInfo{Code: "VALIDATION_FAILED", Message: "validation failed", Layer: layer}
	case layer == "config" && strings.Contains(lowerRoot, "unsupported version"):
		return &ErrorInfo{Code: "UNSUPPORTED_VERSION", Message: "unsupported schema version", Layer: layer}
	case layer == "config":
		return &ErrorInfo{Code: "CONFIG_ERROR", Message: root.Error(), Layer: layer}
	case layer == "ssh" && (strings.Contains(lowerRoot, "authenticate") || strings.Contains(lowerRoot, "authentication")):
		return &ErrorInfo{Code: "SSH_AUTH_FAILED", Message: "ssh authentication failed", Layer: layer}
	case layer == "ssh":
		return &ErrorInfo{Code: "SSH_CONNECTION_FAILED", Message: root.Error(), Layer: layer}
	case layer == "proxy":
		return &ErrorInfo{Code: "PROXY_DIAL_FAILED", Message: "proxy dial failed", Layer: layer}
	case layer == "mysql" && strings.Contains(lowerRoot, "access denied"):
		return &ErrorInfo{Code: "MYSQL_ACCESS_DENIED", Message: "mysql access denied", Layer: layer}
	case layer == "mysql":
		return &ErrorInfo{Code: "MYSQL_ERROR", Message: root.Error(), Layer: layer}
	case layer == "template" && strings.Contains(lowerRoot, "not found"):
		return &ErrorInfo{Code: "TEMPLATE_NOT_FOUND", Message: "template not found", Layer: layer}
	case layer == "template":
		return &ErrorInfo{Code: "TEMPLATE_ERROR", Message: root.Error(), Layer: layer}
	case layer == "sql execution":
		return &ErrorInfo{Code: "SQL_EXECUTION_FAILED", Message: "sql execution failed", Layer: layer}
	default:
		return &ErrorInfo{Code: "UNKNOWN_ERROR", Message: root.Error(), Layer: layer}
	}
}

func deepestLayer(err error) string {
	layer := ""
	for current := err; current != nil; current = errors.Unwrap(current) {
		layerErr, ok := current.(*LayerError)
		if !ok {
			continue
		}
		layer = layerErr.Layer
	}
	return layer
}

func rootCause(err error) error {
	current := err
	for {
		layerErr, ok := current.(*LayerError)
		if !ok || layerErr.Err == nil {
			break
		}
		current = layerErr.Err
	}
	if current == nil {
		return fmt.Errorf("unknown error")
	}
	return current
}

func coalesceLayer(layer string, fallback string) string {
	if strings.TrimSpace(layer) != "" {
		return layer
	}
	return fallback
}
