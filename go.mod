module pkg.gostartkit.com/dbx

go 1.25.0

require (
	github.com/go-sql-driver/mysql v1.10.0
	golang.org/x/crypto v0.51.0
	golang.org/x/net v0.53.0
	golang.org/x/term v0.43.0
	pkg.gostartkit.com/cmd v0.2.1-0.20260522082509-ec310b3acadf
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
)

replace pkg.gostartkit.com/cmd => ../cmd
