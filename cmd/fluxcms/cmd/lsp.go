package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/infrasutra/fsl/lsp"
	"github.com/spf13/cobra"
)

var (
	lspStdio  bool
	lspSocket string
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the FSL Language Server Protocol server",
	Long: `Start the FSL Language Server for editor integration.

The LSP server provides:
  - Real-time error diagnostics
  - Autocomplete for types and decorators
  - Hover information
  - Go-to-definition
  - Document symbols outline

Examples:
  # Start with stdio transport (default, for editors)
  fluxcms lsp --stdio

  # Start with socket transport
  fluxcms lsp --socket=:9999`,
	RunE: runLSP,
}

func init() {
	rootCmd.AddCommand(lspCmd)
	lspCmd.Flags().BoolVar(&lspStdio, "stdio", true, "Use stdio transport")
	lspCmd.Flags().StringVar(&lspSocket, "socket", "", "Use socket transport on specified address (e.g., :9999)")
}

func runLSP(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if lspSocket != "" {
		return runSocketServer(ctx, lspSocket)
	}

	return runStdioServer(ctx)
}

func runStdioServer(ctx context.Context) error {
	server := lsp.NewServer(os.Stdin, os.Stdout)
	return server.Run(ctx)
}

func runSocketServer(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer listener.Close()

	fmt.Fprintf(os.Stderr, "FSL LSP server listening on %s\n", addr)

	// Accept connections in a loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Handle connection in goroutine
		go func(c net.Conn) {
			defer c.Close()
			server := lsp.NewServer(c, c)
			server.Run(context.Background())
		}(conn)
	}
}
