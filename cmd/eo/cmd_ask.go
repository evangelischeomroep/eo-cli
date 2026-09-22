package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"charm.land/huh/v2"

	"github.com/evangelischeomroep/eo-cli/internal/eochat"
)

const eochatKeyHint = "Create an API key at https://chat.eo.nl → Settings → Account → API keys and run `eo ask login`."

type askOptions struct {
	model     string
	cont      bool
	system    string
	files     []string
	knowledge []string
	web       bool
	raw       bool
	prompt    string
}

func parseAskArgs(args []string) (askOptions, error) {
	var o askOptions
	var words []string
	takeValue := func(i *int, flag string) (string, error) {
		if eq := strings.IndexByte(flag, '='); eq >= 0 {
			return flag[eq+1:], nil
		}
		if *i+1 >= len(args) {
			return "", fmt.Errorf("flag %s needs a value", flag)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name := a
		if eq := strings.IndexByte(a, '='); eq >= 0 && strings.HasPrefix(a, "-") {
			name = a[:eq]
		}
		switch name {
		case "--":
			words = append(words, args[i+1:]...)
			i = len(args)
		case "-m", "--model":
			v, err := takeValue(&i, a)
			if err != nil {
				return o, err
			}
			o.model = v
		case "-c", "--continue":
			o.cont = true
		case "-s", "--system":
			v, err := takeValue(&i, a)
			if err != nil {
				return o, err
			}
			o.system = v
		case "-f", "--file":
			v, err := takeValue(&i, a)
			if err != nil {
				return o, err
			}
			o.files = append(o.files, v)
		case "-k", "--knowledge":
			v, err := takeValue(&i, a)
			if err != nil {
				return o, err
			}
			o.knowledge = append(o.knowledge, v)
		case "--web":
			o.web = true
		case "--raw":
			o.raw = true
		default:
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return o, fmt.Errorf("unknown flag %s (see `eo ask --help`)", a)
			}
			words = append(words, a)
		}
	}
	o.prompt = strings.TrimSpace(strings.Join(words, " "))
	return o, nil
}

func stdinIsTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func readPipedStdin() string {
	if stdinIsTerminal() {
		return ""
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// eochatClient builds a client from config + environment, or explains how to log in.
func eochatClient() (*eochat.Client, eochat.Config, error) {
	fileCfg, err := eochat.LoadConfig()
	if err != nil {
		return nil, eochat.Config{}, err
	}
	cfg := fileCfg.WithEnv()
	if cfg.APIKey == "" {
		return nil, cfg, errors.New("not logged in to EOchat. " + eochatKeyHint)
	}
	client := eochat.NewClient(cfg.BaseURL(), cfg.APIKey)
	client.UserAgent = "eo-cli/" + version
	return client, cfg, nil
}

func describeAskError(err error) error {
	if errors.Is(err, eochat.ErrUnauthorized) {
		return errors.New("EOchat rejected your API key. " + eochatKeyHint)
	}
	return err
}

func cmdAsk(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "login":
			return cmdAskLogin(args[1:])
		case "logout":
			return cmdAskLogout()
		case "status":
			return cmdAskStatus()
		case "models":
			return cmdAskModels()
		case "model":
			return cmdAskModel(args[1:])
		case "knowledge":
			return cmdAskKnowledge()
		}
	}

	opts, err := parseAskArgs(args)
	if err != nil {
		return err
	}

	piped := readPipedStdin()
	switch {
	case piped != "" && opts.prompt == "":
		opts.prompt = piped
	case piped != "":
		opts.prompt = opts.prompt + "\n\n" + piped
	}
	interactive := opts.prompt == "" && stdinIsTerminal()
	if opts.prompt == "" && !interactive {
		return errors.New("nothing to ask: pass a question or pipe input into `eo ask`")
	}

	client, cfg, err := eochatClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	var session eochat.Session
	if opts.cont {
		s, ok, err := eochat.LoadSession()
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("no previous conversation to continue")
		}
		session = s
	}
	if opts.system != "" {
		session.System = opts.system
	}

	if opts.model != "" {
		models, err := client.Models(ctx)
		if err != nil {
			return describeAskError(err)
		}
		m, err := eochat.ResolveModel(models, opts.model)
		if err != nil {
			return err
		}
		session.Model = m.ID
	}
	if session.Model == "" {
		session.Model, err = pickDefaultModel(ctx, client, cfg)
		if err != nil {
			return describeAskError(err)
		}
	}

	if err := attachInputs(ctx, client, &session, opts); err != nil {
		return describeAskError(err)
	}

	if interactive {
		return runAskREPL(client, &session, opts)
	}
	return askOnce(ctx, client, &session, opts.prompt, opts)
}

// pickDefaultModel prefers the configured model, then the server default,
// then the first model the user may use.
func pickDefaultModel(ctx context.Context, client *eochat.Client, cfg eochat.Config) (string, error) {
	if cfg.Model != "" {
		return cfg.Model, nil
	}
	if id, err := client.DefaultModel(ctx); err == nil && id != "" {
		return id, nil
	}
	models, err := client.Models(ctx)
	if err != nil {
		return "", err
	}
	if len(models) == 0 {
		return "", errors.New("no models available on EOchat for your account")
	}
	return models[0].ID, nil
}

func attachInputs(ctx context.Context, client *eochat.Client, session *eochat.Session, opts askOptions) error {
	for _, path := range opts.files {
		fmt.Fprintln(os.Stderr, dim("→ Uploading "+path+"..."))
		f, err := client.UploadFile(ctx, path)
		if err != nil {
			return fmt.Errorf("uploading %s: %w", path, err)
		}
		session.Files = append(session.Files, eochat.Attachment{Type: "file", ID: f.ID, Name: f.Filename})
	}
	if len(opts.knowledge) > 0 {
		items, err := client.Knowledge(ctx)
		if err != nil {
			return err
		}
		for _, q := range opts.knowledge {
			k, err := eochat.ResolveKnowledge(items, q)
			if err != nil {
				return err
			}
			session.Files = append(session.Files, eochat.Attachment{Type: "collection", ID: k.ID, Name: k.Name})
		}
	}
	return nil
}

// askOnce sends one prompt, streams the answer to stdout and saves the session.
func askOnce(ctx context.Context, client *eochat.Client, session *eochat.Session, prompt string, opts askOptions) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	_, err := streamAnswer(ctx, client, session, prompt, opts)
	return err
}

// streamAnswer appends the prompt to the session, streams the reply and
// records it. Returns the reply text; on interruption the partial reply is kept.
func streamAnswer(ctx context.Context, client *eochat.Client, session *eochat.Session, prompt string, opts askOptions) (string, error) {
	session.Messages = append(session.Messages, eochat.Message{Role: "user", Content: prompt})

	styled := useColor && !opts.raw
	md := newMDWriter(os.Stdout, styled)
	var first sync.Once
	if styled {
		fmt.Print(dim("…"))
	}
	clearSpinner := func() {
		if styled {
			fmt.Print("\r\033[K")
		}
	}
	result, err := client.ChatStream(ctx, eochat.ChatRequest{
		Model:     session.Model,
		Messages:  session.PromptMessages(),
		Files:     session.Files,
		WebSearch: opts.web,
	}, func(delta string) {
		first.Do(clearSpinner)
		md.write(delta)
	})
	first.Do(clearSpinner)
	md.flush()
	if result.Content != "" && !strings.HasSuffix(result.Content, "\n") {
		fmt.Println()
	}

	if result.Content != "" {
		session.Messages = append(session.Messages, eochat.Message{Role: "assistant", Content: result.Content})
	} else if err != nil {
		// Nothing came back: drop the question so a retry does not duplicate it.
		session.Messages = session.Messages[:len(session.Messages)-1]
	}
	if saveErr := eochat.SaveSession(*session); saveErr != nil && err == nil {
		err = saveErr
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, dim("(interrupted)"))
			return result.Content, nil
		}
		return result.Content, describeAskError(err)
	}
	if len(result.Sources) > 0 && styled {
		names := make([]string, 0, len(result.Sources))
		for _, s := range result.Sources {
			names = append(names, s.Name)
		}
		fmt.Println(dim("sources: " + strings.Join(names, ", ")))
	}
	return result.Content, nil
}

const replPrompt = "❯ "

func runAskREPL(client *eochat.Client, session *eochat.Session, opts askOptions) error {
	fmt.Println(bold("EOchat") + dim(" · "+session.Model))
	if opts.cont && len(session.Messages) > 0 {
		fmt.Println(dim(fmt.Sprintf("continuing a conversation with %d messages", len(session.Messages))))
	}
	fmt.Println(dim("Type your question. /help for commands, /quit or Ctrl+D to exit."))
	fmt.Println()

	// Ctrl+C cancels a running answer; pressed twice while idle it exits.
	var mu sync.Mutex
	var cancelCurrent context.CancelFunc
	var lastInterrupt time.Time
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		for range sigCh {
			mu.Lock()
			if cancelCurrent != nil {
				cancelCurrent()
			} else {
				if time.Since(lastInterrupt) < 2*time.Second {
					fmt.Println()
					os.Exit(0)
				}
				lastInterrupt = time.Now()
				fmt.Print("\n" + dim("(press Ctrl+C again or /quit to exit)") + "\n" + cyan(replPrompt))
			}
			mu.Unlock()
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(cyan(replPrompt))
		input, err := readInput(reader)
		if err != nil {
			fmt.Println()
			return nil
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.HasPrefix(input, "/") {
			quit, err := handleReplCommand(client, session, &opts, input)
			if err != nil {
				fmt.Fprintln(os.Stderr, red("✗ ")+err.Error())
			}
			if quit {
				return nil
			}
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		mu.Lock()
		cancelCurrent = cancel
		mu.Unlock()

		fmt.Println()
		_, err = streamAnswer(ctx, client, session, input, opts)

		mu.Lock()
		cancelCurrent = nil
		mu.Unlock()
		cancel()

		if err != nil {
			fmt.Fprintln(os.Stderr, red("✗ ")+err.Error())
		}
		fmt.Println()
	}
}

// readInput reads one line; a trailing backslash continues on the next line.
func readInput(reader *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasSuffix(line, "\\") {
			sb.WriteString(strings.TrimSuffix(line, "\\"))
			sb.WriteByte('\n')
			fmt.Print(dim("… "))
			continue
		}
		sb.WriteString(line)
		return sb.String(), nil
	}
}

func handleReplCommand(client *eochat.Client, session *eochat.Session, opts *askOptions, input string) (quit bool, err error) {
	fields := strings.Fields(input)
	cmd, rest := fields[0], strings.TrimSpace(strings.TrimPrefix(input, fields[0]))
	ctx := context.Background()
	switch cmd {
	case "/quit", "/exit", "/q":
		return true, nil
	case "/help", "/?":
		fmt.Println(dim("  /model [name]   switch model (interactive picker without a name)"))
		fmt.Println(dim("  /models         list available models"))
		fmt.Println(dim("  /knowledge      list knowledge bases"))
		fmt.Println(dim("  /use <name>     attach a knowledge base to this conversation"))
		fmt.Println(dim("  /file <path>    upload and attach a file"))
		fmt.Println(dim("  /system <text>  set a system prompt"))
		fmt.Println(dim("  /web            toggle web search"))
		fmt.Println(dim("  /new            start a fresh conversation"))
		fmt.Println(dim("  /quit           exit (Ctrl+D also works)"))
		fmt.Println(dim("  end a line with \\ to continue on the next line"))
		return false, nil
	case "/new":
		session.Messages = nil
		session.Files = nil
		if err := eochat.SaveSession(*session); err != nil {
			return false, err
		}
		fmt.Println(dim("new conversation"))
		return false, nil
	case "/models":
		return false, cmdAskModels()
	case "/knowledge":
		return false, cmdAskKnowledge()
	case "/model":
		models, err := client.Models(ctx)
		if err != nil {
			return false, describeAskError(err)
		}
		var m eochat.Model
		if rest == "" {
			m, err = selectModel(models, session.Model)
		} else {
			m, err = eochat.ResolveModel(models, rest)
		}
		if err != nil {
			return false, err
		}
		session.Model = m.ID
		fmt.Println(dim("model: ") + cyan(m.ID))
		return false, nil
	case "/use":
		if rest == "" {
			return false, errors.New("usage: /use <knowledge base>")
		}
		if err := attachInputs(ctx, client, session, askOptions{knowledge: []string{rest}}); err != nil {
			return false, describeAskError(err)
		}
		fmt.Println(dim("attached: ") + cyan(session.Files[len(session.Files)-1].Name))
		return false, nil
	case "/file":
		if rest == "" {
			return false, errors.New("usage: /file <path>")
		}
		if err := attachInputs(ctx, client, session, askOptions{files: []string{rest}}); err != nil {
			return false, describeAskError(err)
		}
		fmt.Println(dim("attached: ") + cyan(session.Files[len(session.Files)-1].Name))
		return false, nil
	case "/system":
		session.System = rest
		if rest == "" {
			fmt.Println(dim("system prompt cleared"))
		} else {
			fmt.Println(dim("system prompt set"))
		}
		return false, nil
	case "/web":
		opts.web = !opts.web
		if opts.web {
			fmt.Println(dim("web search: on"))
		} else {
			fmt.Println(dim("web search: off"))
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown command %s — try /help", cmd)
	}
}

func selectModel(models []eochat.Model, current string) (eochat.Model, error) {
	if len(models) == 0 {
		return eochat.Model{}, errors.New("no models available")
	}
	options := make([]huh.Option[string], 0, len(models))
	byID := map[string]eochat.Model{}
	for _, m := range models {
		byID[m.ID] = m
		label := m.Name
		if label == "" {
			label = m.ID
		}
		if label != m.ID {
			label += " " + dim(m.ID)
		}
		options = append(options, huh.NewOption(label, m.ID))
	}
	selected := current
	if _, ok := byID[selected]; !ok {
		selected = models[0].ID
	}
	err := huh.NewSelect[string]().
		Title("Choose a model").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return eochat.Model{}, err
	}
	return byID[selected], nil
}

// --- subcommands ---

func cmdAskLogin(args []string) error {
	fileCfg, err := eochat.LoadConfig()
	if err != nil {
		return err
	}
	key := firstPositional(args)
	if key == "" && stdinIsTerminal() {
		fmt.Println(bold("EOchat login"))
		fmt.Println(dim("  Create a personal API key at ") + cyan(fileCfg.BaseURL()) + dim(" → Settings → Account → API keys."))
		fmt.Println()
		err := huh.NewInput().
			Title("API key").
			EchoMode(huh.EchoModePassword).
			Value(&key).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New("paste your API key")
				}
				return nil
			}).
			Run()
		if err != nil {
			return err
		}
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("usage: eo ask login <api-key>")
	}

	ctx := context.Background()
	client := eochat.NewClient(fileCfg.WithEnv().BaseURL(), key)
	client.UserAgent = "eo-cli/" + version
	fmt.Println(dim("→ Checking key with EOchat..."))
	user, err := client.Me(ctx)
	if err != nil {
		return describeAskError(err)
	}

	fileCfg.APIKey = key
	if err := eochat.SaveConfig(fileCfg); err != nil {
		return err
	}
	fmt.Printf("%s Logged in as %s %s\n", green("✓"), bold(user.Name), dim("("+user.Email+")"))

	if fileCfg.Model == "" && stdinIsTerminal() {
		models, err := client.Models(ctx)
		if err == nil && len(models) > 0 {
			current, _ := client.DefaultModel(ctx)
			fmt.Println()
			m, err := selectModel(models, current)
			if err == nil {
				fileCfg.Model = m.ID
				if err := eochat.SaveConfig(fileCfg); err != nil {
					return err
				}
				fmt.Printf("%s Default model: %s\n", green("✓"), cyan(m.ID))
			}
		}
	}
	fmt.Println()
	fmt.Println(dim("Try: ") + cyan(`eo ask "hoe werkt onze deploy pipeline?"`))
	return nil
}

func cmdAskLogout() error {
	cfg, err := eochat.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.APIKey == "" {
		fmt.Println(dim("Not logged in."))
		return nil
	}
	cfg.APIKey = ""
	if err := eochat.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Println(green("✓ ") + "Logged out of EOchat. The key itself is still valid — revoke it in EOchat settings if needed.")
	return nil
}

func cmdAskStatus() error {
	fileCfg, err := eochat.LoadConfig()
	if err != nil {
		return err
	}
	cfg := fileCfg.WithEnv()
	row := func(label, value string) { fmt.Printf("  %-14s%s\n", label, value) }
	row("Server", cyan(cfg.BaseURL()))
	if cfg.APIKey == "" {
		row("Login", red("not logged in")+dim(" — run `eo ask login`"))
		return nil
	}
	client := eochat.NewClient(cfg.BaseURL(), cfg.APIKey)
	client.UserAgent = "eo-cli/" + version
	user, err := client.Me(context.Background())
	if err != nil {
		row("Login", red("key rejected")+dim(" — run `eo ask login`"))
	} else {
		row("User", bold(user.Name)+" "+dim("("+user.Email+")"))
	}
	model := cfg.Model
	if model == "" {
		model = dim("server default")
	} else {
		model = cyan(model)
	}
	row("Model", model)
	if s, ok, _ := eochat.LoadSession(); ok {
		row("Last chat", fmt.Sprintf("%d messages, %s %s", len(s.Messages), s.UpdatedAt.Format("2006-01-02 15:04"), dim("(continue with `eo ask -c`)")))
	}
	if path, err := eochat.ConfigPath(); err == nil {
		row("Config", dim(path))
	}
	return nil
}

func cmdAskModels() error {
	client, cfg, err := eochatClient()
	if err != nil {
		return err
	}
	models, err := client.Models(context.Background())
	if err != nil {
		return describeAskError(err)
	}
	if len(models) == 0 {
		fmt.Println(dim("No models available."))
		return nil
	}
	width := 0
	for _, m := range models {
		if len(m.ID) > width {
			width = len(m.ID)
		}
	}
	for _, m := range models {
		marker := "  "
		if m.ID == cfg.Model {
			marker = green("✓ ")
		}
		name := m.Name
		if name == "" || name == m.ID {
			name = ""
		}
		line := marker + cyan(pad(m.ID, width)) + "  " + name
		if d := m.Description(); d != "" {
			line += "\n    " + dim(firstLine(d))
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println(dim("Set a default with ") + cyan("eo ask model <name>") + dim(", or use one once with ") + cyan("eo ask -m <name> ..."))
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 100 {
		s = s[:100] + "…"
	}
	return s
}

func cmdAskModel(args []string) error {
	client, _, err := eochatClient()
	if err != nil {
		return err
	}
	fileCfg, err := eochat.LoadConfig()
	if err != nil {
		return err
	}
	models, err := client.Models(context.Background())
	if err != nil {
		return describeAskError(err)
	}
	var m eochat.Model
	if q := firstPositional(args); q != "" {
		m, err = eochat.ResolveModel(models, q)
	} else if stdinIsTerminal() {
		m, err = selectModel(models, fileCfg.Model)
	} else {
		return errors.New("usage: eo ask model <name>")
	}
	if err != nil {
		return err
	}
	fileCfg.Model = m.ID
	if err := eochat.SaveConfig(fileCfg); err != nil {
		return err
	}
	fmt.Printf("%s Default model: %s\n", green("✓"), cyan(m.ID))
	return nil
}

func cmdAskKnowledge() error {
	client, _, err := eochatClient()
	if err != nil {
		return err
	}
	items, err := client.Knowledge(context.Background())
	if err != nil {
		return describeAskError(err)
	}
	if len(items) == 0 {
		fmt.Println(dim("No knowledge bases available."))
		return nil
	}
	for _, k := range items {
		line := "  " + cyan(k.Name)
		if d := strings.TrimSpace(k.Description); d != "" {
			line += "  " + dim(firstLine(d))
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println(dim("Use one with ") + cyan(`eo ask -k "<name>" "..."`))
	return nil
}
