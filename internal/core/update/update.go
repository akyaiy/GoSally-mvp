package update

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/akyaiy/GoSally-mvp/internal/core/run_manager"
	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"golang.org/x/net/context"
)

const (
	UpdateBranchStable  = "stable"
	UpdateBranchDev     = "dev"
	UpdateBranchTesting = "testing"
)

type Version string
type Branch string
type IsNewUpdate bool

type UpdaterContract interface {
	CkeckUpdates() (IsNewUpdate, error)
	Update() error
	GetCurrentVersion() (Version, Branch, error)
	GetLatestVersion(updateBranch Branch) (Version, Branch, error)
}

type Updater struct {
	log    *log.Logger
	config *config.Conf
	env    *config.Env

	ctx    context.Context
	cancel context.CancelFunc
}

func NewUpdater(ctx context.Context, log *log.Logger, cfg *config.Conf, env *config.Env) *Updater {
	return &Updater{
		log:    log,
		config: cfg,
		env:    env,
		ctx:    ctx,
	}
}

func splitVersionString(versionStr string) (Version, Branch, error) {
	versionStr = strings.TrimSpace(versionStr)
	if !strings.HasPrefix(versionStr, "v") {
		return "", "unknown", errors.New("version string does not start with 'v'")
	}
	parts := strings.SplitN(versionStr[len("v"):], "-", 2)
	parts[0] = strings.TrimPrefix(parts[0], "version")
	if len(parts) != 2 {
		return Version(parts[0]), Branch("unknown"), errors.New("version string format invalid")
	}
	return Version(parts[0]), Branch(parts[1]), nil
}

// isVersionNewer compares two version strings and returns true if the current version is newer than the latest version.
func isVersionNewer(current, latest Version) bool {
	if current == latest {
		return false
	}

	currentParts := strings.Split(string(current), ".")
	latestParts := strings.Split(string(latest), ".")

	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}

	for i := 0; i < maxLen; i++ {
		var curPart, latPart int

		if i < len(currentParts) {
			cur, err := strconv.Atoi(currentParts[i])
			if err != nil {
				cur = 0
			}
			curPart = cur
		} else {
			curPart = 0
		}

		if i < len(latestParts) {
			lat, err := strconv.Atoi(latestParts[i])
			if err != nil {
				lat = 0
			}
			latPart = lat
		} else {
			latPart = 0
		}

		if curPart < latPart {
			return true
		}
		if curPart > latPart {
			return false
		}
	}
	return false
}

// GetCurrentVersion reads the current version from the version file and returns it along with the branch.
func (u *Updater) GetCurrentVersion() (Version, Branch, error) {
	version, branch, err := splitVersionString(string(config.NodeVersion))
	if err != nil {
		u.log.Printf("Failed to parse version string: %s", err.Error())
		return "", "", err
	}
	switch branch {
	case UpdateBranchDev, UpdateBranchStable, UpdateBranchTesting:
		return Version(version), Branch(branch), nil
	default:
		return Version(version), Branch("unknown"), nil
	}
}

func (u *Updater) GetLatestVersion(updateBranch Branch) (Version, Branch, error) {
	repoURL := u.config.Updates.RepositoryURL
	if repoURL == "" {
		u.log.Printf("Failed to get latest version: %s", "RepositoryURL is empty in config")
		return "", "", errors.New("repository URL is empty")
	}
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		u.log.Printf("Failed to get latest version: %s: %s", "RepositoryURL does not start with http:// or https:/", repoURL)
		return "", "", errors.New("repository URL must start with http:// or https://")
	}
	response, err := http.Get(repoURL + "/" + config.ActualFileName)
	if err != nil {
		u.log.Printf("Failed to fetch latest version: %s", err.Error())
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		u.log.Printf("Failed to fetch latest version: HTTP status %d", response.StatusCode)
		return "", "", errors.New("failed to fetch latest version, status code: " + http.StatusText(response.StatusCode))
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		u.log.Printf("Failed to read latest version response: %s", err.Error())
		return "", "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		version, branch, err := splitVersionString(string(line))
		if err != nil {
			u.log.Printf("Failed to parse version string: %s", err.Error())
			return "", "", err
		}
		if branch == updateBranch {
			return Version(version), Branch(branch), nil
		}
	}
	return "", "", errors.New("no version found for branch: " + string(updateBranch))
}

func (u *Updater) CkeckUpdates() (IsNewUpdate, error) {
	currentVersion, currentBranch, err := u.GetCurrentVersion()
	if err != nil {
		return false, err
	}
	latestVersion, latestBranch, err := u.GetLatestVersion(currentBranch)
	if err != nil {
		return false, err
	}
	if currentVersion == latestVersion && currentBranch == latestBranch {
		return false, nil
	}
	return true, nil
}

func (u *Updater) Update() error {
	if !u.config.Updates.UpdatesEnabled {
		return errors.New("updates are disabled in config, skipping update")
	}

	if err := run_manager.SetDir("update"); err != nil {
		return fmt.Errorf("failed to create update dir: %w", err)
	}

	downloadPath := filepath.Join(run_manager.RuntimeDir(), "update")

	_, currentBranch, err := u.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	latestVersion, latestBranch, err := u.GetLatestVersion(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	updateArchiveName := fmt.Sprintf("%s.v%s-%s", config.UpdateArchiveName, latestVersion, latestBranch)
	updateDest := fmt.Sprintf("%s/%s.%s", u.config.Updates.RepositoryURL, updateArchiveName, "tar.gz")

	resp, err := http.Get(updateDest)
	if err != nil {
		return fmt.Errorf("failed to fetch archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP status: %s, body: %s", resp.Status, body)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader error: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		relativeParts := strings.SplitN(header.Name, string(os.PathSeparator), 2)
		if len(relativeParts) < 2 {
			// It's either a top level directory or garbage.
			continue
		}
		cleanName := relativeParts[1]
		targetPath := filepath.Join(downloadPath, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir error: %w", err)
			}
		case tar.TypeReg:
			if err := run_manager.Set(filepath.Join("update", cleanName)); err != nil {
				return fmt.Errorf("set file error: %w", err)
			}
			f := run_manager.File(filepath.Join("update", cleanName))
			outFile, err := f.Open()
			if err != nil {
				return fmt.Errorf("open file error: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("copy file error: %w", err)
			}
			outFile.Close()
		default:
			return fmt.Errorf("unsupported tar type: %v", header.Typeflag)
		}
	}

	return u.InstallAndRestart()
}

func (u *Updater) InstallAndRestart() error {

	nodePath := u.env.NodePath
	if nodePath == "" {
		return errors.New("GS_NODE_PATH environment variable is not set")
	}
	installDir := filepath.Join(nodePath, "bin")
	targetPath := filepath.Join(installDir, "node")

	f := run_manager.File("update/node")
	input, err := f.Open()
	if err != nil {
		return fmt.Errorf("cannot open new binary: %w", err)
	}
	defer f.Close()

	output, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("cannot create target binary: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return fmt.Errorf("copy failed: %w", err)
	}
	output.Close()

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	u.log.Printf("Launching new version: path is %s", targetPath)
	// cmd := exec.Command(targetPath, os.Args[1:]...)
	// cmd.Env = os.Environ()
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// cmd.Stdin = os.Stdin
	args := os.Args
	args[0] = targetPath
	env := utils.SetEviron(os.Environ(), "GS_PARENT_PID=-1")

	if err := run_manager.Clean(); err != nil {
		return err
	}
	return syscall.Exec(targetPath, args, env)
	//u.cancel()

	// TODO: fix this crap and find a better way to update without errors
	// for {
	// 	_, err := run_manager.Get("run.lock")
	// 	if err != nil {
	// 		break
	// 	}
	// }

	// return cmd.Start()
}

func (u *Updater) Shutdownfunc(f context.CancelFunc) {
	u.cancel = f
}
