package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gabotechs/dep-tree/internal/config"
	"github.com/gabotechs/dep-tree/internal/dart"
	"github.com/gabotechs/dep-tree/internal/dummy"
	"github.com/gabotechs/dep-tree/internal/js"
	"github.com/gabotechs/dep-tree/internal/language"
	"github.com/gabotechs/dep-tree/internal/python"
	"github.com/gabotechs/dep-tree/internal/rust"
	"github.com/gabotechs/dep-tree/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const renderGroupId = "render"
const checkGroupId = "check"
const defaultCommand = "entropy"

var configPath string
var unwrapExports bool
var jsTsConfigPaths bool
var jsWorkspaces bool
var pythonExcludeConditionalImports bool
var exclude []string

var root *cobra.Command

func NewRoot(args []string) *cobra.Command {
	if args == nil {
		args = os.Args[1:]
	}

	root = &cobra.Command{
		Use:               "dep-tree",
		Version:           "v0.19.9",
		Short:             "Visualize and check your project's dependency graph",
		SilenceUsage:      true,
		Args:              cobra.ArbitraryArgs,
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
		Example: `$ dep-tree src/index.ts
$ dep-tree entropy src/index.ts
$ dep-tree tree package/main.py --unwrap-exports
$ dep-tree check`,
		Long: `
      ____         _ __       _
     |  _ \   ___ |  _ \    _| |_  _ __  ___   ___
     | | | | / _ \| |_) |  |_   _||  __|/ _ \ / _ \
     | |_| ||  __/| .__/     | |  | |  |  __/|  __/
     |____/  \__| |_|        | \__|_|   \___| \___|
`,
	}

	root.SetArgs(args)

	root.AddCommand(
		EntropyCmd(),
		TreeCmd(),
		CheckCmd(),
		ConfigCmd(),
	)

	root.AddGroup(&cobra.Group{ID: renderGroupId, Title: "Visualize your dependencies graphically"})
	root.AddGroup(&cobra.Group{ID: checkGroupId, Title: "Check your dependencies against your own rules"})

	root.Flags().SortFlags = false
	root.PersistentFlags().SortFlags = false
	root.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to dep-tree's config file. (default .dep-tree.yml)")
	root.PersistentFlags().BoolVar(&unwrapExports, "unwrap-exports", false, "trace re-exported symbols to the file where they are declared. (default false)")
	root.PersistentFlags().StringArrayVar(&exclude, "exclude", nil, "Files that match this glob pattern will be ignored. You can provide an arbitrary number of --exclude flags.")
	root.PersistentFlags().BoolVar(&jsTsConfigPaths, "js-tsconfig-paths", true, "follow the tsconfig.json paths while resolving imports.")
	root.PersistentFlags().BoolVar(&jsWorkspaces, "js-workspaces", true, "take the workspaces attribute in the root package.json into account for resolving paths.")
	root.PersistentFlags().BoolVar(&pythonExcludeConditionalImports, "python-exclude-conditional-imports", false, "exclude imports wrapped inside if or try statements. (default false)")

	switch {
	case len(args) > 0 && utils.InArray(args[0], []string{"help", "completion", "-v", "--version", "-h", "--help"}):
		// do nothing.
	case len(args) == 0:
		// if not args where provided, default to help.
		root.SetArgs([]string{"help"})
	default:
		// if some args where provided, but it's none of the root commands,
		// choose a default command.
		cmd, _, err := root.Find(args)
		if err == nil && cmd.Use == root.Use && !errors.Is(cmd.Flags().Parse(args), pflag.ErrHelp) {
			root.SetArgs(append([]string{defaultCommand}, args...))
		}
	}
	return root
}

func inferLang(files []string, cfg *config.Config) (language.Language, error) {
	score := struct {
		js     int
		python int
		rust   int
		dummy  int
		dart   int
	}{}
	top := struct {
		lang string
		v    int
	}{}
	for _, file := range files {
		switch {
		case utils.EndsWith(file, js.Extensions):
			score.js += 1
			if score.js > top.v {
				top.v = score.js
				top.lang = "js"
			}
		case utils.EndsWith(file, rust.Extensions):
			score.rust += 1
			if score.rust > top.v {
				top.v = score.rust
				top.lang = "rust"
			}
		case utils.EndsWith(file, python.Extensions):
			score.python += 1
			if score.python > top.v {
				top.v = score.python
				top.lang = "python"
			}
		case utils.EndsWith(file, dummy.Extensions):
			score.dummy += 1
			if score.dummy > top.v {
				top.v = score.dummy
				top.lang = "dummy"
			}
		case utils.EndsWith(file, dart.Extensions):
			score.dart += 1
			if score.dart > top.v {
				top.v = score.dart
				top.lang = "dart"
			}
		}
	}
	if top.lang == "" {
		return nil, errors.New("at least one file must be provided")
	}
	switch top.lang {
	case "js":
		return js.MakeJsLanguage(&cfg.Js)
	case "rust":
		return rust.MakeRustLanguage(&cfg.Rust)
	case "python":
		return python.MakePythonLanguage(&cfg.Python)
	case "dart":
		return &dart.Language{}, nil
	case "dummy":
		return &dummy.Language{}, nil
	default:
		return nil, fmt.Errorf("file \"%s\" not supported", files[0])
	}
}

func filesFromArgs(args []string) ([]string, error) {
	var result []string
	var errs []error
	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			errs = append(errs, err)
		} else if !utils.FileExists(abs) {
			errs = append(errs, fmt.Errorf("file %s does not exist", arg))
		} else {
			result = append(result, abs)
		}
	}
	if len(result) == 0 {
		if len(errs) == 1 {
			return result, errs[0]
		}
		return result, errors.New("no valid files where provided")
	}

	return result, nil
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return nil, err
	}
	if root.PersistentFlags().Changed("unwrap-exports") {
		cfg.UnwrapExports = unwrapExports
	}
	if root.PersistentFlags().Changed("js-tsconfig-paths") {
		cfg.Js.TsConfigPaths = jsTsConfigPaths
	}
	if root.PersistentFlags().Changed("js-workspaces") {
		cfg.Js.Workspaces = jsWorkspaces
	}
	if root.PersistentFlags().Changed("python-exclude-conditional-imports") {
		cfg.Python.ExcludeConditionalImports = pythonExcludeConditionalImports
	}
	// NOTE: hard-enable this for now, as they don't produce a very good output.
	cfg.Python.IgnoreFromImportsAsExports = true
	cfg.Python.IgnoreDirectoryImports = true

	absExclude := make([]string, len(exclude))
	for i, file := range exclude {
		if !filepath.IsAbs(file) {
			cwd, _ := os.Getwd()
			absExclude[i] = filepath.Join(cwd, file)
		} else {
			absExclude[i] = file
		}
	}
	cfg.Exclude = append(cfg.Exclude, exclude...)
	// validate exclusion patterns.
	for _, exclusion := range cfg.Exclude {
		if _, err := utils.GlobstarMatch(exclusion, ""); err != nil {
			return nil, fmt.Errorf("exclude pattern '%s' is not correctly formatted", exclusion)
		}
	}
	return cfg, nil
}
