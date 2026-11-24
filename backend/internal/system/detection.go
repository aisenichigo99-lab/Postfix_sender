package system

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Check struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Status  string `json:"status"` // info, warning, blocking, ok
	Message string `json:"message"`
}

type SystemInfo struct {
	Distribution     string   `json:"distribution"`
	Version          string   `json:"version"`
	PrettyName       string   `json:"pretty_name"`
	Architecture     string   `json:"architecture"`
	PackageManager   string   `json:"package_manager"`
	IsRoot           bool     `json:"is_root"`
	Hostname         string   `json:"hostname"`
	MailName         string   `json:"mail_name"`
	PostfixInstalled bool     `json:"postfix_installed"`
	PostfixStatus    string   `json:"postfix_status"`
	MTAConflicts     []string `json:"mta_conflicts"`
	SystemdAvailable bool     `json:"systemd_available"`
	Supported        bool     `json:"supported"`
	Ready            bool     `json:"ready"`
	Checks           []Check  `json:"checks"`
}

func readOSRelease() map[string]string {
	result := make(map[string]string)

	file, err := os.Open("/etc/os-release")
	if err != nil {
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		value = strings.TrimSpace(value)

		// Les valeurs de /etc/os-release peuvent être entre guillemets.
		if strings.HasPrefix(value, `"`) {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			}
		} else {
			value = strings.Trim(value, "'")
		}

		result[key] = value
	}

	return result
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	output, err := exec.CommandContext(commandCtx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func packageInstalled(ctx context.Context, name string) bool {
	if commandExists("dpkg-query") {
		output, err := runCommand(
			ctx,
			"dpkg-query",
			"-W",
			"-f=${db:Status-Status}",
			name,
		)

		return err == nil && output == "installed"
	}

	if commandExists("rpm") {
		_, err := runCommand(ctx, "rpm", "-q", name)
		return err == nil
	}

	return false
}

func detectPostfix(ctx context.Context) (bool, string) {
	installed := packageInstalled(ctx, "postfix")

	if !installed {
		return false, "not_installed"
	}

	if !commandExists("systemctl") {
		return true, "unknown"
	}

	output, err := runCommand(ctx, "systemctl", "is-active", "postfix")

	if err != nil && output == "" {
		return true, "unknown"
	}

	switch output {
	case "active":
		return true, "active"
	case "inactive":
		return true, "inactive"
	case "failed":
		return true, "failed"
	default:
		if output == "" {
			return true, "unknown"
		}
		return true, output
	}
}

func detectMTAConflicts(ctx context.Context) []string {
	candidates := []string{
		"exim4",
		"sendmail",
		"nullmailer",
		"opensmtpd",
		"ssmtp",
	}

	var conflicts []string

	for _, candidate := range candidates {
		if packageInstalled(ctx, candidate) {
			conflicts = append(conflicts, candidate)
		}
	}

	return conflicts
}

func readMailName() string {
	data, err := os.ReadFile("/etc/mailname")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func addCheck(checks *[]Check, id, label, status, message string) {
	*checks = append(*checks, Check{
		ID:      id,
		Label:   label,
		Status:  status,
		Message: message,
	})
}

func Detect() SystemInfo {
	return DetectContext(context.Background())
}

func DetectContext(ctx context.Context) SystemInfo {
	osInfo := readOSRelease()

	id := strings.ToLower(osInfo["ID"])

	packageManager := "unknown"

	switch id {
	case "ubuntu", "debian":
		packageManager = "apt"
	case "fedora", "rhel", "centos", "rocky", "almalinux":
		packageManager = "dnf"
	case "arch":
		packageManager = "pacman"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	mailName := readMailName()

	installed, postfixStatus := detectPostfix(ctx)
	conflicts := detectMTAConflicts(ctx)
	systemdAvailable := commandExists("systemctl")

	supported := id == "ubuntu" || id == "debian"
	aptAvailable := commandExists("apt-get")

	checks := make([]Check, 0)

	if supported {
		addCheck(&checks, "os", "Operating system", "ok",
			osInfo["PRETTY_NAME"])
	} else {
		addCheck(&checks, "os", "Operating system", "blocking",
			"Cette distribution n'est pas encore prise en charge.")
	}

	if packageManager == "apt" && aptAvailable {
		addCheck(&checks, "package_manager", "Package manager", "ok",
			"APT est disponible.")
	} else {
		addCheck(&checks, "package_manager", "Package manager", "blocking",
			"Le gestionnaire de paquets attendu n'est pas disponible.")
	}

	if hostname != "" && hostname != "localhost" && hostname != "localhost.localdomain" {
		addCheck(&checks, "hostname", "Hostname", "ok", hostname)
	} else {
		addCheck(&checks, "hostname", "Hostname", "warning",
			"Le hostname est absent ou semble être une valeur locale par défaut.")
	}

	if mailName != "" {
		addCheck(&checks, "mailname", "Mail name", "ok", mailName)
	} else {
		addCheck(&checks, "mailname", "Mail name", "warning",
			"Le fichier /etc/mailname est absent ou vide.")
	}

	if systemdAvailable {
		addCheck(&checks, "systemd", "Service manager", "ok",
			"systemctl est disponible.")
	} else {
		addCheck(&checks, "systemd", "Service manager", "warning",
			"systemctl n'a pas été trouvé.")
	}

	if installed {
		addCheck(&checks, "postfix", "Postfix", "warning",
			"Postfix est déjà installé. État du service : "+postfixStatus)
	} else {
		addCheck(&checks, "postfix", "Postfix", "ok",
			"Postfix n'est pas encore installé.")
	}

	if len(conflicts) > 0 {
		addCheck(&checks, "mta_conflicts", "Other mail servers", "blocking",
			"Autres MTA détectés : "+strings.Join(conflicts, ", "))
	} else {
		addCheck(&checks, "mta_conflicts", "Other mail servers", "ok",
			"Aucun MTA concurrent connu n'a été détecté.")
	}

	ready := supported && aptAvailable && len(conflicts) == 0

	return SystemInfo{
		Distribution:     id,
		Version:          osInfo["VERSION_ID"],
		PrettyName:       osInfo["PRETTY_NAME"],
		Architecture:     runtime.GOARCH,
		PackageManager:   packageManager,
		IsRoot:           os.Geteuid() == 0,
		Hostname:         hostname,
		MailName:         mailName,
		PostfixInstalled: installed,
		PostfixStatus:    postfixStatus,
		MTAConflicts:     conflicts,
		SystemdAvailable: systemdAvailable,
		Supported:        supported,
		Ready:            ready,
		Checks:           checks,
	}
}
