package utils

import (
	"log"
	"log/slog"
	"os"
	"path"

	"github.com/google/uuid"
)

// TODO: this could be a conf option ?
var tempWorkDir = "tmp"
var logsDir = "logs"

// get the directory of temp work dirs
func basePath(dir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(cwd, dir)
}

// ensure the path exists
func ensurePath(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

}

// create a temp stdio files for the request
func CreateTempStdioFiles(reqID uuid.UUID) (*os.File, *os.File) {
	runDir := path.Join(basePath(tempWorkDir), reqID.String())

	stdout, err := os.OpenFile(path.Join(runDir, "stdout.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := os.OpenFile(path.Join(runDir, "stderr.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return stdout, stderr
}

// read the data from the temp stdio files then closes them
// returns the data as strings
func CleanupTempStdioFiles(stdout *os.File, stderr *os.File) (string, string) {
	stdoutPath := stdout.Name()
	stderrPath := stderr.Name()

	stdout.Close()
	stderr.Close()

	stdoutLogs, err := os.ReadFile(stdoutPath)
	if err != nil {
		slog.Error("error reading stdout", "error", err)
	}

	stderrLogs, err := os.ReadFile(stderrPath)
	if err != nil {
		slog.Error("error reading stderr", "error", err)
	}

	return string(stdoutLogs), string(stderrLogs)

}

// create a temp work dir for the request
func CreateTemporyWorkDir(reqID uuid.UUID) (string, error) {
	runDir := path.Join(basePath(tempWorkDir), reqID.String())

	// make directory
	err := os.MkdirAll(runDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return runDir, nil
}

// remove the temp work dir
func CleanupTemporaryWorkDir(reqID uuid.UUID) bool {
	runDir := path.Join(basePath(tempWorkDir), reqID.String())

	if err := os.RemoveAll(runDir); err != nil {
		return false
	}

	return true
}

// persistant logs
// get append only stdio files by app id
func GetStdioByAppID(appID uuid.UUID) (*os.File, *os.File, error) {
	base := path.Join(basePath(logsDir), appID.String())

	ensurePath(base)

	stdoutPath := path.Join(base, "stdout.log")
	stderrPath := path.Join(base, "stderr.log")

	stdout, err := os.OpenFile(stdoutPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	stderr, err := os.OpenFile(stderrPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	return stdout, stderr, nil
}

// read the data in stdio files to strings
func ReadStdioByAppID(appID uuid.UUID) (string, string, error) {
	base := path.Join(basePath(logsDir), appID.String())

	ensurePath(base)

	stdoutPath := path.Join(base, "stdout.log")
	stderrPath := path.Join(base, "stderr.log")

	stdout, err := os.ReadFile(stdoutPath)
	if err != nil {
		// ignore not exist errors
		if os.IsNotExist(err) {
			return "", "", nil
		}

		return "", "", err
	}

	stderr, err := os.ReadFile(stderrPath)
	if err != nil {
		// ignore not exist errors
		if os.IsNotExist(err) {
			return "", "", nil
		}

		return "", "", err
	}

	return string(stdout), string(stderr), nil
}

func RemoveEmptyStrings(a *[]string) {
	var r []string
	for _, str := range *a {
		if str != "" {
			r = append(r, str)
		}
	}
	*a = r
}
