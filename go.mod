module prometheus-llm-proxy

go 1.24.4

replace github.com/WoodProgrammer/prometheus-llm-proxy/cmd => ./cmd

replace github.com/WoodProgrammer/prometheus-llm-proxy/internal => ./internal

require (
	github.com/WoodProgrammer/prometheus-llm-proxy/cmd v0.0.0-00010101000000-000000000000
	github.com/WoodProgrammer/prometheus-llm-proxy/internal v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
