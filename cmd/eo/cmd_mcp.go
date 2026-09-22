package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/eochat"
	"github.com/evangelischeomroep/eo-cli/internal/mcp"
)

const mcpServerName = "eochat"

func cmdMCP(args []string) error {
	if len(args) > 0 && args[0] == "install" {
		return cmdMCPInstall(args[1:])
	}
	if len(args) > 0 {
		return fmt.Errorf("unknown mcp command %q (see `eo mcp --help`)", args[0])
	}
	return serveMCP()
}

// serveMCP runs the stdio MCP server. Login problems are reported per tool
// call rather than at startup, so the client shows a readable error instead
// of a dead server.
func serveMCP() error {
	server := mcp.NewServer("eo-"+mcpServerName, version, os.Stderr)

	getClient := func() (*eochat.Client, eochat.Config, error) {
		return eochatClient()
	}

	server.AddTool(mcp.Tool{
		Name: "eochat_ask",
		Description: "Ask EOchat, the internal AI assistant of Evangelische Omroep (EO). " +
			"Use it for questions about EO-specific knowledge, internal documentation, systems and conventions, " +
			"or to consult one of EO's custom models. Optionally ground the answer in one or more EOchat knowledge bases.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The question or task for EOchat. Include all relevant context; each call is a fresh conversation.",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "Model ID or name (see eochat_list_models). Defaults to the user's configured model.",
				},
				"knowledge": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Names or IDs of EOchat knowledge bases to use for retrieval (see eochat_list_knowledge).",
				},
				"system": map[string]any{
					"type":        "string",
					"description": "Optional system prompt.",
				},
			},
			"required": []string{"prompt"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			prompt := strings.TrimSpace(mcp.StringArg(args, "prompt"))
			if prompt == "" {
				return "", errors.New("prompt is required")
			}
			client, cfg, err := getClient()
			if err != nil {
				return "", err
			}
			session := eochat.Session{System: mcp.StringArg(args, "system")}
			if q := mcp.StringArg(args, "model"); q != "" {
				models, err := client.Models(ctx)
				if err != nil {
					return "", describeAskError(err)
				}
				m, err := eochat.ResolveModel(models, q)
				if err != nil {
					return "", err
				}
				session.Model = m.ID
			} else {
				session.Model, err = pickDefaultModel(ctx, client, cfg)
				if err != nil {
					return "", describeAskError(err)
				}
			}
			if kb := mcp.StringSliceArg(args, "knowledge"); len(kb) > 0 {
				if err := attachInputs(ctx, client, &session, askOptions{knowledge: kb}); err != nil {
					return "", describeAskError(err)
				}
			}
			session.Messages = []eochat.Message{{Role: "user", Content: prompt}}
			result, err := client.Chat(ctx, eochat.ChatRequest{
				Model:    session.Model,
				Messages: session.PromptMessages(),
				Files:    session.Files,
			})
			if err != nil {
				return "", describeAskError(err)
			}
			text := result.Content
			if len(result.Sources) > 0 {
				names := make([]string, 0, len(result.Sources))
				for _, s := range result.Sources {
					names = append(names, s.Name)
				}
				text += "\n\nSources: " + strings.Join(names, ", ")
			}
			return text, nil
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "eochat_list_models",
		Description: "List the EOchat models available to the current user, with their IDs and descriptions.",
		Handler: func(ctx context.Context, _ map[string]any) (string, error) {
			client, cfg, err := getClient()
			if err != nil {
				return "", err
			}
			models, err := client.Models(ctx)
			if err != nil {
				return "", describeAskError(err)
			}
			var sb strings.Builder
			for _, m := range models {
				fmt.Fprintf(&sb, "- %s", m.ID)
				if m.Name != "" && m.Name != m.ID {
					fmt.Fprintf(&sb, " (%s)", m.Name)
				}
				if m.ID == cfg.Model {
					sb.WriteString(" [default]")
				}
				if d := m.Description(); d != "" {
					fmt.Fprintf(&sb, ": %s", firstLine(d))
				}
				sb.WriteByte('\n')
			}
			if sb.Len() == 0 {
				return "No models available.", nil
			}
			return sb.String(), nil
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "eochat_list_knowledge",
		Description: "List the EOchat knowledge bases (document collections) the current user can use to ground an eochat_ask call.",
		Handler: func(ctx context.Context, _ map[string]any) (string, error) {
			client, _, err := getClient()
			if err != nil {
				return "", err
			}
			items, err := client.Knowledge(ctx)
			if err != nil {
				return "", describeAskError(err)
			}
			var sb strings.Builder
			for _, k := range items {
				fmt.Fprintf(&sb, "- %s (id: %s)", k.Name, k.ID)
				if d := strings.TrimSpace(k.Description); d != "" {
					fmt.Fprintf(&sb, ": %s", firstLine(d))
				}
				sb.WriteByte('\n')
			}
			if sb.Len() == 0 {
				return "No knowledge bases available.", nil
			}
			return sb.String(), nil
		},
	})

	return server.Serve(context.Background(), os.Stdin, os.Stdout)
}

// cmdMCPInstall registers the server with Claude Code via `claude mcp add`.
func cmdMCPInstall(args []string) error {
	scope := "user"
	if hasFlag(args, "--project") {
		scope = "project"
	}
	exe := "eo"
	if _, err := exec.LookPath("eo"); err != nil {
		if path, err := os.Executable(); err == nil {
			exe = path
		}
	}

	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Println(yellow("⚠ ") + "Claude Code (`claude`) not found in PATH. Add the server by hand:")
		fmt.Println()
		fmt.Println("  " + cyan(fmt.Sprintf("claude mcp add --scope %s %s -- %s mcp", scope, mcpServerName, exe)))
		fmt.Println()
		fmt.Println(dim("  or in .mcp.json / ~/.claude.json:"))
		fmt.Printf("  {\"mcpServers\": {\"%s\": {\"command\": \"%s\", \"args\": [\"mcp\"]}}}\n", mcpServerName, exe)
		return nil
	}

	cmd := exec.Command(claude, "mcp", "add", "--scope", scope, mcpServerName, "--", exe, "mcp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Println(dim("→ " + strings.Join(cmd.Args, " ")))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude mcp add failed: %w", err)
	}
	fmt.Println()
	fmt.Println(green("✓ ") + "EOchat is available in Claude Code as the " + cyan(mcpServerName) + " MCP server.")
	fmt.Println(dim("  Ask Claude Code something like: \"Use eochat_ask to check how EO handles X.\""))
	if _, _, err := eochatClient(); err != nil {
		fmt.Println()
		fmt.Println(yellow("⚠ ") + "You are not logged in yet — run " + cyan("eo ask login") + " so the server can reach EOchat.")
	}
	return nil
}
