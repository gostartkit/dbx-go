package util

import (
	"errors"
	"os"
	"testing"
)

func TestDescribeErrorCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "config not found", err: WrapLayer("config", "load connection", os.ErrNotExist), want: "CONFIG_NOT_FOUND"},
		{name: "validation failed", err: WrapLayer("validation", "validate", errors.New("bad input")), want: "VALIDATION_FAILED"},
		{name: "ssh auth", err: WrapLayer("ssh", "handshake", errors.New("unable to authenticate")), want: "SSH_AUTH_FAILED"},
		{name: "proxy dial", err: WrapLayer("proxy", "dial", errors.New("connection refused")), want: "PROXY_DIAL_FAILED"},
		{name: "mysql access denied", err: WrapLayer("mysql", "ping", errors.New("Error 1045: Access denied for user")), want: "MYSQL_ACCESS_DENIED"},
		{name: "template missing", err: WrapLayer("template", "resolve", errors.New("template not found")), want: "TEMPLATE_NOT_FOUND"},
		{name: "sql execution", err: WrapLayer("sql execution", "execute", errors.New("syntax error")), want: "SQL_EXECUTION_FAILED"},
		{name: "unsupported version", err: WrapLayer("config", "load", errors.New("unsupported version 2")), want: "UNSUPPORTED_VERSION"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := DescribeError(tc.err)
			if info == nil || info.Code != tc.want {
				t.Fatalf("DescribeError(%v) = %+v, want %s", tc.err, info, tc.want)
			}
		})
	}
}
