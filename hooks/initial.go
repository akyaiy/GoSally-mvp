package hooks

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/core/run_manager"
	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"gopkg.in/ini.v1"
)

var Compositor *config.Compositor = config.NewCompositor()

func Init0Hook(cs *corestate.CoreState, x *app.AppX) {
	x.Config = Compositor
	x.Log.SetOutput(os.Stdout)
	x.Log.SetPrefix(logs.SetBrightBlack(fmt.Sprintf("(%s) ", cs.Stage)))
	x.Log.SetFlags(log.Ldate | log.Ltime)
}

// First stage: pre-init
func Init1Hook(cs *corestate.CoreState, x *app.AppX) {
	*cs = *corestate.NewCorestate(&corestate.CoreState{
		UUID32DirName:      "uuid",
		NodeBinName:        filepath.Base(os.Args[0]),
		NodeVersion:        config.NodeVersion,
		MetaDir:            "./.meta",
		Stage:              corestate.StagePreInit,
		StartTimestampUnix: time.Now().Unix(),
	})
}

func Init2Hook(cs *corestate.CoreState, x *app.AppX) {
	x.Log.SetPrefix(logs.SetBlue(fmt.Sprintf("(%s) ", cs.Stage)))

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

func Init3Hook(cs *corestate.CoreState, x *app.AppX) {
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
}

func Init4Hook(cs *corestate.CoreState, x *app.AppX) {
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
func Init5Hook(cs *corestate.CoreState, x *app.AppX) {
	cs.Stage = corestate.StagePostInit
	x.Log.SetPrefix(logs.SetYellow(fmt.Sprintf("(%s) ", cs.Stage)))

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

func Init6Hook(cs *corestate.CoreState, x *app.AppX) {
	if !slices.Contains(*x.Config.Conf.DisableWarnings, "--WNonStdTmpDir") && os.TempDir() != "/tmp" {
		x.Log.Printf("%s: %s", logs.PrintWarn(), "Non-standard value specified for temporary directory")
	}
	if strings.Contains(*x.Config.Conf.Log.OutPath, `%tmp%`) {
		replaced := strings.ReplaceAll(*x.Config.Conf.Log.OutPath, "%tmp%", filepath.Clean(run_manager.RuntimeDir()))
		x.Config.Conf.Log.OutPath = &replaced
	}
}

func Init7Hook(cs *corestate.CoreState, x *app.AppX) {
	cs.Stage = corestate.StageReady
	x.Log.SetPrefix(logs.SetGreen(fmt.Sprintf("(%s) ", cs.Stage)))

	x.SLog = new(slog.Logger)
	newSlog, err := logs.SetupLogger(x.Config.Conf.Log)
	if err != nil {
		_ = run_manager.Clean()
		x.Log.Fatalf("Unexpected failure: %s", err.Error())
	}
	*x.SLog = *newSlog
}

// repl := map[string]string{
// 			"tmp": filepath.Clean(run_manager.RuntimeDir()),
// 		}
// 		re := regexp.MustCompile(`%(\w+)%`)
// 		result := re.ReplaceAllStringFunc(x.Config.Conf.Log.OutPath, func(match string) string {
// 			sub := re.FindStringSubmatch(match)
// 			if len(sub) < 2 {
// 				return match
// 			}
// 			key := sub[1]
// 			if val, ok := repl[key]; ok {
// 				return val
// 			}
// 			return match
// 		})

// 		if strings.Contains(x.Config.Conf.Log.OutPath, "%tmp%") {
// 			relPath := strings.TrimPrefix(result, filepath.Clean(run_manager.RuntimeDir()))
// 			if err := run_manager.SetDir(relPath); err != nil {
// 				_ = run_manager.Clean()
// 				x.Log.Fatalf("Unexpected failure: %s", err.Error())
// 			}
// 		}
