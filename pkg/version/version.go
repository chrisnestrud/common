package version

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	UnknownPackage = "Unknown"
)

// Note: This function is copied from containers/podman libpod/util.go
// Please see https://github.com/containers/common/pull/1460
func queryPackageVersion(cmdArg ...string) string {
	_, err := os.Stat(cmdArg[0])
	if err != nil {
		return ""
	}
	output := UnknownPackage
	if 1 < len(cmdArg) {
		cmd := exec.Command(cmdArg[0], cmdArg[1:]...)
		if outp, err := cmd.Output(); err == nil {
			output = string(outp)
			deb := false
			if cmdArg[0] == "/usr/bin/dlocate" {
				// can return multiple matches
				l := strings.Split(output, "\n")
				output = l[0]
				deb = true
			} else if cmdArg[0] == "/usr/bin/dpkg" {
				deb = true
			}
			if deb {
				r := strings.Split(output, ": ")
				queryFormat := `${Package}_${Version}_${Architecture}`
				cmd = exec.Command("/usr/bin/dpkg-query", "-f", queryFormat, "-W", r[0])
				if outp, err := cmd.Output(); err == nil {
					output = string(outp)
				}
			}
		}
		if cmdArg[0] == "/sbin/apk" {
			prefix := cmdArg[len(cmdArg)-1] + " is owned by "
			output = strings.Replace(output, prefix, "", 1)
		}
	}
	return strings.Trim(output, "\n")
}

// Note: This function is copied from containers/podman libpod/util.go
// Please see https://github.com/containers/common/pull/1460
func Package(program string) string { // program is full path
	_, err := os.Stat(program)
	if err != nil {
		return UnknownPackage
	}

	type Packager struct {
		Format  string
		Command []string
	}
	packagers := []Packager{
		{"rpm", []string{"/usr/bin/rpm", "-q", "-f"}},
		{"deb", []string{"/usr/bin/dlocate", "-F"}},             // Debian, Ubuntu (quick)
		{"deb", []string{"/usr/bin/dpkg", "-S"}},                // Debian, Ubuntu (slow)
		{"pacman", []string{"/usr/bin/pacman", "-Qo"}},          // Arch
		{"gentoo", []string{"/usr/bin/qfile", "-qv"}},           // Gentoo (quick)
		{"gentoo", []string{"/usr/bin/equery", "b"}},            // Gentoo (slow)
		{"apk", []string{"/sbin/apk", "info", "-W"}},            // Alpine
		{"pkg", []string{"/usr/local/sbin/pkg", "which", "-q"}}, // FreeBSD
	}

	lastformat := ""
	for _, packager := range packagers {
		if packager.Format == lastformat {
			continue
		}
		cmd := packager.Command
		cmd = append(cmd, program)
		if out := queryPackageVersion(cmd...); out != UnknownPackage {
			if out == "" {
				continue
			}
			return out
		}
		lastformat = packager.Format
	}
	return UnknownPackage
}

// Note: This function is copied from containers/podman libpod/util.go
// Please see https://github.com/containers/common/pull/1460
func Program(name string) (string, error) {
	return program(name, false)
}

func ProgramDnsname(name string) (string, error) {
	return program(name, true)
}

func program(program string, dnsname bool) (string, error) {
	cmd := exec.Command(program, "--version")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("`%v --version` failed: %v %v (%v)", program, stderr.String(), stdout.String(), err)
	}

	output := strings.TrimSuffix(stdout.String(), "\n")
	// dnsname --version returns the information to stderr
	if dnsname {
		output = strings.TrimSuffix(stderr.String(), "\n")
	}

	return output, nil
}
