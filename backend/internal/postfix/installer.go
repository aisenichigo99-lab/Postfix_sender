package postfix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type InstallRequest struct {
	MailName          string      `json:"mail_name"`
	ConfigurationType string      `json:"configuration_type"`
	Relay             RelayConfig `json:"relay"`
}

var domainPattern = regexp.MustCompile(
	`(?i)^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`,
)

type InstallResult struct {
	Success bool     `json:"success"`
	Logs    []string `json:"logs"`
}

func validateRequest(req InstallRequest) error {
	req.MailName = strings.TrimSpace(req.MailName)

	if !domainPattern.MatchString(req.MailName) {
		return errors.New("nom de domaine invalide")
	}

	if req.ConfigurationType != "Internet with smarthost" {
		return errors.New("type de configuration non autorisé")
	}

	if err := validateRelay(req.Relay); err != nil {
		return fmt.Errorf("configuration du relais : %w", err)
	}

	return nil
}

func readDistribution() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("lecture de /etc/os-release : %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			value := strings.Trim(
				strings.TrimSpace(strings.TrimPrefix(line, "ID=")),
				`"'`,
			)
			return strings.ToLower(value), nil
		}
	}

	return "", errors.New("distribution Linux inconnue")
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func Install(
	ctx context.Context,
	req InstallRequest,
	report func(string),
) (InstallResult, error) {
	result := InstallResult{
		Success: false,
		Logs:    []string{},
	}

	log := func(message string) {
		result.Logs = append(result.Logs, message)

		if report != nil {
			report(message)
		}
	}

	if os.Geteuid() != 0 {
		return result, errors.New("l'installation nécessite les privilèges root")
	}

	req.MailName = strings.TrimSpace(req.MailName)
	req.Relay.Host = strings.TrimSpace(req.Relay.Host)

	if err := validateRequest(req); err != nil {
		return result, err
	}

	distro, err := readDistribution()
	if err != nil {
		return result, err
	}

	if distro != "debian" && distro != "ubuntu" {
		return result, fmt.Errorf(
			"distribution non prise en charge : %s",
			distro,
		)
	}

	for _, command := range []string{
		"apt-get",
		"debconf-set-selections",
		"dpkg-query",
		"systemctl",
	} {
		if !commandExists(command) {
			return result, fmt.Errorf("commande requise introuvable : %s", command)
		}
	}

	// Refuser les autres MTA afin de ne pas modifier un serveur
	// de messagerie déjà géré par un autre logiciel.
	for _, service := range []string{
		"exim4",
		"sendmail",
		"opensmtpd",
		"nullmailer",
	} {
		output, _ := runCommand(
			ctx,
			"dpkg-query",
			[]string{"-W", "-f=${Status}", service},
			nil,
			"",
			nil,
		)

		if strings.Contains(output, "install ok installed") {
			return result, fmt.Errorf(
				"un autre MTA est installé : %s ; vérifiez sa configuration avant de continuer",
				service,
			)
		}
	}

	log("Distribution compatible détectée : " + distro)
	log("Préparation de la configuration Postfix...")

	relayHost := fmt.Sprintf(
		"[%s]:%d",
		strings.TrimSuffix(req.Relay.Host, "."),
		req.Relay.Port,
	)

	debconf := fmt.Sprintf(
		"postfix postfix/mailname string %s\n"+
			"postfix postfix/main_mailer_type select %s\n"+
			"postfix postfix/relayhost string %s\n",
		req.MailName,
		req.ConfigurationType,
		relayHost,
	)

	_, err = runCommand(
		ctx,
		"debconf-set-selections",
		nil,
		nil,
		debconf,
		nil,
	)
	if err != nil {
		return result, fmt.Errorf("configuration debconf : %w", err)
	}

	log("Mise à jour des dépôts APT...")

	updateCtx, updateCancel := context.WithTimeout(ctx, 10*time.Minute)
	_, err = runCommand(
		updateCtx,
		"apt-get",
		[]string{"update"},
		[]string{"DEBIAN_FRONTEND=noninteractive"},
		"",
		func(line string) { log(line) },
	)
	updateCancel()

	if err != nil {
		return result, fmt.Errorf("mise à jour APT : %w", err)
	}

	log("Installation de Postfix, SASL et des certificats...")

	installCtx, installCancel := context.WithTimeout(ctx, 15*time.Minute)
	_, err = runCommand(
		installCtx,
		"apt-get",
		[]string{
			"install",
			"-y",
			"postfix",
			"libsasl2-modules",
			"ca-certificates",
		},
		[]string{"DEBIAN_FRONTEND=noninteractive"},
		"",
		func(line string) { log(line) },
	)
	installCancel()

	if err != nil {
		return result, fmt.Errorf("installation des paquets : %w", err)
	}

	log("Vérification des composants Postfix...")

	if !commandExists("postconf") ||
		!commandExists("postmap") ||
		!commandExists("postfix") {
		return result, errors.New("les outils Postfix sont incomplets après installation")
	}

	output, err := runCommand(
		ctx,
		"postconf",
		[]string{"-A"},
		nil,
		"",
		nil,
	)
	if err != nil {
		return result, fmt.Errorf("vérification SASL : %w", err)
	}

	if !strings.Contains(strings.ToLower(output), "cyrus") {
		return result, errors.New("support SASL Cyrus absent dans Postfix")
	}

	log("Support SASL Cyrus détecté.")

	log("Configuration du relais SMTP...")

	if err := ConfigureRelay(ctx, req.Relay, func(message string) {
		log(message)
	}); err != nil {
		return result, fmt.Errorf("configuration du relais SMTP : %w", err)
	}

	log("Activation du service Postfix...")

	_, err = runCommand(
		ctx,
		"systemctl",
		[]string{"enable", "--now", "postfix"},
		nil,
		"",
		func(line string) { log(line) },
	)
	if err != nil {
		return result, fmt.Errorf("activation du service Postfix : %w", err)
	}

	output, err = runCommand(
		ctx,
		"systemctl",
		[]string{"is-active", "postfix"},
		nil,
		"",
		nil,
	)
	if err != nil || strings.TrimSpace(output) != "active" {
		return result, errors.New("le service Postfix n'est pas actif")
	}

	log("Postfix est installé et actif.")
	log("Configuration du relais SMTP terminée.")

	result.Success = true
	return result, nil
}
