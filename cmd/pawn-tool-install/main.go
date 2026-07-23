// Command pawn-tool-install installs one verified PawnKit tool archive.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pawnkit/pawn-actions/toolinstall"
)

const (
	maxDownloadSize  = 200 << 20
	maxVersionOutput = 64 << 10
)

var (
	binaryPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	versionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
)

type options struct {
	binary      string
	version     string
	archiveURL  string
	checksum    string
	destination string
}

func main() {
	var settings options
	flag.StringVar(&settings.binary, "binary", "", "tool executable name")
	flag.StringVar(&settings.version, "version", "", "exact tool version")
	flag.StringVar(&settings.archiveURL, "url", "", "release archive URL")
	flag.StringVar(&settings.checksum, "sha256", "", "release archive SHA-256")
	flag.StringVar(&settings.destination, "destination", "", "installation directory")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: pawn-tool-install -binary NAME -version VERSION -url URL -sha256 HASH -destination DIR")
		os.Exit(2)
	}
	if err := run(context.Background(), settings); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, settings options) error {
	archiveName, err := validateOptions(settings)
	if err != nil {
		return err
	}
	binaryName := settings.binary
	if filepath.Ext(settings.destination) == ".exe" {
		return errors.New("destination must be a directory")
	}
	if runtimeBinary := executableName(settings.binary); runtimeBinary != settings.binary {
		binaryName = runtimeBinary
	}
	destinationBinary := filepath.Join(settings.destination, binaryName)
	marker := filepath.Join(settings.destination, ".archive-sha256")
	if cached(settings, destinationBinary, marker) {
		return writeOutputs(settings, destinationBinary)
	}

	content, err := download(ctx, settings.archiveURL)
	if err != nil {
		return err
	}
	want, _ := hex.DecodeString(settings.checksum)
	actual := sha256.Sum256(content)
	if subtle.ConstantTimeCompare(actual[:], want) != 1 {
		return errors.New("tool archive checksum mismatch")
	}
	files, err := toolinstall.Extract(archiveName, content, settings.binary)
	if err != nil {
		return err
	}
	if err := install(settings, files, binaryName); err != nil {
		return err
	}
	if err := checkVersion(ctx, destinationBinary, settings.version); err != nil {
		return err
	}
	if err := os.WriteFile(marker, []byte(settings.checksum+"\n"), 0o600); err != nil {
		return fmt.Errorf("write checksum marker: %w", err)
	}
	return writeOutputs(settings, destinationBinary)
}

func validateOptions(settings options) (string, error) {
	if !binaryPattern.MatchString(settings.binary) {
		return "", errors.New("binary must be a simple lowercase name")
	}
	if !versionPattern.MatchString(settings.version) {
		return "", errors.New("version must be an exact semantic version")
	}
	checksum, err := hex.DecodeString(settings.checksum)
	if err != nil || len(checksum) != sha256.Size || settings.checksum != strings.ToLower(settings.checksum) {
		return "", errors.New("sha256 must be 64 lowercase hexadecimal characters")
	}
	parsed, err := url.Parse(settings.archiveURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("url must be an HTTPS GitHub release archive")
	}
	want := "/pawnkit/"
	if !strings.HasPrefix(parsed.Path, want) || !strings.Contains(parsed.Path, "/releases/download/"+settings.version+"/") {
		return "", errors.New("url must match the requested PawnKit release")
	}
	if err := validateDestination(settings.destination); err != nil {
		return "", err
	}
	return pathBase(parsed.Path), nil
}

func validateDestination(name string) error {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	volume := filepath.VolumeName(absolute)
	root := volume + string(filepath.Separator)
	if name == "" || !filepath.IsAbs(name) || filepath.Clean(absolute) == filepath.Clean(root) {
		return errors.New("destination must be an absolute non-root directory")
	}
	return nil
}

func cached(settings options, binary, marker string) bool {
	content, err := os.ReadFile(marker) //nolint:gosec // The destination is an explicit action input.
	if err != nil || strings.TrimSpace(string(content)) != settings.checksum {
		return false
	}
	info, err := os.Lstat(binary)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func download(ctx context.Context, archiveURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if request.URL.Scheme != "https" {
				return errors.New("download redirected away from HTTPS")
			}
			return nil
		},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	request.Header.Set("User-Agent", "pawn-actions-tool-install")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download tool: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download tool: %s", response.Status)
	}
	if response.ContentLength > maxDownloadSize {
		return nil, errors.New("tool archive is too large")
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("read tool archive: %w", err)
	}
	if len(content) > maxDownloadSize {
		return nil, errors.New("tool archive is too large")
	}
	return content, nil
}

func install(settings options, files []toolinstall.File, binaryName string) error {
	parent := filepath.Dir(settings.destination)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("create tool cache: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".pawn-tool-*")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(staging)
	}()

	for _, file := range files {
		name := filepath.FromSlash(file.Name)
		if file.Name == settings.binary || file.Name == settings.binary+".exe" {
			name = binaryName
		}
		destination := filepath.Join(staging, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
			return fmt.Errorf("create tool directory: %w", err)
		}
		mode := os.FileMode(0o644)
		if name == binaryName {
			mode = 0o755
		}
		if err := os.WriteFile(destination, file.Data, mode); err != nil {
			return fmt.Errorf("write tool file: %w", err)
		}
	}
	if err := os.RemoveAll(settings.destination); err != nil {
		return fmt.Errorf("replace old tool: %w", err)
	}
	if err := os.Rename(staging, settings.destination); err != nil {
		return fmt.Errorf("activate tool: %w", err)
	}
	return nil
}

func checkVersion(ctx context.Context, binary, version string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var output limitedBuffer
	command := exec.CommandContext(checkCtx, binary, "--version") //nolint:gosec // The binary was selected from a verified archive.
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return fmt.Errorf("run tool version check: %w", err)
	}
	if !strings.Contains(output.String(), strings.TrimPrefix(version, "v")) {
		return fmt.Errorf("tool version output does not match %s", version)
	}
	return nil
}

func writeOutputs(settings options, binary string) error {
	fmt.Println(binary)
	if output := os.Getenv("GITHUB_OUTPUT"); output != "" {
		handle, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec
		if err != nil {
			return fmt.Errorf("open GitHub output: %w", err)
		}
		defer func() {
			_ = handle.Close()
		}()
		if _, err := fmt.Fprintf(handle, "path=%s\nversion=%s\n", filepath.Dir(binary), settings.version); err != nil {
			return fmt.Errorf("write GitHub output: %w", err)
		}
	}
	if pathFile := os.Getenv("GITHUB_PATH"); pathFile != "" {
		handle, err := os.OpenFile(pathFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec
		if err != nil {
			return fmt.Errorf("open GitHub path: %w", err)
		}
		defer func() {
			_ = handle.Close()
		}()
		if _, err := fmt.Fprintln(handle, filepath.Dir(binary)); err != nil {
			return fmt.Errorf("write GitHub path: %w", err)
		}
	}
	return nil
}

func executableName(binary string) string {
	if os.PathSeparator == '\\' {
		return binary + ".exe"
	}
	return binary
}

func pathBase(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

type limitedBuffer struct {
	data bytes.Buffer
}

func (buffer *limitedBuffer) Write(content []byte) (int, error) {
	if buffer.data.Len()+len(content) > maxVersionOutput {
		return 0, errors.New("version output is too large")
	}
	return buffer.data.Write(content)
}

func (buffer *limitedBuffer) String() string {
	return buffer.data.String()
}
