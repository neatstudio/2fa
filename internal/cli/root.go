package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
	"github.com/gouki/tools/2fa/internal/store"
	"github.com/gouki/tools/2fa/internal/totp"
	"github.com/gouki/tools/2fa/internal/ui"
)

type rootOptions struct {
	storePath string
	once      bool
}

func Run(args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	args = normalizeRootArgs(args)
	defaultPath, err := store.DefaultPath()
	if err != nil {
		return err
	}

	root := flag.NewFlagSet("2fa", flag.ContinueOnError)
	root.SetOutput(errOut)
	opts := rootOptions{}
	root.StringVar(&opts.storePath, "store", defaultPath, "path to accounts JSON file")
	root.BoolVar(&opts.once, "once", false, "print once instead of watching")
	root.Usage = func() {
		fmt.Fprint(root.Output(), helpText(defaultPath))
	}
	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	rest := root.Args()
	if len(rest) == 0 {
		return listAccounts(opts, "", out)
	}

	switch rest[0] {
	case "add":
		return addAccount(opts.storePath, rest[1:], out, errOut)
	case "edit":
		return editAccount(opts.storePath, rest[1:], out, errOut)
	case "delete", "rm":
		return deleteAccount(opts.storePath, rest[1:], in, out, errOut)
	case "help", "--help", "-h":
		fmt.Fprint(out, helpText(defaultPath))
		return nil
	default:
		if strings.HasPrefix(rest[0], "-") {
			return fmt.Errorf("unknown option: %s", rest[0])
		}
		if len(rest) > 1 {
			return fmt.Errorf("unexpected argument: %s", rest[1])
		}
		return listAccounts(opts, rest[0], out)
	}
}

func helpText(defaultPath string) string {
	return fmt.Sprintf(`2fa - local TOTP account viewer

Usage:
  2fa [--once] [--store PATH] [group]
  2fa [--store PATH] add --name NAME --secret BASE32 [--group GROUP] [--note NOTE]
  2fa [--store PATH] edit NAME [--secret BASE32] [--group GROUP] [--note NOTE]
  2fa [--store PATH] delete NAME [--yes]

Commands:
  add       add a TOTP account; name must be globally unique
  edit      update group, note, secret, or any combination
  delete    delete an account by name; asks for confirmation by default

Options:
  --once        print one table and exit
  --store PATH  accounts JSON path (default %s)
  --help        show this help

`, defaultPath)
}

func addAccount(path string, args []string, out io.Writer, errOut io.Writer) error {
	fs := flag.NewFlagSet("2fa add", flag.ContinueOnError)
	fs.SetOutput(errOut)
	name := fs.String("name", "", "globally unique account name")
	secret := fs.String("secret", "", "base32 TOTP secret")
	group := fs.String("group", model.DefaultGroup, "account group")
	note := fs.String("note", "", "account note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	account, err := model.NewAccount(*name, *secret, *group, *note, time.Now())
	if err != nil {
		return err
	}
	if _, err := totp.GenerateDefault(account.Secret, time.Now()); err != nil {
		return err
	}

	st := store.New(path)
	accounts, err := st.Load()
	if err != nil {
		return err
	}
	accounts, err = accounts.Add(account)
	if err != nil {
		return err
	}
	if err := st.Save(accounts); err != nil {
		return err
	}

	table, err := ui.RenderTable(model.Accounts{account}, time.Now())
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(out, table)
	return err
}

func editAccount(path string, args []string, out io.Writer, errOut io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: 2fa edit NAME [--secret BASE32] [--group GROUP] [--note NOTE]")
	}
	name := args[0]
	fs := flag.NewFlagSet("2fa edit", flag.ContinueOnError)
	fs.SetOutput(errOut)
	groupFlag := optionalStringFlag(fs, "group", "account group")
	noteFlag := optionalStringFlag(fs, "note", "account note")
	secretFlag := optionalStringFlag(fs, "secret", "base32 TOTP secret")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: 2fa edit NAME [--secret BASE32] [--group GROUP] [--note NOTE]")
	}
	if groupFlag.Value == nil && noteFlag.Value == nil && secretFlag.Value == nil {
		return errors.New("nothing to edit; pass --group, --note, or --secret")
	}

	changes := model.AccountChanges{
		Group:  groupFlag.Value,
		Note:   noteFlag.Value,
		Secret: secretFlag.Value,
	}
	if changes.Secret != nil {
		normalized := model.NormalizeSecret(*changes.Secret)
		if _, err := totp.GenerateDefault(normalized, time.Now()); err != nil {
			return err
		}
		changes.Secret = &normalized
	}

	st := store.New(path)
	accounts, err := st.Load()
	if err != nil {
		return err
	}
	accounts, err = accounts.Update(name, changes, time.Now())
	if err != nil {
		return err
	}
	if err := st.Save(accounts); err != nil {
		return err
	}
	account, _ := accounts.Find(name)
	table, err := ui.RenderTable(model.Accounts{account}, time.Now())
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(out, table)
	return err
}

func deleteAccount(path string, args []string, in io.Reader, out io.Writer, errOut io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: 2fa delete NAME [--yes]")
	}
	name := args[0]
	fs := flag.NewFlagSet("2fa delete", flag.ContinueOnError)
	fs.SetOutput(errOut)
	yes := fs.Bool("yes", false, "skip confirmation")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: 2fa delete NAME [--yes]")
	}

	if !*yes {
		fmt.Fprintf(out, "Delete %q? type yes to confirm: ", name)
		answer, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "yes" {
			fmt.Fprintln(out, "aborted")
			return nil
		}
	}

	st := store.New(path)
	accounts, err := st.Load()
	if err != nil {
		return err
	}
	accounts, err = accounts.Delete(name)
	if err != nil {
		return err
	}
	if err := st.Save(accounts); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "deleted %s\n", name)
	return err
}

func listAccounts(opts rootOptions, group string, out io.Writer) error {
	st := store.New(opts.storePath)
	accounts, err := st.Load()
	if err != nil {
		return err
	}
	if group != "" {
		accounts = accounts.FilterByGroup(group)
	}

	if opts.once || !isTerminal(out) {
		table, err := ui.RenderTable(accounts, time.Now())
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(out, table)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return ui.Watch(ctx, out, accounts)
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isTerminalFD(file.Fd())
}

func normalizeRootArgs(args []string) []string {
	if len(args) < 2 {
		return args
	}
	normalized := make([]string, 0, len(args))
	tail := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--once" {
			normalized = append(normalized, arg)
			continue
		}
		if arg == "--store" && i+1 < len(args) {
			normalized = append(normalized, arg, args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "--store=") {
			normalized = append(normalized, arg)
			continue
		}
		tail = append(tail, arg)
	}
	return append(normalized, tail...)
}

func optionalStringFlag(fs *flag.FlagSet, name, usage string) *optionalStringValue {
	v := &optionalStringValue{}
	fs.Var(v, name, usage)
	return v
}

type optionalStringValue struct {
	Value *string
}

func (v *optionalStringValue) Set(value string) error {
	copied := value
	v.Value = &copied
	return nil
}

func (v *optionalStringValue) String() string {
	if v.Value == nil {
		return ""
	}
	return *v.Value
}
