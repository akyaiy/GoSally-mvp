package hooks

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/akyaiy/GoSally-mvp/internal/colors"
	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/core/run_manager"
	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"gopkg.in/ini.v1"
)

// The config composer needs to be in the global scope
var Compositor *config.Compositor = config.NewCompositor()

func InitGlobalLoggerHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	x.Config = Compositor
	x.Log.SetOutput(os.Stdout)
	x.Log.SetPrefix(colors.SetBrightBlack(fmt.Sprintf("(%s) ", cs.Stage)))
	x.Log.SetFlags(log.Ldate | log.Ltime)
}

// First stage: pre-init
func InitCorestateHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	*cs = *corestate.NewCorestate(&corestate.CoreState{
		UUID32DirName:      "uuid",
		NodeBinName:        filepath.Base(os.Args[0]),
		NodeVersion:        config.NodeVersion,
		MetaDir:            "./.meta",
		Stage:              corestate.StagePreInit,
		StartTimestampUnix: time.Now().Unix(),
	})
}

func InitConfigLoadHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	x.Log.SetPrefix(colors.SetYellow(fmt.Sprintf("(%s) ", cs.Stage)))

	if err := x.Config.LoadEnv(); err != nil {
		x.Log.Fatalf("env load error: %s", err)
	}
	cs.NodePath = *x.Config.Env.NodePath

	if cfgPath := x.Config.CMDLine.Run.ConfigPath; cfgPath != "" {
		x.Config.Env.ConfigPath = &cfgPath
	}
	if err := x.Config.LoadConf(*x.Config.Env.ConfigPath); err != nil {
		x.Log.Fatalf("conf load error: %s", err)
	}
}

// The hook reads or prepares a persistent uuid for the node
func InitUUIDHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	uuid32, err := corestate.GetNodeUUID(filepath.Join(cs.MetaDir, "uuid"))
	if errors.Is(err, fs.ErrNotExist) {
		if err := corestate.SetNodeUUID(filepath.Join(cs.NodePath, cs.MetaDir, cs.UUID32DirName)); err != nil {
			x.Log.Fatalf("Cannod generate node uuid: %s", err.Error())
		}
		uuid32, err = corestate.GetNodeUUID(filepath.Join(cs.MetaDir, "uuid"))
		if err != nil {
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
	}
	if err != nil {
		x.Log.Fatalf("uuid load error: %s", err)
	}
	cs.UUID32 = uuid32
	corestate.NODE_UUID = uuid32
}

// The hook is responsible for checking the initialization stage 
// and restarting in some cases
func InitRuntimeHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	if *x.Config.Env.ParentStagePID != os.Getpid() {
		// still pre-init stage
		runDir, err := run_manager.Create(cs.UUID32)
		if err != nil {
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
		cs.RunDir = runDir
		input, err := os.Open(os.Args[0])
		if err != nil {
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
		if err := run_manager.Set(cs.NodeBinName); err != nil {
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
		fmgr := run_manager.File(cs.NodeBinName)
		output, err := fmgr.Open()
		if err != nil {
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}

		if _, err := io.Copy(output, input); err != nil {
			fmgr.Close()
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
		if err := os.Chmod(filepath.Join(cs.RunDir, cs.NodeBinName), 0755); err != nil {
			fmgr.Close()
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
		input.Close()
		fmgr.Close()
		runArgs := os.Args
		runArgs[0] = filepath.Join(cs.RunDir, cs.NodeBinName)

		// prepare environ
		env := utils.SetEviron(os.Environ(), fmt.Sprintf("GS_PARENT_PID=%d", os.Getpid()))

		if err := syscall.Exec(runArgs[0], runArgs, env); err != nil {
			_ = run_manager.Clean()
			x.Log.Fatalf("Unexpected failure: %s", err.Error())
		}
	}
	x.Log.Printf("Node uuid is %s", cs.UUID32)
}

// post-init stage
// The hook creates a run.lock file, which contains information 
// about the process and the node, in the runtime directory.
func InitRunlockHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	NodeApp.Fallback(func(ctx context.Context, cs *corestate.CoreState, x *app.AppX) {
		x.Log.Println("Cleaning up...")

		if err := run_manager.Clean(); err != nil {
			x.Log.Printf("%s: Cleanup error: %s", colors.PrintError(), err.Error())
		}
		x.Log.Println("bye!")
	})

	cs.Stage = corestate.StagePostInit
	x.Log.SetPrefix(colors.SetBlue(fmt.Sprintf("(%s) ", cs.Stage)))

	cs.RunDir = run_manager.Toggle()
	exist, err := utils.ExistsMatchingDirs(filepath.Join(os.TempDir(), fmt.Sprintf("/*-%s-%s", cs.UUID32, "gosally-runtime")), cs.RunDir)
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	if exist {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unable to continue node operation: A node with the same identifier was found in the runtime environment")
	}

	if err := run_manager.Set("run.lock"); err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	lockPath, err := run_manager.Get("run.lock")
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	lockFile := ini.Empty()
	secRun, err := lockFile.NewSection("runtime")
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	secRun.Key("pid").SetValue(fmt.Sprintf("%d/%d", os.Getpid(), x.Config.Env.ParentStagePID))
	secRun.Key("version").SetValue(cs.NodeVersion)
	secRun.Key("uuid").SetValue(cs.UUID32)
	secRun.Key("timestamp").SetValue(time.Unix(cs.StartTimestampUnix, 0).Format("2006-01-02/15:04:05 MST"))
	secRun.Key("timestamp-unix").SetValue(fmt.Sprintf("%d", cs.StartTimestampUnix))

	err = lockFile.SaveTo(lockPath)
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
}

// The hook reads the configuration and replaces special expressions 
// (%tmp% and so on) in string fields with the required data.
func InitConfigReplHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	if !slices.Contains(*x.Config.Conf.DisableWarnings, "--WNonStdTmpDir") && os.TempDir() != "/tmp" {
		x.Log.Printf("%s: %s", colors.PrintWarn(), "Non-standard value specified for temporary directory")
	}

	replacements := map[string]any{
		"%tmp%":    filepath.Clean(run_manager.RuntimeDir()),
		"%path%":   *x.Config.Env.NodePath,
		"%stdout%": "_1STDout",
		"%stderr%": "_2STDerr",
		"%1%":      "_1STDout",
		"%2%":      "_2STDerr",
	}

	processConfig(&x.Config.Conf, replacements)

	if !slices.Contains(logs.Levels.Available, *x.Config.Conf.Log.Level) {
		if !slices.Contains(*x.Config.Conf.DisableWarnings, "--WUndefLogLevel") {
			x.Log.Printf("%s: %s", colors.PrintWarn(), fmt.Sprintf("Unknown logging level %s, fallback level: %s", *x.Config.Conf.Log.Level, logs.Levels.Fallback))
		}
		x.Config.Conf.Log.Level = &logs.Levels.Fallback
	}
}

// The hook is responsible for outputting the 
// final config and asking for confirmation.
func InitConfigPrintHook(ctx context.Context, cs *corestate.CoreState, x *app.AppX) {
	if *x.Config.Conf.Node.ShowConfig {
		fmt.Printf("Configuration from %s:\n", x.Config.CMDLine.Run.ConfigPath)
		x.Config.Print(x.Config.Conf)

		fmt.Printf("Environment:\n")
		x.Config.Print(x.Config.Env)

		if cs.UUID32 != "" && !askConfirm("Is that ok?", true) {
			x.Log.Printf("Cancel launch")
			NodeApp.CallFallback(ctx)
		}
	}

	x.Log.Printf("Starting \"%s\" node", *x.Config.Conf.Node.Name)
}

func InitSLogHook(_ context.Context, cs *corestate.CoreState, x *app.AppX) {
	cs.Stage = corestate.StageReady
	x.Log.SetPrefix(colors.SetGreen(fmt.Sprintf("(%s) ", cs.Stage)))

	x.SLog = new(slog.Logger)
	newSlog, err := logs.SetupLogger(x.Config.Conf.Log)
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	*x.SLog = *newSlog
}

// The method goes through the entire config structure through 
// reflection and replaces string fields with the required ones.
func processConfig(conf any, replacements map[string]any) error {
	val := reflect.ValueOf(conf)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if field.CanAddr() && field.CanSet() {
				if err := processConfig(field.Addr().Interface(), replacements); err != nil {
					return err
				}
			}
		}

	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if elem.CanAddr() && elem.CanSet() {
				if err := processConfig(elem.Addr().Interface(), replacements); err != nil {
					return err
				}
			}
		}

	case reflect.Map:
		for _, key := range val.MapKeys() {
			elem := val.MapIndex(key)
			if elem.CanInterface() {
				newVal := reflect.New(elem.Type()).Elem()
				newVal.Set(elem)

				if err := processConfig(newVal.Addr().Interface(), replacements); err != nil {
					return err
				}

				val.SetMapIndex(key, newVal)
			}
		}

	case reflect.String:
		str := val.String()

		if replacement, exists := replacements[str]; exists {
			if err := setValue(val, replacement); err != nil {
				return fmt.Errorf("failed to set %q: %v", str, err)
			}
		} else {
			for placeholder, replacement := range replacements {
				if strings.Contains(str, placeholder) {
					replacementStr, err := toString(replacement)
					if err != nil {
						return fmt.Errorf("invalid replacement for %q: %v", placeholder, err)
					}
					newStr := strings.ReplaceAll(str, placeholder, replacementStr)
					val.SetString(newStr)
				}
			}
		}

	case reflect.Ptr:
		if !val.IsNil() {
			elem := val.Elem()
			if elem.Kind() == reflect.String {
				str := elem.String()
				if replacement, exists := replacements[str]; exists {
					strVal, err := toString(replacement)
					if err != nil {
						return fmt.Errorf("cannot convert replacement to string: %v", err)
					}
					elem.SetString(strVal)
				} else {
					for placeholder, replacement := range replacements {
						if strings.Contains(str, placeholder) {
							replacementStr, err := toString(replacement)
							if err != nil {
								return fmt.Errorf("invalid replacement for %q: %v", placeholder, err)
							}
							newStr := strings.ReplaceAll(str, placeholder, replacementStr)
							elem.SetString(newStr)
						}
					}
				}
			} else {
				return processConfig(elem.Addr().Interface(), replacements)
			}
		}
	}
	return nil
}

func setValue(val reflect.Value, replacement any) error {
	if !val.CanSet() {
		return fmt.Errorf("value is not settable")
	}

	replacementVal := reflect.ValueOf(replacement)
	if replacementVal.Type().AssignableTo(val.Type()) {
		val.Set(replacementVal)
		return nil
	}

	if val.Kind() == reflect.String {
		str, err := toString(replacement)
		if err != nil {
			return fmt.Errorf("cannot convert replacement to string: %v", err)
		}
		val.SetString(str)
		return nil
	}

	return fmt.Errorf("type mismatch: cannot assign %T to %v", replacement, val.Type())
}

func toString(v any) (string, error) {
	switch s := v.(type) {
	case string:
		return s, nil
	case fmt.Stringer:
		return s.String(), nil
	default:
		return fmt.Sprint(v), nil
	}
}

func askConfirm(prompt string, defaultYes bool) bool {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	fmt.Print(prompt)
	if defaultYes {
		fmt.Printf(" (%s/%s): ", colors.SetBrightGreen("Y"), colors.SetBrightRed("n"))
	} else {
		fmt.Printf(" (%s/%s): ", colors.SetBrightGreen("n"), colors.SetBrightRed("Y"))
	}

	inputChan := make(chan string, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		inputChan <- text
	}()

	select {
	case <-ctx.Done():
		fmt.Println("")
		NodeApp.CallFallback(ctx)
		os.Exit(3)
	case text := <-inputChan:
		text = strings.TrimSpace(strings.ToLower(text))
		if text == "" {
			return defaultYes
		}
		if text == "y" || text == "yes" {
			return true
		}
		return false
	}
	return defaultYes
}
