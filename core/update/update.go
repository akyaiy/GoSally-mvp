package update

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/akyaiy/GoSally-mvp/core/config"
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
	Log    slog.Logger
	Config *config.ConfigConf
}

func NewUpdater(log slog.Logger, cfg *config.ConfigConf) *Updater {
	return &Updater{
		Log:    log,
		Config: cfg,
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
				cur = 0 // или можно обработать ошибку иначе
			}
			curPart = cur
		} else {
			curPart = 0 // Если части в current меньше, считаем недостающие нулями
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
		// если равны — идём дальше
	}
	return false // все части равны, значит не новее
}

// if len(currentParts) >= 1 && len(latestParts) >= 1 {
// 		if currentParts[0] < latestParts[0] {
// 			if len(currentParts) < 2 || len(latestParts) < 2 {
// 				if currentParts[1] < latestParts[1] {
// 					return true
// 				}
// 				if currentParts[1] > latestParts[1] {
// 					return false
// 				}
// 		}
// 		if currentParts[0] > latestParts[0] {
// 			return false
// 		}
// 	}

// GetCurrentVersion reads the current version from the version file and returns it along with the branch.
func (u *Updater) GetCurrentVersion() (Version, Branch, error) {
	version, branch, err := splitVersionString(string(config.GetUpdateConsts().GetNodeVersion()))
	if err != nil {
		u.Log.Error("Failed to parse version string", slog.String("version", string(config.GetUpdateConsts().GetNodeVersion())), slog.String("error", err.Error()))
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
	repoURL := u.Config.Updates.RepositoryURL
	if repoURL == "" {
		u.Log.Error("RepositoryURL is empty in config")
		return "", "", errors.New("repository URL is empty")
	}
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		u.Log.Error("RepositoryURL does not start with http:// or https://", slog.String("RepositoryURL", repoURL))
		return "", "", errors.New("repository URL must start with http:// or https://")
	}
	response, err := http.Get(repoURL + "/" + config.GetUpdateConsts().GetActualFileName())
	if err != nil {
		u.Log.Error("Failed to fetch latest version", slog.String("error", err.Error()))
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		u.Log.Error("Failed to fetch latest version", slog.Int("status", response.StatusCode))
		return "", "", errors.New("failed to fetch latest version, status code: " + http.StatusText(response.StatusCode))
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		u.Log.Error("Failed to read latest version response", slog.String("error", err.Error()))
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
			u.Log.Error("Failed to parse version string", slog.String("version", string(line)), slog.String("error", err.Error()))
			return "", "", err
		}
		if branch == updateBranch {
			return Version(version), Branch(branch), nil
		}
	}
	u.Log.Warn("No version found for branch", slog.String("branch", string(updateBranch)))
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
	if !(u.Config.UpdatesEnabled) {
		return errors.New("updates are disabled in config, skipping update")
	}
	downloadPath, err := os.MkdirTemp("", "*-gs-up")
	if err != nil {
		return errors.New("failed to create temp dir " + err.Error())
	}
	_, currentBranch, err := u.GetCurrentVersion()
	if err != nil {
		return errors.New("failed to get current version: " + err.Error())
	}
	latestVersion, latestBranch, err := u.GetLatestVersion(currentBranch)
	if err != nil {
		return errors.New("failed to get latest version: " + err.Error())
	}
	updateArchiveName := config.GetUpdateConsts().GetUpdateArchiveName() + ".v" + string(latestVersion) + "-" + string(latestBranch)
	updateDest := u.Config.Updates.RepositoryURL + "/" + updateArchiveName + ".tar.gz"
	resp, err := http.Get(updateDest)
	if err != nil {
		return errors.New("failed to fetch latest version archive: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.New("failed to fetch latest version archive: status " + resp.Status + ", body: " + string(body))
	}
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.New("failed to create gzip reader: " + err.Error())
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // archive is fully read
		}
		if err != nil {
			return errors.New("failed to read tar header: " + err.Error())
		}

		targetPath := filepath.Join(downloadPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Создаём директорию
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return errors.New("failed to create directory: " + err.Error())
			}
		case tar.TypeReg:
			// Создаём директорию, если её ещё нет
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return errors.New("failed to create directory for file: " + err.Error())
			}
			// Создаём файл
			outFile, err := os.Create(targetPath)
			if err != nil {
				return errors.New("failed to create file: " + err.Error())
			}
			// Копируем содержимое
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return errors.New("failed to copy file content: " + err.Error())
			}
			outFile.Close()
		default:
			return errors.New("unsupported tar entry type: " + string(header.Typeflag))
		}
	}
	return u.InstallAndRestart(filepath.Join(downloadPath, updateArchiveName, "node"))
}

func (u *Updater) InstallAndRestart(newBinaryPath string) error {
	nodePath := os.Getenv("NODE_PATH")
	if nodePath == "" {
		return errors.New("NODE_PATH environment variable is not set")
	}
	installDir := filepath.Join(nodePath, "bin")
	targetPath := filepath.Join(installDir, "node")

	// Копируем новый бинарник
	input, err := os.Open(newBinaryPath)
	if err != nil {
		return err
	}

	output, err := os.Create(targetPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, input); err != nil {
		return err
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return errors.New("failed to chmod file: " + err.Error())
	}

	input.Close()
	toClean := regexp.MustCompile(`^(/tmp/\d+-gs-up/)`).FindStringSubmatch(newBinaryPath)
	if len(toClean) > 1 {
		os.RemoveAll(toClean[0])
	}
	output.Close()
	// Запускаем новый процесс
	u.Log.Info("Launching new version...", slog.String("path", targetPath))
	cmd := exec.Command(targetPath, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = nil
	if err = cmd.Start(); err != nil {
		return err
	}
	u.Log.Info("Shutting down")
	os.Exit(0)
	return errors.New("failed to shutdown the process")
}

// func (u *Updater) Update() error {
// 	if !(u.Config.UpdatesEnabled && u.Config.Updates.AllowUpdates && u.Config.Updates.AllowDowngrades) {
// 		u.Log.Info("Updates are disabled in config, skipping update")
// 		return nil
// 	}
// 	wantedVersion := u.Config.Updates.WantedVersion
// 	_, wantedBranch, _ := splitVersionString(wantedVersion)
// 	newVersion, newBranch, err := u.GetLatestVersion(wantedBranch)
// 	if err != nil {
// 		return err
// 	}
// 	if wantedBranch != newBranch {
// 		u.Log.Info("Wanted version branch does not match latest version branch: updating wanted branch",
// 			slog.String("wanted_branch", string(wantedBranch)),
// 			slog.String("latest_branch", string(newBranch)),
// 		)
// 	}
// }
