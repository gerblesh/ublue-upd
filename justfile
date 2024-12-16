set shell := ["bash", "-uc"]
export UBLUE_ROOT := env_var_or_default("UBLUE_ROOT", "/app/output")
export TARGET := "uupd"
export SOURCE_DIR := UBLUE_ROOT + "/" + TARGET
export RPMBUILD := UBLUE_ROOT + "/rpmbuild"

default:
	just --list

build:
	go build -o output/uupd

run: build
	sudo ./output/uupd

spec: output
	rpkg spec --outdir "$PWD/output"

build-rpm:
	rpkg local --outdir "$PWD/output"

builddep:
	dnf builddep -y output/uupd.spec

container-install-deps:
	#!/usr/bin/env bash
	set -eou pipefail
	dnf install                       \
		--disablerepo='*'             \
		--enablerepo='fedora,updates' \
		--setopt install_weak_deps=0  \
		--nodocs                      \
		--assumeyes                   \
		'dnf-command(builddep)'       \
		rpkg                          \
		rpm-build                     \
		git

# Used internally by build containers
container-rpm-build: container-install-deps spec builddep build-rpm
	#!/usr/bin/env bash
	set -eou pipefail

	# clean up files
	for RPM in ${UBLUE_ROOT}/*/*.rpm; do
		NAME="$(rpm -q $RPM --queryformat='%{NAME}')"
		mkdir -p "${UBLUE_ROOT}/ublue-os/rpms/"
		cp "${RPM}" "${UBLUE_ROOT}/ublue-os/rpms/$(rpm -q "${RPM}" --queryformat='%{NAME}.rpm')"
	done

output:
	mkdir -p output

dnf-install:
	dnf install -y "output/noarch/*.rpm"

container-build:
	podman build . -t test-container -f Containerfile

container-test:
	#!/usr/bin/env bash
	set -eou pipefail

	podman run -d --replace --name uupd-test --security-opt label=disable --device /dev/fuse:rw --privileged --systemd true test-container 
	while [[ "$(podman exec uupd-test systemctl is-system-running)" != "running" && "$(podman exec uupd-test systemctl is-system-running)" != "degraded" ]]; do
		echo "Waiting for systemd to finish booting..."
		sleep 1
	done
	podman exec -t uupd-test systemd-run --machine 0@ --pipe --quiet /usr/bin/uupd --dry-run
	podman rm -f uupd-test
clean:
	rm -rf "$UBLUE_ROOT"

lint:
	golangci-lint run

release:
	goreleaser

test directory="":
	#!/usr/bin/env bash
	if [ "{{directory}}" != "" ] ; then
		go test -v -cover ./{{directory}}/...
	else
		go test -v -cover ./...
	fi

test-coverage directory="":
	#!/usr/bin/env bash
	t="/tmp/go-cover.$$.tmp"

	if [ "{{directory}}" != "" ] ; then
		go test -v -coverprofile=$t ./{{directory}}/... $@ && go tool cover -html=$t && unlink $t
	else
		go test -v -coverprofile=$t ./... $@ && go tool cover -html=$t && unlink $t
	fi
