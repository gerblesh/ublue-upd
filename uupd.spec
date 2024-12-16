Name:           uupd
Version:        1.0
Release:        1%{?dist}
Summary:       Centralized update service/checker made for Universal Blue
Vendor:        ublue-os
URL:           https://github.com/%{vendor}/%{name}
# Detailed information about the source Git repository and the source commit
# for the created rpm package
VCS:           {{{ git_dir_vcs }}}
# git_dir_pack macro places the repository content (the source files) into a tarball
# and returns its filename. The tarball will be used to build the rpm.
Source:        {{{ git_dir_pack }}}
License:        Apache-2.0

BuildRequires:  golang
BuildRequires:  systemd-rpm-macros
Requires:       bootc
Requires:       distrobox
Requires:       flatpak
Requires:       libnotify
Requires:       systemd
Provides:       %{name} = %{version}

%description
A simple updater for Universal Blue systems

%global debug_package %{nil}

%prep
{{{ git_dir_setup_macro }}}

%build
go build -v -o %{name}

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 644 %{name}.service %{buildroot}%{_unitdir}/%{name}.service
install -Dpm 644 %{name}.timer %{buildroot}%{_unitdir}/%{name}.timer
install -Dpm 644 %{name}.rules %{buildroot}%{_sysconfdir}/polkit-1/rules.d/%{name}.rules

%check
go test -v ./...

%post
%systemd_post %{name}.timer

%preun
%systemd_preun %{name}.timer

%files
%{_bindir}/%{name}
%{_unitdir}/%{name}.service
%{_unitdir}/%{name}.timer
%config(noreplace) %{_sysconfdir}/polkit-1/rules.d/%{name}.rules

%changelog
%autochangelog
