package postfix

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const postfixConfigDir = "/etc/postfix"

type RelayConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLSMode  string `json:"tls_mode"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type fileSnapshot struct {
	Path   string
	Data   []byte
	Mode   os.FileMode
	UID    int
	GID    int
	Exists bool
}

func validateRelay(req RelayConfig) error {
	host := strings.TrimSpace(req.Host)

	if host == "" || len(host) > 253 {
		return errors.New("nom d'hôte SMTP invalide")
	}

	if net.ParseIP(host) == nil {
		labels := strings.Split(strings.TrimSuffix(host, "."), ".")

		if len(labels) < 2 {
			return errors.New("le relais SMTP doit utiliser un nom DNS complet ou une adresse IP")
		}

		for _, label := range labels {
			if len(label) == 0 || len(label) > 63 {
				return errors.New("nom d'hôte SMTP invalide")
			}

			if label[0] == '-' || label[len(label)-1] == '-' {
				return errors.New("nom d'hôte SMTP invalide")
			}

			for _, r := range label {
				if !(r >= 'a' && r <= 'z') &&
					!(r >= 'A' && r <= 'Z') &&
					!(r >= '0' && r <= '9') &&
					r != '-' {
					return errors.New("caractère interdit dans le nom d'hôte SMTP")
				}
			}
		}
	}

	if req.Port != 587 && req.Port != 465 {
		return errors.New("port SMTP non pris en charge : utilisez 587 ou 465")
	}

	if req.Port == 587 && req.TLSMode != "starttls" {
		return errors.New("le port 587 nécessite le mode starttls")
	}

	if req.Port == 465 && req.TLSMode != "implicit" {
		return errors.New("le port 465 nécessite le mode implicit")
	}

	if strings.TrimSpace(req.Username) == "" {
		return errors.New("identifiant SMTP obligatoire")
	}

	if strings.TrimSpace(req.Password) == "" {
		return errors.New("mot de passe SMTP obligatoire")
	}

	for _, value := range []string{
		req.Host,
		req.Username,
		req.Password,
	} {
		if strings.ContainsAny(value, "\r\n\x00") {
			return errors.New("caractère interdit dans les paramètres SMTP")
		}
	}

	return nil
}

func snapshotFile(path string) (fileSnapshot, error) {
	snapshot := fileSnapshot{
		Path: path,
	}

	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, err
	}

	if !info.Mode().IsRegular() {
		return snapshot, fmt.Errorf("%s n'est pas un fichier régulier", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return snapshot, fmt.Errorf("impossible de lire les permissions de %s", path)
	}

	snapshot.Data = data
	snapshot.Mode = info.Mode().Perm()
	snapshot.UID = int(stat.Uid)
	snapshot.GID = int(stat.Gid)
	snapshot.Exists = true

	return snapshot, nil
}

func atomicWrite(
	path string,
	data []byte,
	mode os.FileMode,
	uid int,
	gid int,
) error {
	dir := filepath.Dir(path)

	temp, err := os.CreateTemp(dir, ".postfix-manager-*")
	if err != nil {
		return err
	}

	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	if err := os.Chown(tempPath, uid, gid); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func restoreSnapshot(snapshot fileSnapshot) error {
	if !snapshot.Exists {
		err := os.Remove(snapshot.Path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	return atomicWrite(
		snapshot.Path,
		snapshot.Data,
		snapshot.Mode,
		snapshot.UID,
		snapshot.GID,
	)
}

func createBackup(snapshot fileSnapshot) error {
	if !snapshot.Exists {
		return nil
	}

	backupPath := snapshot.Path + ".postfix-manager.bak"

	return atomicWrite(
		backupPath,
		snapshot.Data,
		0600,
		0,
		0,
	)
}

// Échappe les caractères qui pourraient être interprétés comme
// séparateurs dans un fichier de table Postfix.
func escapeMapValue(value string) string {
	var result strings.Builder

	for i, r := range value {
		if unicode.IsSpace(r) || r == '\\' || (i == 0 && r == '#') {
			result.WriteByte('\\')
		}
		result.WriteRune(r)
	}

	return result.String()
}

func runPostconf(ctx context.Context, setting string, report func(string)) error {
	_, err := runCommand(
		ctx,
		"postconf",
		[]string{"-e", setting},
		nil,
		"",
		report,
	)
	return err
}

func ConfigureRelay(
	ctx context.Context,
	req RelayConfig,
	report func(string),
) error {
	log := func(message string) {
		if report != nil {
			report(message)
		}
	}

	if os.Geteuid() != 0 {
		return errors.New("la configuration du relais nécessite les privilèges root")
	}

	// Normaliser le nom d'hôte avant sa validation.
	req.Host = strings.TrimSuffix(strings.TrimSpace(req.Host), ".")

	if err := validateRelay(req); err != nil {
		return err
	}

	if !commandExists("postconf") ||
		!commandExists("postmap") ||
		!commandExists("postfix") {
		return errors.New("Postfix ou ses outils sont introuvables")
	}

	// Vérifier que la table hash est disponible.
	mapTypes, err := runCommand(
		ctx,
		"postconf",
		[]string{"-m"},
		nil,
		"",
		nil,
	)
	if err != nil {
		return fmt.Errorf("détection des tables Postfix : %w", err)
	}

	hashAvailable := false
	for _, line := range strings.Split(mapTypes, "\n") {
		if strings.TrimSpace(line) == "hash" {
			hashAvailable = true
			break
		}
	}

	if !hashAvailable {
		return errors.New("le type de table hash n'est pas disponible dans Postfix")
	}

	mainPath := filepath.Join(postfixConfigDir, "main.cf")
	passwordPath := filepath.Join(postfixConfigDir, "sasl_passwd")
	databasePath := passwordPath + ".db"

	// Capturer les fichiers avant toute modification.
	mainSnapshot, err := snapshotFile(mainPath)
	if err != nil {
		return fmt.Errorf("sauvegarde de main.cf : %w", err)
	}

	passwordSnapshot, err := snapshotFile(passwordPath)
	if err != nil {
		return fmt.Errorf("sauvegarde des identifiants existants : %w", err)
	}

	databaseSnapshot, err := snapshotFile(databasePath)
	if err != nil {
		return fmt.Errorf("sauvegarde de la base SASL : %w", err)
	}

	// Créer des sauvegardes persistantes pour permettre une restauration
	// manuelle même après un redémarrage.
	if err := createBackup(mainSnapshot); err != nil {
		return fmt.Errorf("création de la sauvegarde main.cf : %w", err)
	}

	if err := createBackup(passwordSnapshot); err != nil {
		return fmt.Errorf("création de la sauvegarde SASL : %w", err)
	}

	rollback := func(cause error) error {
		log("Échec de configuration : restauration des fichiers précédents...")

		var restoreErrors []string

		for _, snapshot := range []fileSnapshot{
			mainSnapshot,
			passwordSnapshot,
			databaseSnapshot,
		} {
			if err := restoreSnapshot(snapshot); err != nil {
				restoreErrors = append(
					restoreErrors,
					fmt.Sprintf("%s : %v", snapshot.Path, err),
				)
			}
		}

		// Redémarrer Postfix avec un contexte indépendant.
		// Le contexte initial peut déjà avoir été annulé.
		if len(restoreErrors) == 0 {
			reloadCtx, cancel := context.WithTimeout(
				context.Background(),
				15*time.Second,
			)
			defer cancel()

			_, reloadErr := runCommand(
				reloadCtx,
				"systemctl",
				[]string{"restart", "postfix"},
				nil,
				"",
				nil,
			)

			if reloadErr != nil {
				restoreErrors = append(
					restoreErrors,
					"redémarrage après restauration : "+reloadErr.Error(),
				)
			}
		}

		if len(restoreErrors) > 0 {
			return fmt.Errorf(
				"%w ; restauration incomplète : %s",
				cause,
				strings.Join(restoreErrors, "; "),
			)
		}

		return cause
	}

	// Le fichier contient les secrets : écriture atomique en root:root 0600.
	relayKey := fmt.Sprintf("[%s]:%d", strings.TrimSuffix(req.Host, "."), req.Port)
	credentials := escapeMapValue(req.Username) + ":" + escapeMapValue(req.Password)
	passwordData := []byte(relayKey + " " + credentials + "\n")

	log("Écriture sécurisée des identifiants SMTP...")

	if err := atomicWrite(passwordPath, passwordData, 0600, 0, 0); err != nil {
		return rollback(fmt.Errorf("écriture du fichier SASL : %w", err))
	}

	log("Génération de la base d'authentification SASL...")

	_, err = runCommand(
		ctx,
		"postmap",
		[]string{"hash:" + passwordPath},
		nil,
		"",
		func(line string) {
			log(line)
		},
	)
	if err != nil {
		return rollback(fmt.Errorf("génération de la base SASL : %w", err))
	}

	if err := os.Chmod(databasePath, 0600); err != nil {
		return rollback(fmt.Errorf("protection de la base SASL : %w", err))
	}

	if err := os.Chown(databasePath, 0, 0); err != nil {
		return rollback(fmt.Errorf("propriété de la base SASL : %w", err))
	}

	log("Application des paramètres du relais SMTP...")

	wrapperMode := "no"
	if req.TLSMode == "implicit" {
		wrapperMode = "yes"
	}

	settings := []string{
		"relayhost=" + relayKey,
		"smtp_sasl_auth_enable=yes",
		"smtp_sasl_password_maps=hash:" + passwordPath,
		"smtp_sasl_security_options=noanonymous",
		"smtp_sasl_tls_security_options=noanonymous",
		"smtp_tls_security_level=encrypt",
		"smtp_tls_CApath=/etc/ssl/certs",
		"smtp_tls_wrappermode=" + wrapperMode,
	}

	for _, setting := range settings {
		if err := runPostconf(ctx, setting, func(line string) {
			log(line)
		}); err != nil {
			return rollback(fmt.Errorf("application d'un paramètre Postfix : %w", err))
		}
	}

	log("Vérification de la configuration Postfix...")

	_, err = runCommand(
		ctx,
		"postfix",
		[]string{"check"},
		nil,
		"",
		func(line string) {
			log(line)
		},
	)
	if err != nil {
		return rollback(fmt.Errorf("postfix check : %w", err))
	}

	log("Redémarrage de Postfix...")

	_, err = runCommand(
		ctx,
		"systemctl",
		[]string{"restart", "postfix"},
		nil,
		"",
		func(line string) {
			log(line)
		},
	)
	if err != nil {
		return rollback(fmt.Errorf("redémarrage de Postfix : %w", err))
	}

	log("Configuration du relais SMTP terminée.")

	return nil
}
